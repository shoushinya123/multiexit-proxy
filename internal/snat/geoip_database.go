package snat

import (
	"encoding/json"
	"fmt"
	"net"
	"os"
	"sync"
	"time"

	"github.com/sirupsen/logrus"
)

// GeoIPDatabase GeoIP数据库接口
type GeoIPDatabase interface {
	Lookup(ip net.IP) (*GeoLocation, error)
	LoadFromFile(filePath string) error
	IsLoaded() bool
}

// MaxMindGeoIP MaxMind GeoIP数据库实现（简化版，支持JSON格式）
type MaxMindGeoIP struct {
	data   map[string]*GeoLocation
	mu     sync.RWMutex
	loaded bool
}

// NewMaxMindGeoIP 创建MaxMind GeoIP数据库
func NewMaxMindGeoIP() *MaxMindGeoIP {
	return &MaxMindGeoIP{
		data: make(map[string]*GeoLocation),
	}
}

// LoadFromFile 从文件加载GeoIP数据库
func (m *MaxMindGeoIP) LoadFromFile(filePath string) error {
	// 验证文件是否存在
	if _, err := os.Stat(filePath); err != nil {
		return fmt.Errorf("GeoIP database file not found: %s: %w", filePath, err)
	}

	file, err := os.Open(filePath)
	if err != nil {
		return fmt.Errorf("failed to open GeoIP file: %w", err)
	}
	defer file.Close()

	decoder := json.NewDecoder(file)
	
	// 假设文件格式为：{"ip": {"country": "...", "lat": ..., "lon": ...}, ...}
	var geoData map[string]map[string]interface{}
	if err := decoder.Decode(&geoData); err != nil {
		return fmt.Errorf("failed to decode GeoIP file (invalid JSON format): %w", err)
	}

	if len(geoData) == 0 {
		return fmt.Errorf("GeoIP database file is empty or invalid")
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	loadedCount := 0
	for ipStr, data := range geoData {
		location := &GeoLocation{
			IP: ipStr,
		}

		if country, ok := data["country"].(string); ok {
			location.Country = country
		}
		if countryCode, ok := data["country_code"].(string); ok {
			location.CountryCode = countryCode
		}
		if city, ok := data["city"].(string); ok {
			location.City = city
		}
		if lat, ok := data["latitude"].(float64); ok {
			location.Latitude = lat
		}
		if lon, ok := data["longitude"].(float64); ok {
			location.Longitude = lon
		}

		m.data[ipStr] = location
		loadedCount++
	}

	if loadedCount == 0 {
		return fmt.Errorf("no valid GeoIP entries found in file")
	}

	m.loaded = true
	logrus.Infof("Successfully loaded %d GeoIP entries from %s", loadedCount, filePath)
	return nil
}

// Lookup 查询IP地理位置
func (m *MaxMindGeoIP) Lookup(ip net.IP) (*GeoLocation, error) {
	if !m.loaded {
		return nil, fmt.Errorf("GeoIP database not loaded")
	}

	ipStr := ip.String()
	m.mu.RLock()
	defer m.mu.RUnlock()

	location, ok := m.data[ipStr]
	if !ok {
		return nil, fmt.Errorf("IP %s not found in database", ipStr)
	}

	// 返回副本
	return &GeoLocation{
		IP:          location.IP,
		Country:     location.Country,
		CountryCode: location.CountryCode,
		Region:      location.Region,
		City:        location.City,
		Latitude:    location.Latitude,
		Longitude:   location.Longitude,
		ISP:         location.ISP,
		ASN:         location.ASN,
	}, nil
}

// IsLoaded 检查数据库是否已加载
func (m *MaxMindGeoIP) IsLoaded() bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.loaded
}

// EnhancedGeoLocationService 增强的地理位置服务（支持本地数据库和API）
type EnhancedGeoLocationService struct {
	database GeoIPDatabase
	apiService *GeoLocationService
	preferLocal bool // 优先使用本地数据库
	mu sync.RWMutex
}

// NewEnhancedGeoLocationService 创建增强的地理位置服务
func NewEnhancedGeoLocationService(dbPath string, apiURL string, preferLocal bool) (*EnhancedGeoLocationService, error) {
	service := &EnhancedGeoLocationService{
		preferLocal: preferLocal,
		apiService: NewGeoLocationService(apiURL),
	}

	// 如果提供了数据库路径，尝试加载
	if dbPath != "" {
		database := NewMaxMindGeoIP()
		if err := database.LoadFromFile(dbPath); err != nil {
			logrus.Warnf("Failed to load GeoIP database from %s: %v, will use API", dbPath, err)
		} else {
			service.database = database
			logrus.Info("GeoIP database loaded successfully")
		}
	}

	return service, nil
}

// GetLocation 获取IP地理位置（优先使用本地数据库）
func (e *EnhancedGeoLocationService) GetLocation(ip net.IP) (*GeoLocation, error) {
	// 如果优先使用本地数据库且已加载
	if e.preferLocal && e.database != nil && e.database.IsLoaded() {
		location, err := e.database.Lookup(ip)
		if err == nil {
			return location, nil
		}
		logrus.Debugf("GeoIP database lookup failed for %s: %v, falling back to API", ip.String(), err)
	}

	// 回退到API查询
	return e.apiService.GetLocation(ip)
}

// GetLocationForHost 获取主机名的地理位置
func (e *EnhancedGeoLocationService) GetLocationForHost(host string) (*GeoLocation, error) {
	ips, err := net.LookupIP(host)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve host %s: %w", host, err)
	}

	if len(ips) == 0 {
		return nil, fmt.Errorf("no IP found for host %s", host)
	}

	var ipv4 net.IP
	for _, ip := range ips {
		if ip.To4() != nil {
			ipv4 = ip
			break
		}
	}

	if ipv4 == nil {
		ipv4 = ips[0]
	}

	return e.GetLocation(ipv4)
}

// LatencyOptimizedSelector 延迟优化的IP选择器
type LatencyOptimizedSelector struct {
	baseSelector   IPSelector
	geoService     *EnhancedGeoLocationService
	exitIPs        []string
	exitLocations  map[string]*GeoLocation
	latencyHistory map[string][]time.Duration // IP -> 延迟历史
	mu             sync.RWMutex
}

// NewLatencyOptimizedSelector 创建延迟优化的IP选择器
func NewLatencyOptimizedSelector(baseSelector IPSelector, geoService *EnhancedGeoLocationService, exitIPs []string) (*LatencyOptimizedSelector, error) {
	selector := &LatencyOptimizedSelector{
		baseSelector:   baseSelector,
		geoService:     geoService,
		exitIPs:        exitIPs,
		exitLocations: make(map[string]*GeoLocation),
		latencyHistory: make(map[string][]time.Duration),
	}

	// 异步加载出口IP地理位置
	go selector.loadExitIPLocations()

	return selector, nil
}

// loadExitIPLocations 加载出口IP地理位置
func (l *LatencyOptimizedSelector) loadExitIPLocations() {
	for _, ipStr := range l.exitIPs {
		ip := net.ParseIP(ipStr)
		if ip == nil {
			continue
		}

		location, err := l.geoService.GetLocation(ip)
		if err != nil {
			logrus.Warnf("Failed to get location for exit IP %s: %v", ipStr, err)
			continue
		}

		l.mu.Lock()
		l.exitLocations[ipStr] = location
		l.mu.Unlock()

		logrus.Infof("Exit IP %s location: %s, %s", ipStr, location.Country, location.City)
	}
}

// RecordLatency 记录IP的延迟
func (l *LatencyOptimizedSelector) RecordLatency(ipStr string, latency time.Duration) {
	l.mu.Lock()
	defer l.mu.Unlock()

	history := l.latencyHistory[ipStr]
	history = append(history, latency)
	
	// 保留最近100个延迟记录
	if len(history) > 100 {
		history = history[len(history)-100:]
	}
	
	l.latencyHistory[ipStr] = history
}

// SelectIP 选择IP（基于地理位置和延迟）
func (l *LatencyOptimizedSelector) SelectIP(targetAddr string, targetPort int) (net.IP, error) {
	host, _, err := net.SplitHostPort(targetAddr)
	if err != nil {
		host = targetAddr
	}

	// 获取目标的地理位置
	targetLocation, err := l.geoService.GetLocationForHost(host)
	if err != nil {
		logrus.Debugf("Failed to get location for target %s, using base selector: %v", host, err)
		return l.baseSelector.SelectIP(targetAddr, targetPort)
	}

	// 找到距离最近且延迟最低的出口IP
	var bestIP net.IP
	var bestScore float64 = -1

	l.mu.RLock()
	for ipStr, exitLocation := range l.exitLocations {
		if exitLocation.Latitude == 0 && exitLocation.Longitude == 0 {
			continue
		}

		// 计算距离
		distance := CalculateDistance(
			targetLocation.Latitude, targetLocation.Longitude,
			exitLocation.Latitude, exitLocation.Longitude,
		)

		// 计算平均延迟
		var avgLatency time.Duration
		if history, ok := l.latencyHistory[ipStr]; ok && len(history) > 0 {
			var total time.Duration
			for _, lat := range history {
				total += lat
			}
			avgLatency = total / time.Duration(len(history))
		} else {
			// 如果没有延迟历史，使用距离估算延迟（每1000km约30ms）
			avgLatency = time.Duration(distance/1000.0*30) * time.Millisecond
		}

		// 综合评分：距离权重0.3，延迟权重0.7
		// 分数越低越好，所以取倒数
		distanceScore := 1.0 / (distance + 1.0) // +1避免除零
		latencyScore := 1.0 / (float64(avgLatency.Milliseconds()) + 1.0)
		score := distanceScore*0.3 + latencyScore*0.7

		if score > bestScore {
			bestScore = score
			bestIP = net.ParseIP(ipStr)
		}
	}
	l.mu.RUnlock()

	if bestIP != nil {
		logrus.Debugf("Selected IP %s for target %s (distance: %.2f km, target: %s)",
			bestIP.String(), host, CalculateDistance(
				targetLocation.Latitude, targetLocation.Longitude,
				l.exitLocations[bestIP.String()].Latitude,
				l.exitLocations[bestIP.String()].Longitude,
			), targetLocation.Country)
		return bestIP, nil
	}

	// 如果没有找到合适的IP，回退到基础选择器
	return l.baseSelector.SelectIP(targetAddr, targetPort)
}

