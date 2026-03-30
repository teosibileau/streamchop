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

// ONVIF SOAP response types
// Namespace constants for ONVIF XML parsing
const (
	nsSOAP  = "http://www.w3.org/2003/05/soap-envelope"
	nsDevice = "http://www.onvif.org/ver10/device/wsdl"
	nsMedia  = "http://www.onvif.org/ver10/media/wsdl"
	nsSchema = "http://www.onvif.org/ver10/schema"
)

type capabilitiesResponse struct {
	XMLName xml.Name `xml:"Envelope"`
	Body    struct {
		Response struct {
			Capabilities struct {
				Media struct {
					XAddr string `xml:"XAddr"`
				} `xml:"Media"`
			} `xml:"Capabilities"`
		} `xml:"GetCapabilitiesResponse"`
	} `xml:"Body"`
}

type profile struct {
	Token string `xml:"token,attr"`
	Name  string `xml:"Name"`
	Video struct {
		Encoding string `xml:"Encoding"`
	} `xml:"VideoEncoderConfiguration"`
}

type profilesResponse struct {
	XMLName xml.Name `xml:"Envelope"`
	Body    struct {
		Response struct {
			Profiles []profile `xml:"Profiles"`
		} `xml:"GetProfilesResponse"`
	} `xml:"Body"`
}

type streamURIResponse struct {
	XMLName xml.Name `xml:"Envelope"`
	Body    struct {
		Response struct {
			MediaURI struct {
				URI string `xml:"Uri"`
			} `xml:"MediaUri"`
		} `xml:"GetStreamUriResponse"`
	} `xml:"Body"`
}
