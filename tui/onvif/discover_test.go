package onvif

import (
	"testing"
)

func TestParseProbeMatch(t *testing.T) {
	xmlData := `<?xml version="1.0" encoding="UTF-8"?>
<s:Envelope xmlns:s="http://www.w3.org/2003/05/soap-envelope"
            xmlns:d="http://schemas.xmlsoap.org/ws/2005/04/discovery"
            xmlns:a="http://schemas.xmlsoap.org/ws/2004/08/addressing">
  <s:Body>
    <d:ProbeMatches>
      <d:ProbeMatch>
        <d:XAddrs>http://192.168.1.100:80/onvif/device_service</d:XAddrs>
        <d:Scopes>onvif://www.onvif.org/name/TestCam onvif://www.onvif.org/type/video_encoder</d:Scopes>
      </d:ProbeMatch>
    </d:ProbeMatches>
  </s:Body>
</s:Envelope>`

	cameras, err := parseProbeMatch([]byte(xmlData))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(cameras) != 1 {
		t.Fatalf("expected 1 camera, got %d", len(cameras))
	}

	cam := cameras[0]
	if cam.IP != "192.168.1.100" {
		t.Errorf("expected IP 192.168.1.100, got %s", cam.IP)
	}
	if cam.Port != "80" {
		t.Errorf("expected port 80, got %s", cam.Port)
	}
	if cam.Name != "TestCam" {
		t.Errorf("expected name TestCam, got %s", cam.Name)
	}
	if cam.XAddr != "http://192.168.1.100:80/onvif/device_service" {
		t.Errorf("unexpected XAddr: %s", cam.XAddr)
	}
}

func TestParseProbeMatchMultipleXAddrs(t *testing.T) {
	xmlData := `<?xml version="1.0" encoding="UTF-8"?>
<s:Envelope xmlns:s="http://www.w3.org/2003/05/soap-envelope"
            xmlns:d="http://schemas.xmlsoap.org/ws/2005/04/discovery">
  <s:Body>
    <d:ProbeMatches>
      <d:ProbeMatch>
        <d:XAddrs>http://192.168.1.50:80/onvif/device_service http://10.0.0.50:80/onvif/device_service</d:XAddrs>
        <d:Scopes>onvif://www.onvif.org/name/DualNIC</d:Scopes>
      </d:ProbeMatch>
    </d:ProbeMatches>
  </s:Body>
</s:Envelope>`

	cameras, err := parseProbeMatch([]byte(xmlData))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(cameras) != 2 {
		t.Fatalf("expected 2 cameras, got %d", len(cameras))
	}
}

func TestParseProbeMatchNoScopes(t *testing.T) {
	xmlData := `<?xml version="1.0" encoding="UTF-8"?>
<s:Envelope xmlns:s="http://www.w3.org/2003/05/soap-envelope"
            xmlns:d="http://schemas.xmlsoap.org/ws/2005/04/discovery">
  <s:Body>
    <d:ProbeMatches>
      <d:ProbeMatch>
        <d:XAddrs>http://192.168.1.200:8080/onvif/device_service</d:XAddrs>
        <d:Scopes></d:Scopes>
      </d:ProbeMatch>
    </d:ProbeMatches>
  </s:Body>
</s:Envelope>`

	cameras, err := parseProbeMatch([]byte(xmlData))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(cameras) != 1 {
		t.Fatalf("expected 1 camera, got %d", len(cameras))
	}

	if cameras[0].Name != "192.168.1.200" {
		t.Errorf("expected name to fallback to IP, got %s", cameras[0].Name)
	}
	if cameras[0].Port != "8080" {
		t.Errorf("expected port 8080, got %s", cameras[0].Port)
	}
}

func TestExtractNameFromScopes(t *testing.T) {
	tests := []struct {
		scopes string
		want   string
	}{
		{
			scopes: "onvif://www.onvif.org/name/MyCam onvif://www.onvif.org/type/video",
			want:   "MyCam",
		},
		{
			scopes: "onvif://www.onvif.org/name/My%20Camera",
			want:   "My Camera",
		},
		{
			scopes: "onvif://www.onvif.org/type/video",
			want:   "",
		},
		{
			scopes: "",
			want:   "",
		},
	}

	for _, tt := range tests {
		got := extractNameFromScopes(tt.scopes)
		if got != tt.want {
			t.Errorf("extractNameFromScopes(%q) = %q, want %q", tt.scopes, got, tt.want)
		}
	}
}
