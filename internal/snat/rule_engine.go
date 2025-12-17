package snat

import (
	"fmt"
	"net"
	"regexp"
	"strings"
	"sync"

	"github.com/sirupsen/logrus"
)

// Rule 路由规则
type Rule struct {
	Name        string   `yaml:"name"`
	Priority    int      `yaml:"priority"`    // 优先级（数字越大优先级越高）
	MatchDomain []string `yaml:"match_domain"` // 匹配域名（支持通配符）
	MatchIP     []string `yaml:"match_ip"`     // 匹配IP（CIDR格式）
	MatchPort   []int    `yaml:"match_port"`   // 匹配端口
	TargetIP    string   `yaml:"target_ip"`   // 目标出口IP
	Action      string   `yaml:"action"`       // 动作：use_ip, skip, reject
	Enabled     bool     `yaml:"enabled"`
}

// RuleEngine 规则引擎
type RuleEngine struct {
	rules    []*Rule
	mu       sync.RWMutex
	compiled map[*Rule]*CompiledRule
}

// CompiledRule 编译后的规则（用于快速匹配）
type CompiledRule struct {
	DomainPatterns []*regexp.Regexp
	IPNets         []*net.IPNet
	Ports          map[int]bool
}

// NewRuleEngine 创建规则引擎
func NewRuleEngine() *RuleEngine {
	return &RuleEngine{
		rules:    make([]*Rule, 0),
		compiled: make(map[*Rule]*CompiledRule),
	}
}

// AddRule 添加规则
func (re *RuleEngine) AddRule(rule *Rule) error {
	// 验证规则
	if err := re.validateRule(rule); err != nil {
		return err
	}

	// 编译规则
	compiled, err := re.compileRule(rule)
	if err != nil {
		return fmt.Errorf("failed to compile rule %s: %w", rule.Name, err)
	}

	re.mu.Lock()
	defer re.mu.Unlock()

	// 按优先级排序插入
	inserted := false
	for i, r := range re.rules {
		if rule.Priority > r.Priority {
			re.rules = append(re.rules[:i], append([]*Rule{rule}, re.rules[i:]...)...)
			inserted = true
			break
		}
	}
	if !inserted {
		re.rules = append(re.rules, rule)
	}

	re.compiled[rule] = compiled
	logrus.Infof("Rule added: %s (priority: %d)", rule.Name, rule.Priority)
	return nil
}

// RemoveRule 移除规则
func (re *RuleEngine) RemoveRule(name string) error {
	re.mu.Lock()
	defer re.mu.Unlock()

	for i, rule := range re.rules {
		if rule.Name == name {
			re.rules = append(re.rules[:i], re.rules[i+1:]...)
			delete(re.compiled, rule)
			logrus.Infof("Rule removed: %s", name)
			return nil
		}
	}

	return fmt.Errorf("rule %s not found", name)
}

// MatchRule 匹配规则
func (re *RuleEngine) MatchRule(targetAddr string, targetPort int) (*Rule, error) {
	host, _, err := net.SplitHostPort(targetAddr)
	if err != nil {
		host = targetAddr
	}

	re.mu.RLock()
	defer re.mu.RUnlock()

	// 按优先级顺序匹配
	for _, rule := range re.rules {
		if !rule.Enabled {
			continue
		}

		compiled := re.compiled[rule]
		if compiled == nil {
			continue
		}

		// 匹配域名
		if len(compiled.DomainPatterns) > 0 {
			matched := false
			for _, pattern := range compiled.DomainPatterns {
				if pattern.MatchString(host) {
					matched = true
					break
				}
			}
			if !matched {
				continue
			}
		}

		// 匹配IP
		if len(compiled.IPNets) > 0 {
			ip := net.ParseIP(host)
			if ip != nil {
				matched := false
				for _, ipNet := range compiled.IPNets {
					if ipNet.Contains(ip) {
						matched = true
						break
					}
				}
				if !matched {
					continue
				}
			} else {
				continue // IP匹配规则但host不是IP
			}
		}

		// 匹配端口
		if len(compiled.Ports) > 0 {
			if !compiled.Ports[targetPort] {
				continue
			}
		}

		// 规则匹配成功
		logrus.Debugf("Rule matched: %s for %s:%d", rule.Name, host, targetPort)
		return rule, nil
	}

	return nil, nil // 没有匹配的规则
}

// validateRule 验证规则
func (re *RuleEngine) validateRule(rule *Rule) error {
	if rule.Name == "" {
		return fmt.Errorf("rule name is required")
	}

	if rule.Action == "use_ip" && rule.TargetIP == "" {
		return fmt.Errorf("target_ip is required when action is use_ip")
	}

	validActions := map[string]bool{
		"use_ip": true,
		"skip":   true,
		"reject": true,
	}
	if !validActions[rule.Action] {
		return fmt.Errorf("invalid action: %s", rule.Action)
	}

	return nil
}

// compileRule 编译规则
func (re *RuleEngine) compileRule(rule *Rule) (*CompiledRule, error) {
	compiled := &CompiledRule{
		Ports: make(map[int]bool),
	}

	// 编译域名模式（支持通配符）
	if len(rule.MatchDomain) > 0 {
		compiled.DomainPatterns = make([]*regexp.Regexp, 0, len(rule.MatchDomain))
		for _, pattern := range rule.MatchDomain {
			// 将通配符转换为正则表达式
			regexPattern := strings.ReplaceAll(pattern, ".", "\\.")
			regexPattern = strings.ReplaceAll(regexPattern, "*", ".*")
			regexPattern = "^" + regexPattern + "$"
			
			re, err := regexp.Compile(regexPattern)
			if err != nil {
				return nil, fmt.Errorf("invalid domain pattern %s: %w", pattern, err)
			}
			compiled.DomainPatterns = append(compiled.DomainPatterns, re)
		}
	}

	// 编译IP网络
	if len(rule.MatchIP) > 0 {
		compiled.IPNets = make([]*net.IPNet, 0, len(rule.MatchIP))
		for _, ipStr := range rule.MatchIP {
			_, ipNet, err := net.ParseCIDR(ipStr)
			if err != nil {
				// 尝试作为单个IP解析
				ip := net.ParseIP(ipStr)
				if ip == nil {
					return nil, fmt.Errorf("invalid IP or CIDR: %s", ipStr)
				}
				// 创建/32或/128网络
				mask := net.CIDRMask(32, 32)
				if ip.To4() == nil {
					mask = net.CIDRMask(128, 128)
				}
				ipNet = &net.IPNet{IP: ip, Mask: mask}
			}
			compiled.IPNets = append(compiled.IPNets, ipNet)
		}
	}

	// 编译端口列表
	for _, port := range rule.MatchPort {
		compiled.Ports[port] = true
	}

	return compiled, nil
}

// GetRules 获取所有规则
func (re *RuleEngine) GetRules() []*Rule {
	re.mu.RLock()
	defer re.mu.RUnlock()

	rules := make([]*Rule, len(re.rules))
	copy(rules, re.rules)
	return rules
}

// RuleBasedSelector 基于规则的选择器
type RuleBasedSelector struct {
	baseSelector IPSelector
	ruleEngine   *RuleEngine
	exitIPs      []string
}

// NewRuleBasedSelector 创建基于规则的选择器
func NewRuleBasedSelector(baseSelector IPSelector, ruleEngine *RuleEngine, exitIPs []string) *RuleBasedSelector {
	return &RuleBasedSelector{
		baseSelector: baseSelector,
		ruleEngine:   ruleEngine,
		exitIPs:      exitIPs,
	}
}

// SelectIP 选择IP（基于规则）
func (r *RuleBasedSelector) SelectIP(targetAddr string, targetPort int) (net.IP, error) {
	// 匹配规则
	rule, err := r.ruleEngine.MatchRule(targetAddr, targetPort)
	if err != nil {
		return nil, err
	}

	if rule != nil {
		switch rule.Action {
		case "use_ip":
			// 使用规则指定的IP
			ip := net.ParseIP(rule.TargetIP)
			if ip == nil {
				return nil, fmt.Errorf("invalid target IP in rule: %s", rule.TargetIP)
			}
			logrus.Debugf("Using rule-specified IP %s for %s:%d", rule.TargetIP, targetAddr, targetPort)
			return ip, nil
		case "skip":
			// 跳过规则，使用基础选择器
			logrus.Debugf("Rule matched but skipped, using base selector")
			return r.baseSelector.SelectIP(targetAddr, targetPort)
		case "reject":
			// 拒绝连接
			return nil, fmt.Errorf("connection rejected by rule: %s", rule.Name)
		}
	}

	// 没有匹配的规则，使用基础选择器
	return r.baseSelector.SelectIP(targetAddr, targetPort)
}

