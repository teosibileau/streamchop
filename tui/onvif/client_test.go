package onvif

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

const capabilitiesXML = `<?xml version="1.0" encoding="UTF-8"?>
<s:Envelope xmlns:s="http://www.w3.org/2003/05/soap-envelope">
  <s:Body>
    <tds:GetCapabilitiesResponse xmlns:tds="http://www.onvif.org/ver10/device/wsdl" xmlns:tt="http://www.onvif.org/ver10/schema">
      <tds:Capabilities>
        <tt:Media>
          <tt:XAddr>MEDIA_URL</tt:XAddr>
        </tt:Media>
      </tds:Capabilities>
    </tds:GetCapabilitiesResponse>
  </s:Body>
</s:Envelope>`

const profilesXML = `<?xml version="1.0" encoding="UTF-8"?>
<s:Envelope xmlns:s="http://www.w3.org/2003/05/soap-envelope">
  <s:Body>
    <trt:GetProfilesResponse xmlns:trt="http://www.onvif.org/ver10/media/wsdl" xmlns:tt="http://www.onvif.org/ver10/schema">
      <trt:Profiles token="profile1">
        <tt:Name>MainStream</tt:Name>
        <tt:VideoEncoderConfiguration>
          <tt:Encoding>H264</tt:Encoding>
        </tt:VideoEncoderConfiguration>
      </trt:Profiles>
      <trt:Profiles token="profile2">
        <tt:Name>SubStream</tt:Name>
        <tt:VideoEncoderConfiguration>
          <tt:Encoding>JPEG</tt:Encoding>
        </tt:VideoEncoderConfiguration>
      </trt:Profiles>
    </trt:GetProfilesResponse>
  </s:Body>
</s:Envelope>`

const streamURIXML = `<?xml version="1.0" encoding="UTF-8"?>
<s:Envelope xmlns:s="http://www.w3.org/2003/05/soap-envelope">
  <s:Body>
    <trt:GetStreamUriResponse xmlns:trt="http://www.onvif.org/ver10/media/wsdl" xmlns:tt="http://www.onvif.org/ver10/schema">
      <trt:MediaUri>
        <tt:Uri>rtsp://192.168.1.100:554/stream1</tt:Uri>
      </trt:MediaUri>
    </trt:GetStreamUriResponse>
  </s:Body>
</s:Envelope>`

func TestGetStreamURIs(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body := make([]byte, r.ContentLength)
		r.Body.Read(body)
		content := string(body)

		w.Header().Set("Content-Type", "application/soap+xml")
		switch {
		case strings.Contains(content, "GetCapabilities"):
			resp := strings.Replace(capabilitiesXML, "MEDIA_URL", "http://"+r.Host+"/onvif/media", 1)
			w.Write([]byte(resp))
		case strings.Contains(content, "GetProfiles"):
			w.Write([]byte(profilesXML))
		case strings.Contains(content, "GetStreamUri"):
			w.Write([]byte(streamURIXML))
		default:
			http.Error(w, "unknown request", http.StatusBadRequest)
		}
	})

	server := httptest.NewServer(handler)
	defer server.Close()

	creds := Credentials{Username: "admin", Password: "admin"}
	streams, err := GetStreamURIs(server.URL+"/onvif/device_service", creds)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(streams) != 2 {
		t.Fatalf("expected 2 streams, got %d", len(streams))
	}

	if streams[0].ProfileToken != "profile1" {
		t.Errorf("expected profile1, got %s", streams[0].ProfileToken)
	}
	if streams[0].ProfileName != "MainStream" {
		t.Errorf("expected MainStream, got %s", streams[0].ProfileName)
	}
	if !strings.Contains(streams[0].URI, "admin:admin") {
		t.Errorf("expected credentials in URI, got %s", streams[0].URI)
	}
}

func TestNormalizeRTSPURI(t *testing.T) {
	creds := Credentials{Username: "user", Password: "pass"}

	tests := []struct {
		name     string
		rtspURI  string
		xaddr    string
		wantHost string
		wantUser string
	}{
		{
			name:     "absolute URI",
			rtspURI:  "rtsp://192.168.1.100:554/stream1",
			xaddr:    "http://192.168.1.100:80/onvif/device_service",
			wantHost: "192.168.1.100:554",
			wantUser: "user:pass",
		},
		{
			name:     "relative URI",
			rtspURI:  "/stream1",
			xaddr:    "http://192.168.1.50:80/onvif/device_service",
			wantHost: "192.168.1.50:80",
			wantUser: "user:pass",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := normalizeRTSPURI(tt.rtspURI, tt.xaddr, creds)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if !strings.Contains(result, tt.wantHost) {
				t.Errorf("expected host %s in %s", tt.wantHost, result)
			}
			if !strings.Contains(result, tt.wantUser) {
				t.Errorf("expected user %s in %s", tt.wantUser, result)
			}
		})
	}
}

func TestFilterH264Profiles(t *testing.T) {
	profiles := []profile{
		{Token: "p1", Name: "Main", Video: struct {
			Encoding string `xml:"Encoding"`
		}{Encoding: "H264"}},
		{Token: "p2", Name: "Sub", Video: struct {
			Encoding string `xml:"Encoding"`
		}{Encoding: "JPEG"}},
	}

	streams := []StreamInfo{
		{ProfileToken: "p1", ProfileName: "Main", URI: "rtsp://a"},
		{ProfileToken: "p2", ProfileName: "Sub", URI: "rtsp://b"},
	}

	filtered := FilterH264Profiles(streams, profiles)
	if len(filtered) != 1 {
		t.Fatalf("expected 1 H264 stream, got %d", len(filtered))
	}
	if filtered[0].ProfileToken != "p1" {
		t.Errorf("expected p1, got %s", filtered[0].ProfileToken)
	}
}

func TestFindNestedElement(t *testing.T) {
	data := strings.Replace(capabilitiesXML, "MEDIA_URL", "http://cam:80/onvif/media", 1)
	result := findNestedElement([]byte(data), "Media", "XAddr")
	if result != "http://cam:80/onvif/media" {
		t.Errorf("expected http://cam:80/onvif/media, got %s", result)
	}
}

func TestParseProfiles(t *testing.T) {
	profiles, err := parseProfiles([]byte(profilesXML))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(profiles) != 2 {
		t.Fatalf("expected 2 profiles, got %d", len(profiles))
	}
	if profiles[0].Token != "profile1" || profiles[0].Name != "MainStream" {
		t.Errorf("unexpected profile 0: %+v", profiles[0])
	}
	if profiles[0].Video.Encoding != "H264" {
		t.Errorf("expected H264 encoding, got %s", profiles[0].Video.Encoding)
	}
	if profiles[1].Token != "profile2" || profiles[1].Video.Encoding != "JPEG" {
		t.Errorf("unexpected profile 1: %+v", profiles[1])
	}
}
