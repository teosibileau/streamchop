package steps

import (
	"fmt"
	"net"
	"strings"
	"sync"
	"time"
)

// DetectHostIP returns the host's LAN IP address by dialing a known external
// address and reading the local address chosen by the OS routing table.
// It does not actually send any traffic.
func DetectHostIP() (string, error) {
	conn, err := net.DialTimeout("udp4", "8.8.8.8:80", 2*time.Second)
	if err != nil {
		return "", fmt.Errorf("detect host IP: %w", err)
	}
	defer func() { _ = conn.Close() }()

	localAddr, ok := conn.LocalAddr().(*net.UDPAddr)
	if !ok {
		return "", fmt.Errorf("unexpected address type")
	}
	return localAddr.IP.String(), nil
}

// ScanMQTTBrokers scans the local subnet for MQTT brokers by attempting
// TCP connections on port 1883. Returns all hosts found.
func ScanMQTTBrokers(hostIP string) ([]string, error) {
	subnet := subnetFromIP(hostIP)
	if subnet == "" {
		return nil, fmt.Errorf("could not determine subnet from %s", hostIP)
	}

	results := make(chan string, 254)
	var wg sync.WaitGroup

	for i := 1; i < 255; i++ {
		ip := fmt.Sprintf("%s.%d", subnet, i)
		wg.Add(1)
		go func(addr string) {
			defer wg.Done()
			conn, err := net.DialTimeout("tcp", addr+":1883", 500*time.Millisecond)
			if err != nil {
				return
			}
			_ = conn.Close()
			results <- addr
		}(ip)
	}

	go func() {
		wg.Wait()
		close(results)
	}()

	var found []string
	for addr := range results {
		found = append(found, addr)
	}

	if len(found) == 0 {
		return nil, fmt.Errorf("no MQTT broker found on %s.0/24", subnet)
	}

	return found, nil
}

func subnetFromIP(ip string) string {
	parts := strings.Split(ip, ".")
	if len(parts) != 4 {
		return ""
	}
	return strings.Join(parts[:3], ".")
}
