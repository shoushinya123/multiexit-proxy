package snat

import "fmt"

// InvalidIPError IP地址无效错误
type InvalidIPError struct {
	IP string
}

func (e *InvalidIPError) Error() string {
	return fmt.Sprintf("invalid IP address: %s", e.IP)
}

// NoIPAvailableError 没有可用IP错误
type NoIPAvailableError struct{}

func (e *NoIPAvailableError) Error() string {
	return "no IP available"
}


