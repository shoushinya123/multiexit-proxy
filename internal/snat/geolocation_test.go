package snat

import (
	"net"
	"testing"
	"time"
)

func TestCalculateDistance(t *testing.T) {
	// 测试两个已知位置之间的距离
	// 北京: 39.9042, 116.4074
	// 上海: 31.2304, 121.4737
	// 预期距离约1067公里

	distance := CalculateDistance(39.9042, 116.4074, 31.2304, 121.4737)
	expectedMin := 1000.0 // 公里
	expectedMax := 1200.0 // 公里

	if distance < expectedMin || distance > expectedMax {
		t.Errorf("Expected distance between 1000-1200 km, got %.2f km", distance)
	}
	t.Logf("Distance: %.2f km", distance)

	// 测试相同位置
	distanceSame := CalculateDistance(39.9042, 116.4074, 39.9042, 116.4074)
	if distanceSame > 1.0 {
		t.Errorf("Expected distance ~0 km for same location, got %.2f km", distanceSame)
	}
}

func TestGeoLocationService_GetLocation(t *testing.T) {
	// 注意：这个测试需要网络连接
	// 如果网络不可用，测试会失败
	service := NewGeoLocationService("")

	// 测试查询一个已知IP（Google DNS）
	ip := net.ParseIP("8.8.8.8")
	if ip == nil {
		t.Fatal("Failed to parse IP")
	}

	location, err := service.GetLocation(ip)
	if err != nil {
		t.Logf("Failed to get location (may need network): %v", err)
		return
	}

	if location.IP != ip.String() {
		t.Errorf("Expected IP %s, got %s", ip.String(), location.IP)
	}

	if location.Country == "" {
		t.Error("Expected country to be set")
	}

	t.Logf("Location for %s: %s, %s", ip.String(), location.Country, location.City)
}

func TestGeoLocationSelector_SelectIP(t *testing.T) {
	// 创建基础选择器
	ips := []string{"8.8.8.8", "1.1.1.1"}
	baseSelector, err := NewRoundRobinSelector(ips)
	if err != nil {
		t.Fatalf("Failed to create base selector: %v", err)
	}

	// 创建地理位置服务
	geoService := NewGeoLocationService("")

	// 创建地理位置选择器
	selector, err := NewGeoLocationSelector(baseSelector, geoService, ips)
	if err != nil {
		t.Fatalf("Failed to create geo location selector: %v", err)
	}

	// 等待位置加载（异步）
	time.Sleep(2 * time.Second)

	// 测试选择IP
	selectedIP, err := selector.SelectIP("example.com", 80)
	if err != nil {
		t.Logf("SelectIP failed (may need network): %v", err)
		return
	}

	if selectedIP == nil {
		t.Error("Expected selected IP, got nil")
	} else {
		t.Logf("Selected IP: %s", selectedIP.String())
	}
}

