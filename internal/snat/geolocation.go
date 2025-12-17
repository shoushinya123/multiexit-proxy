package snat

import (
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net"
	"net/http"
	"sync"
	"time"

	"github.com/sirupsen/logrus"
)

// GeoLocation IP地理位置信息
type GeoLocation struct {
	IP          string  `json:"ip"`
	Country     string  `json:"country"`
	CountryCode string  `json:"country_code"`
	Region      string  `json:"region"`
	City        string  `json:"city"`
	Latitude    float64 `json:"latitude"`
	Longitude   float64 `json:"longitude"`
	ISP         string  `json:"isp"`
	ASN         string  `json:"asn"`
}

// GeoLocationService 地理位置服务
type GeoLocationService struct {
	apiURL      string
	httpClient  *http.Client
	cache       map[string]*GeoLocation
	cacheMu     sync.RWMutex
	cacheExpiry time.Duration
}

// NewGeoLocationService 创建地理位置服务
func NewGeoLocationService(apiURL string) *GeoLocationService {
	if apiURL == "" {
		// 默认使用ip-api.com（免费，无需API密钥）
		apiURL = "http://ip-api.com/json"
	}

	return &GeoLocationService{
		apiURL: apiURL,
		httpClient: &http.Client{
			Timeout: 5 * time.Second,
		},
		cache:       make(map[string]*GeoLocation),
		cacheExpiry: 24 * time.Hour, // 缓存24小时
	}
}

// GetLocation 获取IP的地理位置信息
func (g *GeoLocationService) GetLocation(ip net.IP) (*GeoLocation, error) {
	ipStr := ip.String()

	// 检查缓存
	g.cacheMu.RLock()
	if cached, ok := g.cache[ipStr]; ok {
		g.cacheMu.RUnlock()
		return cached, nil
	}
	g.cacheMu.RUnlock()

	// 查询地理位置
	url := fmt.Sprintf("%s/%s", g.apiURL, ipStr)
	resp, err := g.httpClient.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to query geo location API: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("geo location API returned status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read geo location API response: %w", err)
	}

	var location GeoLocation
	if err := json.Unmarshal(body, &location); err != nil {
		return nil, fmt.Errorf("failed to parse geo location API response: %w", err)
	}

	location.IP = ipStr

	// 缓存结果
	g.cacheMu.Lock()
	g.cache[ipStr] = &location
	g.cacheMu.Unlock()

	logrus.Debugf("Geo location for %s: %s, %s", ipStr, location.Country, location.City)
	return &location, nil
}

// GetLocationForHost 获取主机名的地理位置（通过DNS解析）
func (g *GeoLocationService) GetLocationForHost(host string) (*GeoLocation, error) {
	// 解析主机名到IP
	ips, err := net.LookupIP(host)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve host %s: %w", host, err)
	}

	if len(ips) == 0 {
		return nil, fmt.Errorf("no IP found for host %s", host)
	}

	// 使用第一个IPv4地址
	var ipv4 net.IP
	for _, ip := range ips {
		if ip.To4() != nil {
			ipv4 = ip
			break
		}
	}

	if ipv4 == nil {
		// 如果没有IPv4，使用第一个IP
		ipv4 = ips[0]
	}

	return g.GetLocation(ipv4)
}

// CalculateDistance 计算两个地理位置之间的距离（公里）
func CalculateDistance(lat1, lon1, lat2, lon2 float64) float64 {
	const earthRadiusKm = 6371.0

	// 转换为弧度
	lat1Rad := lat1 * 3.141592653589793 / 180.0
	lon1Rad := lon1 * 3.141592653589793 / 180.0
	lat2Rad := lat2 * 3.141592653589793 / 180.0
	lon2Rad := lon2 * 3.141592653589793 / 180.0

	// Haversine公式
	dlat := lat2Rad - lat1Rad
	dlon := lon2Rad - lon1Rad

	a := math.Sin(dlat/2)*math.Sin(dlat/2) +
		math.Cos(lat1Rad)*math.Cos(lat2Rad)*
			math.Sin(dlon/2)*math.Sin(dlon/2)

	c := 2 * math.Atan2(math.Sqrt(a), math.Sqrt(1-a))

	return earthRadiusKm * c
}

// GeoLocationSelector 基于地理位置的IP选择器
type GeoLocationSelector struct {
	baseSelector   IPSelector
	geoService     GeoLocationServiceInterface
	exitIPs        []string
	exitLocations  map[string]*GeoLocation
	mu             sync.RWMutex
}

// NewGeoLocationSelector 创建基于地理位置的IP选择器
func NewGeoLocationSelector(baseSelector IPSelector, geoService GeoLocationServiceInterface, exitIPs []string) (*GeoLocationSelector, error) {
	selector := &GeoLocationSelector{
		baseSelector:  baseSelector,
		geoService:    geoService,
		exitIPs:       exitIPs,
		exitLocations: make(map[string]*GeoLocation),
	}

	// 异步加载所有出口IP的地理位置
	go selector.loadExitIPLocations()

	return selector, nil
}

// loadExitIPLocations 加载所有出口IP的地理位置
func (g *GeoLocationSelector) loadExitIPLocations() {
	for _, ipStr := range g.exitIPs {
		ip := net.ParseIP(ipStr)
		if ip == nil {
			continue
		}

		location, err := g.geoService.GetLocation(ip)
		if err != nil {
			logrus.Warnf("Failed to get location for exit IP %s: %v", ipStr, err)
			continue
		}

		g.mu.Lock()
		g.exitLocations[ipStr] = location
		g.mu.Unlock()

		logrus.Infof("Exit IP %s location: %s, %s (lat: %.2f, lon: %.2f)",
			ipStr, location.Country, location.City, location.Latitude, location.Longitude)
	}
}

// SelectIP 选择IP（基于地理位置）
func (g *GeoLocationSelector) SelectIP(targetAddr string, targetPort int) (net.IP, error) {
	host, _, err := net.SplitHostPort(targetAddr)
	if err != nil {
		host = targetAddr
	}

	// 获取目标的地理位置
	targetLocation, err := g.geoService.GetLocationForHost(host)
	if err != nil {
		logrus.Warnf("Failed to get location for target %s, using base selector: %v", host, err)
		// 如果无法获取目标位置，回退到基础选择器
		return g.baseSelector.SelectIP(targetAddr, targetPort)
	}

	// 找到距离最近的出口IP
	var bestIP net.IP
	var minDistance float64 = math.MaxFloat64

	g.mu.RLock()
	for ipStr, exitLocation := range g.exitLocations {
		if exitLocation.Latitude == 0 && exitLocation.Longitude == 0 {
			continue // 跳过无效位置
		}

		distance := CalculateDistance(
			targetLocation.Latitude, targetLocation.Longitude,
			exitLocation.Latitude, exitLocation.Longitude,
		)

		if distance < minDistance {
			minDistance = distance
			bestIP = net.ParseIP(ipStr)
		}
	}
	g.mu.RUnlock()

	if bestIP != nil {
		logrus.Debugf("Selected IP %s for target %s (distance: %.2f km, target: %s, %s)",
			bestIP.String(), host, minDistance, targetLocation.Country, targetLocation.City)
		return bestIP, nil
	}

	// 如果没有找到合适的IP，回退到基础选择器
	logrus.Debugf("No geo-located exit IP found, using base selector")
	return g.baseSelector.SelectIP(targetAddr, targetPort)
}

