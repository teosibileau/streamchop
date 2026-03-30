package onvif

import "encoding/xml"

// Camera represents a discovered ONVIF camera on the network.
type Camera struct {
	Name  string
	IP    string
	Port  string
	XAddr string
}

// Credentials holds authentication info for an ONVIF camera.
type Credentials struct {
	Username string
	Password string
}

// StreamInfo holds the resolved RTSP URI for a camera profile.
type StreamInfo struct {
	ProfileToken string
	ProfileName  string
	URI          string
}

// WS-Discovery types

type probeMatch struct {
	XMLName xml.Name `xml:"Envelope"`
	Body    struct {
		ProbeMatches struct {
			Matches []struct {
				XAddrs string `xml:"XAddrs"`
				Scopes string `xml:"Scopes"`
			} `xml:"ProbeMatch"`
		} `xml:"ProbeMatches"`
	} `xml:"Body"`
}

// profile holds parsed ONVIF media profile data.
type profile struct {
	Token string
	Name  string
	Video struct {
		Encoding string
	}
}
