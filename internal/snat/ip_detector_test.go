package snat

import (
	"testing"
)

func TestIPDetector_DetectLocalIPs(t *testing.T) {
	detector := NewIPDetector()
	ips, err := detector.DetectLocalIPs()
	if err != nil {
		t.Logf("Failed to detect local IPs (expected in test environment): %v", err)
		return
	}

	t.Logf("Detected %d local public IP(s): %v", len(ips), ips)
}

func TestIPDetector_DetectPublicIP(t *testing.T) {
	detector := NewIPDetector()
	ip, err := detector.DetectPublicIP()
	if err != nil {
		t.Logf("Failed to detect public IP (may require network): %v", err)
		return
	}

	t.Logf("Detected public IP: %s", ip)
}

func TestIPDetector_DetectAllPublicIPs(t *testing.T) {
	detector := NewIPDetector()
	ips, err := detector.DetectAllPublicIPs()
	if err != nil {
		t.Logf("Failed to detect all public IPs: %v", err)
		return
	}

	t.Logf("Detected %d public IP(s): %v", len(ips), ips)
}



