package snat

import "net"

// GeoLocationServiceInterface 地理位置服务接口
type GeoLocationServiceInterface interface {
	GetLocation(ip net.IP) (*GeoLocation, error)
	GetLocationForHost(host string) (*GeoLocation, error)
}

