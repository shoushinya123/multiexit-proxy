package cluster

import (
	"context"
	"fmt"
	"net"
	"sync"
	"sync/atomic"
	"time"

	"github.com/sirupsen/logrus"
)

// Node 集群节点
type Node struct {
	ID          string
	Address     string
	Status      string // "active", "inactive", "failed"
	LastSeen    time.Time
	Connections int64
	mu          sync.RWMutex
}

// ClusterManager 集群管理器
type ClusterManager struct {
	nodes          map[string]*Node
	mu             sync.RWMutex
	healthInterval time.Duration
	loadBalancer   LoadBalancer
}

// LoadBalancer 负载均衡器接口
type LoadBalancer interface {
	SelectNode(nodes []*Node) *Node
}

// RoundRobinLoadBalancer 轮询负载均衡器
type RoundRobinLoadBalancer struct {
	current int64
}

// NewRoundRobinLoadBalancer 创建轮询负载均衡器
func NewRoundRobinLoadBalancer() *RoundRobinLoadBalancer {
	return &RoundRobinLoadBalancer{}
}

// SelectNode 选择节点（轮询）
func (rr *RoundRobinLoadBalancer) SelectNode(nodes []*Node) *Node {
	if len(nodes) == 0 {
		return nil
	}

	activeNodes := make([]*Node, 0)
	for _, node := range nodes {
		node.mu.RLock()
		if node.Status == "active" {
			activeNodes = append(activeNodes, node)
		}
		node.mu.RUnlock()
	}

	if len(activeNodes) == 0 {
		return nil
	}

	index := atomic.AddInt64(&rr.current, 1) % int64(len(activeNodes))
	return activeNodes[index]
}

// LeastConnectionsLoadBalancer 最少连接负载均衡器
type LeastConnectionsLoadBalancer struct{}

// NewLeastConnectionsLoadBalancer 创建最少连接负载均衡器
func NewLeastConnectionsLoadBalancer() *LeastConnectionsLoadBalancer {
	return &LeastConnectionsLoadBalancer{}
}

// SelectNode 选择节点（最少连接）
func (lc *LeastConnectionsLoadBalancer) SelectNode(nodes []*Node) *Node {
	var bestNode *Node
	var minConnections int64 = -1

	for _, node := range nodes {
		node.mu.RLock()
		if node.Status == "active" {
			conns := atomic.LoadInt64(&node.Connections)
			if minConnections == -1 || conns < minConnections {
				minConnections = conns
				bestNode = node
			}
		}
		node.mu.RUnlock()
	}

	return bestNode
}

// NewClusterManager 创建集群管理器
func NewClusterManager(healthInterval time.Duration, lbType string) *ClusterManager {
	var loadBalancer LoadBalancer
	switch lbType {
	case "least_connections":
		loadBalancer = NewLeastConnectionsLoadBalancer()
	default:
		loadBalancer = NewRoundRobinLoadBalancer()
	}

	return &ClusterManager{
		nodes:          make(map[string]*Node),
		healthInterval: healthInterval,
		loadBalancer:  loadBalancer,
	}
}

// AddNode 添加节点
func (cm *ClusterManager) AddNode(id, address string) error {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	if _, exists := cm.nodes[id]; exists {
		return fmt.Errorf("node %s already exists", id)
	}

	node := &Node{
		ID:       id,
		Address:  address,
		Status:   "active",
		LastSeen: time.Now(),
	}

	cm.nodes[id] = node
	logrus.Infof("Node added to cluster: %s (%s)", id, address)
	return nil
}

// RemoveNode 移除节点
func (cm *ClusterManager) RemoveNode(id string) error {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	if _, exists := cm.nodes[id]; !exists {
		return fmt.Errorf("node %s not found", id)
	}

	delete(cm.nodes, id)
	logrus.Infof("Node removed from cluster: %s", id)
	return nil
}

// SelectNode 选择节点（负载均衡）
func (cm *ClusterManager) SelectNode() (*Node, error) {
	cm.mu.RLock()
	nodes := make([]*Node, 0, len(cm.nodes))
	for _, node := range cm.nodes {
		nodes = append(nodes, node)
	}
	cm.mu.RUnlock()

	node := cm.loadBalancer.SelectNode(nodes)
	if node == nil {
		return nil, fmt.Errorf("no active nodes available")
	}

	return node, nil
}

// HealthCheck 健康检查
func (cm *ClusterManager) HealthCheck(ctx context.Context) {
	ticker := time.NewTicker(cm.healthInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			cm.checkAllNodes()
		}
	}
}

// checkAllNodes 检查所有节点
func (cm *ClusterManager) checkAllNodes() {
	cm.mu.RLock()
	nodes := make([]*Node, 0, len(cm.nodes))
	for _, node := range cm.nodes {
		nodes = append(nodes, node)
	}
	cm.mu.RUnlock()

	for _, node := range nodes {
		go cm.checkNode(node)
	}
}

// checkNode 检查单个节点
func (cm *ClusterManager) checkNode(node *Node) {
	// 尝试连接到节点
	conn, err := net.DialTimeout("tcp", node.Address, 3*time.Second)
	if err != nil {
		node.mu.Lock()
		if node.Status == "active" {
			node.Status = "failed"
			logrus.Warnf("Node %s (%s) health check failed: %v", node.ID, node.Address, err)
		}
		node.mu.Unlock()
		return
	}
	conn.Close()

	node.mu.Lock()
	if node.Status != "active" {
		logrus.Infof("Node %s (%s) recovered", node.ID, node.Address)
	}
	node.Status = "active"
	node.LastSeen = time.Now()
	node.mu.Unlock()
}

// OnNodeConnectionStart 节点连接开始
func (cm *ClusterManager) OnNodeConnectionStart(nodeID string) {
	cm.mu.RLock()
	node, exists := cm.nodes[nodeID]
	cm.mu.RUnlock()

	if exists {
		atomic.AddInt64(&node.Connections, 1)
	}
}

// OnNodeConnectionEnd 节点连接结束
func (cm *ClusterManager) OnNodeConnectionEnd(nodeID string) {
	cm.mu.RLock()
	node, exists := cm.nodes[nodeID]
	cm.mu.RUnlock()

	if exists {
		current := atomic.LoadInt64(&node.Connections)
		if current > 0 {
			atomic.AddInt64(&node.Connections, -1)
		}
	}
}

// GetNodes 获取所有节点
func (cm *ClusterManager) GetNodes() []*Node {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	nodes := make([]*Node, 0, len(cm.nodes))
	for _, node := range cm.nodes {
		node.mu.RLock()
		nodes = append(nodes, &Node{
			ID:          node.ID,
			Address:     node.Address,
			Status:      node.Status,
			LastSeen:    node.LastSeen,
			Connections: atomic.LoadInt64(&node.Connections),
		})
		node.mu.RUnlock()
	}

	return nodes
}

// GetActiveNodes 获取活跃节点
func (cm *ClusterManager) GetActiveNodes() []*Node {
	allNodes := cm.GetNodes()
	activeNodes := make([]*Node, 0)
	for _, node := range allNodes {
		if node.Status == "active" {
			activeNodes = append(activeNodes, node)
		}
	}
	return activeNodes
}

