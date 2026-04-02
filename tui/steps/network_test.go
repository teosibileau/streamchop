package steps

import "testing"

func TestSubnetFromIP(t *testing.T) {
	tests := []struct {
		ip   string
		want string
	}{
		{"192.168.1.5", "192.168.1"},
		{"10.0.0.1", "10.0.0"},
		{"invalid", ""},
		{"", ""},
	}

	for _, tt := range tests {
		got := subnetFromIP(tt.ip)
		if got != tt.want {
			t.Errorf("subnetFromIP(%q) = %q, want %q", tt.ip, got, tt.want)
		}
	}
}

func TestDetectHostIP(t *testing.T) {
	ip, err := DetectHostIP()
	if err != nil {
		t.Skipf("no network available: %v", err)
	}
	if ip == "" {
		t.Error("expected non-empty IP")
	}
	if ip == "127.0.0.1" {
		t.Error("expected LAN IP, got loopback")
	}
}
