package proxy

import (
	"fmt"
	"net"
	"regexp"
	"strings"
	"sync"

	"github.com/sirupsen/logrus"
)

// Rule 规则
type Rule struct {
	ID          string
	Name        string
	Priority    int           // 优先级（数字越大优先级越高）
	Type        string        // "domain", "ip", "cidr", "regex"
	Pattern     string        // 匹配模式
	Action      string        // "use_ip", "block", "redirect"
	TargetIP    string        // 目标IP（当action为use_ip时）
	Enabled     bool
	Description string
}

// RuleEngine 规则引擎
type RuleEngine struct {
	rules []*Rule
	mu    sync.RWMutex
}

// NewRuleEngine 创建规则引擎
func NewRuleEngine() *RuleEngine {
	return &RuleEngine{
		rules: make([]*Rule, 0),
	}
}

// AddRule 添加规则
func (re *RuleEngine) AddRule(rule *Rule) error {
	// 验证规则
	if err := re.validateRule(rule); err != nil {
		return err
	}

	re.mu.Lock()
	defer re.mu.Unlock()

	// 检查ID是否已存在
	for _, r := range re.rules {
		if r.ID == rule.ID {
			return fmt.Errorf("rule with ID %s already exists", rule.ID)
		}
	}

	re.rules = append(re.rules, rule)
	
	// 按优先级排序
	re.sortRules()

	logrus.Infof("Rule added: %s (priority: %d, type: %s)", rule.Name, rule.Priority, rule.Type)
	return nil
}

// RemoveRule 删除规则
func (re *RuleEngine) RemoveRule(ruleID string) error {
	re.mu.Lock()
	defer re.mu.Unlock()

	for i, rule := range re.rules {
		if rule.ID == ruleID {
			re.rules = append(re.rules[:i], re.rules[i+1:]...)
			logrus.Infof("Rule removed: %s", ruleID)
			return nil
		}
	}

	return fmt.Errorf("rule %s not found", ruleID)
}

// UpdateRule 更新规则
func (re *RuleEngine) UpdateRule(ruleID string, newRule *Rule) error {
	if err := re.validateRule(newRule); err != nil {
		return err
	}

	re.mu.Lock()
	defer re.mu.Unlock()

	for i, rule := range re.rules {
		if rule.ID == ruleID {
			re.rules[i] = newRule
			re.sortRules()
			logrus.Infof("Rule updated: %s", ruleID)
			return nil
		}
	}

	return fmt.Errorf("rule %s not found", ruleID)
}

// Match 匹配规则
func (re *RuleEngine) Match(targetAddr string) (*Rule, error) {
	re.mu.RLock()
	defer re.mu.RUnlock()

	host, port, err := net.SplitHostPort(targetAddr)
	if err != nil {
		host = targetAddr
		port = ""
	}

	// 按优先级从高到低匹配
	for _, rule := range re.rules {
		if !rule.Enabled {
			continue
		}

		if re.matchRule(rule, host, port) {
			logrus.Debugf("Rule matched: %s for target %s", rule.Name, targetAddr)
			return rule, nil
		}
	}

	return nil, nil // 没有匹配的规则
}

// matchRule 匹配单个规则
func (re *RuleEngine) matchRule(rule *Rule, host, port string) bool {
	switch rule.Type {
	case "domain":
		return re.matchDomain(rule.Pattern, host)
	case "ip":
		return re.matchIP(rule.Pattern, host)
	case "cidr":
		return re.matchCIDR(rule.Pattern, host)
	case "regex":
		return re.matchRegex(rule.Pattern, host)
	default:
		return false
	}
}

// matchDomain 匹配域名
func (re *RuleEngine) matchDomain(pattern, host string) bool {
	// 精确匹配
	if pattern == host {
		return true
	}

	// 后缀匹配（如 *.example.com）
	if strings.HasPrefix(pattern, "*.") {
		suffix := pattern[2:]
		return strings.HasSuffix(host, suffix)
	}

	// 前缀匹配
	if strings.HasSuffix(pattern, "*") {
		prefix := pattern[:len(pattern)-1]
		return strings.HasPrefix(host, prefix)
	}

	return false
}

// matchIP 匹配IP地址
func (re *RuleEngine) matchIP(pattern, host string) bool {
	return pattern == host
}

// matchCIDR 匹配CIDR
func (re *RuleEngine) matchCIDR(pattern, host string) bool {
	_, ipNet, err := net.ParseCIDR(pattern)
	if err != nil {
		return false
	}

	ip := net.ParseIP(host)
	if ip == nil {
		return false
	}

	return ipNet.Contains(ip)
}

// matchRegex 匹配正则表达式
func (re *RuleEngine) matchRegex(pattern, host string) bool {
	matched, err := regexp.MatchString(pattern, host)
	if err != nil {
		logrus.Warnf("Invalid regex pattern %s: %v", pattern, err)
		return false
	}
	return matched
}

// validateRule 验证规则
func (re *RuleEngine) validateRule(rule *Rule) error {
	if rule.ID == "" {
		return fmt.Errorf("rule ID is required")
	}

	if rule.Type == "" {
		return fmt.Errorf("rule type is required")
	}

	if rule.Pattern == "" {
		return fmt.Errorf("rule pattern is required")
	}

	if rule.Action == "" {
		return fmt.Errorf("rule action is required")
	}

	// 验证类型
	validTypes := map[string]bool{
		"domain": true,
		"ip":     true,
		"cidr":   true,
		"regex":  true,
	}
	if !validTypes[rule.Type] {
		return fmt.Errorf("invalid rule type: %s", rule.Type)
	}

	// 验证动作
	validActions := map[string]bool{
		"use_ip":   true,
		"block":    true,
		"redirect": true,
	}
	if !validActions[rule.Action] {
		return fmt.Errorf("invalid rule action: %s", rule.Action)
	}

	// 如果动作是use_ip，验证目标IP
	if rule.Action == "use_ip" && rule.TargetIP == "" {
		return fmt.Errorf("target IP is required for use_ip action")
	}

	if rule.Action == "use_ip" {
		if ip := net.ParseIP(rule.TargetIP); ip == nil {
			return fmt.Errorf("invalid target IP: %s", rule.TargetIP)
		}
	}

	return nil
}

// sortRules 按优先级排序规则
func (re *RuleEngine) sortRules() {
	// 简单的冒泡排序（按优先级从高到低）
	for i := 0; i < len(re.rules)-1; i++ {
		for j := i + 1; j < len(re.rules); j++ {
			if re.rules[i].Priority < re.rules[j].Priority {
				re.rules[i], re.rules[j] = re.rules[j], re.rules[i]
			}
		}
	}
}

// ListRules 列出所有规则
func (re *RuleEngine) ListRules() []*Rule {
	re.mu.RLock()
	defer re.mu.RUnlock()

	rules := make([]*Rule, len(re.rules))
	copy(rules, re.rules)
	return rules
}

// GetRule 获取规则
func (re *RuleEngine) GetRule(ruleID string) (*Rule, error) {
	re.mu.RLock()
	defer re.mu.RUnlock()

	for _, rule := range re.rules {
		if rule.ID == ruleID {
			return rule, nil
		}
	}

	return nil, fmt.Errorf("rule %s not found", ruleID)
}

// EnableRule 启用规则
func (re *RuleEngine) EnableRule(ruleID string) error {
	re.mu.Lock()
	defer re.mu.Unlock()

	for _, rule := range re.rules {
		if rule.ID == ruleID {
			rule.Enabled = true
			logrus.Infof("Rule enabled: %s", ruleID)
			return nil
		}
	}

	return fmt.Errorf("rule %s not found", ruleID)
}

// DisableRule 禁用规则
func (re *RuleEngine) DisableRule(ruleID string) error {
	re.mu.Lock()
	defer re.mu.Unlock()

	for _, rule := range re.rules {
		if rule.ID == ruleID {
			rule.Enabled = false
			logrus.Infof("Rule disabled: %s", ruleID)
			return nil
		}
	}

	return fmt.Errorf("rule %s not found", ruleID)
}



