package proxy

import (
	"fmt"
	"net"
	"sync"
	"sync/atomic"
	"time"

	"github.com/sirupsen/logrus"
)

// ClusterNode 集群节点
type ClusterNode struct {
	ID          string
	Address     string
	Weight      int           // 权重（用于负载均衡）
	Healthy     bool          // 健康状态
	LastCheck   time.Time     // 最后检查时间
	Connections int64         // 当前连接数
	Latency     time.Duration // 延迟
	mu          sync.RWMutex
}

// ClusterManager 集群管理器
type ClusterManager struct {
	nodes       map[string]*ClusterNode
	mu          sync.RWMutex
	healthCheck func(node *ClusterNode) bool
	lbStrategy  string // "round_robin", "weighted", "least_connections"
	currentIdx  int64  // 当前轮询索引（原子操作）
}

// ClusterConfig 集群配置
type ClusterConfig struct {
	Nodes      []ClusterNodeConfig
	Strategy   string        // 负载均衡策略
	HealthCheckInterval time.Duration
}

// ClusterNodeConfig 集群节点配置
type ClusterNodeConfig struct {
	ID      string
	Address string
	Weight  int
}

// NewClusterManager 创建集群管理器
func NewClusterManager(config ClusterConfig) *ClusterManager {
	manager := &ClusterManager{
		nodes:      make(map[string]*ClusterNode),
		lbStrategy: config.Strategy,
		healthCheck: func(node *ClusterNode) bool {
			// 默认健康检查：尝试连接
			conn, err := net.DialTimeout("tcp", node.Address, 3*time.Second)
			if err != nil {
				return false
			}
			conn.Close()
			return true
		},
	}

	// 初始化节点
	for _, nodeConfig := range config.Nodes {
		node := &ClusterNode{
			ID:      nodeConfig.ID,
			Address: nodeConfig.Address,
			Weight:  nodeConfig.Weight,
			Healthy: true,
		}
		manager.nodes[node.ID] = node
	}

	// 启动健康检查
	if config.HealthCheckInterval > 0 {
		go manager.startHealthCheck(config.HealthCheckInterval)
	}

	return manager
}

// SelectNode 选择节点（负载均衡）
func (cm *ClusterManager) SelectNode() (*ClusterNode, error) {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	healthyNodes := make([]*ClusterNode, 0)
	for _, node := range cm.nodes {
		node.mu.RLock()
		if node.Healthy {
			healthyNodes = append(healthyNodes, node)
		}
		node.mu.RUnlock()
	}

	if len(healthyNodes) == 0 {
		return nil, fmt.Errorf("no healthy nodes available")
	}

	switch cm.lbStrategy {
	case "least_connections":
		return cm.selectLeastConnections(healthyNodes), nil
	case "weighted":
		return cm.selectWeighted(healthyNodes), nil
	default: // round_robin
		return cm.selectRoundRobin(healthyNodes), nil
	}
}

// selectRoundRobin 轮询选择
func (cm *ClusterManager) selectRoundRobin(nodes []*ClusterNode) *ClusterNode {
	idx := atomic.AddInt64(&cm.currentIdx, 1)
	return nodes[int(idx)%len(nodes)]
}

// selectLeastConnections 选择连接数最少的节点
func (cm *ClusterManager) selectLeastConnections(nodes []*ClusterNode) *ClusterNode {
	var bestNode *ClusterNode
	var minConnections int64 = -1

	for _, node := range nodes {
		node.mu.RLock()
		conns := atomic.LoadInt64(&node.Connections)
		node.mu.RUnlock()

		if minConnections == -1 || conns < minConnections {
			minConnections = conns
			bestNode = node
		}
	}

	return bestNode
}

// selectWeighted 加权选择
func (cm *ClusterManager) selectWeighted(nodes []*ClusterNode) *ClusterNode {
	totalWeight := 0
	for _, node := range nodes {
		totalWeight += node.Weight
	}

	if totalWeight == 0 {
		return cm.selectRoundRobin(nodes)
	}

	// 简单的加权轮询
	idx := atomic.AddInt64(&cm.currentIdx, 1)
	current := int(idx) % totalWeight

	for _, node := range nodes {
		current -= node.Weight
		if current < 0 {
			return node
		}
	}

	return nodes[0]
}

// startHealthCheck 启动健康检查
func (cm *ClusterManager) startHealthCheck(interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for range ticker.C {
		cm.mu.RLock()
		nodes := make([]*ClusterNode, 0, len(cm.nodes))
		for _, node := range cm.nodes {
			nodes = append(nodes, node)
		}
		cm.mu.RUnlock()

		for _, node := range nodes {
			healthy := cm.healthCheck(node)
			
			node.mu.Lock()
			wasHealthy := node.Healthy
			node.Healthy = healthy
			node.LastCheck = time.Now()
			node.mu.Unlock()

			if wasHealthy != healthy {
				if healthy {
					logrus.Infof("Cluster node %s (%s) recovered", node.ID, node.Address)
				} else {
					logrus.Warnf("Cluster node %s (%s) is unhealthy", node.ID, node.Address)
				}
			}
		}
	}
}

// OnNodeConnectionStart 节点连接开始
func (cm *ClusterManager) OnNodeConnectionStart(nodeID string) {
	cm.mu.RLock()
	node, ok := cm.nodes[nodeID]
	cm.mu.RUnlock()

	if ok {
		atomic.AddInt64(&node.Connections, 1)
	}
}

// OnNodeConnectionEnd 节点连接结束
func (cm *ClusterManager) OnNodeConnectionEnd(nodeID string) {
	cm.mu.RLock()
	node, ok := cm.nodes[nodeID]
	cm.mu.RUnlock()

	if ok {
		current := atomic.LoadInt64(&node.Connections)
		if current > 0 {
			atomic.AddInt64(&node.Connections, -1)
		}
	}
}

// GetHealthyNodes 获取健康节点列表
func (cm *ClusterManager) GetHealthyNodes() []*ClusterNode {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	healthy := make([]*ClusterNode, 0)
	for _, node := range cm.nodes {
		node.mu.RLock()
		if node.Healthy {
			healthy = append(healthy, node)
		}
		node.mu.RUnlock()
	}

	return healthy
}

// Failover 故障转移（选择下一个健康节点）
func (cm *ClusterManager) Failover(currentNodeID string) (*ClusterNode, error) {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	// 标记当前节点为不健康
	if node, ok := cm.nodes[currentNodeID]; ok {
		node.mu.Lock()
		node.Healthy = false
		node.mu.Unlock()
	}

	// 选择新的健康节点
	healthyNodes := make([]*ClusterNode, 0)
	for _, node := range cm.nodes {
		node.mu.RLock()
		if node.Healthy && node.ID != currentNodeID {
			healthyNodes = append(healthyNodes, node)
		}
		node.mu.RUnlock()
	}

	if len(healthyNodes) == 0 {
		return nil, fmt.Errorf("no healthy nodes available for failover")
	}

	node, err := cm.SelectNode()
	if err != nil {
		return nil, err
	}
	return node, nil
}

