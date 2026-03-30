package onvif

import (
	"bytes"
	"crypto/sha1"
	"encoding/base64"
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/google/uuid"
)

// GetStreamURIs connects to an ONVIF camera, discovers its media profiles,
// and returns the RTSP stream URI for each profile.
func GetStreamURIs(xaddr string, creds Credentials) ([]StreamInfo, error) {
	mediaURL, err := getMediaURL(xaddr, creds)
	if err != nil {
		return nil, fmt.Errorf("get media URL: %w", err)
	}

	profiles, err := getProfiles(mediaURL, creds)
	if err != nil {
		return nil, fmt.Errorf("get profiles: %w", err)
	}

	var streams []StreamInfo
	for _, p := range profiles {
		uri, err := getStreamURI(mediaURL, creds, p.Token)
		if err != nil {
			continue
		}

		normalized, err := normalizeRTSPURI(uri, xaddr, creds)
		if err != nil {
			continue
		}

		streams = append(streams, StreamInfo{
			ProfileToken: p.Token,
			ProfileName:  p.Name,
			URI:          normalized,
		})
	}

	if len(streams) == 0 {
		return nil, fmt.Errorf("no stream URIs found")
	}

	return streams, nil
}

func getMediaURL(xaddr string, creds Credentials) (string, error) {
	body := soapEnvelope(creds, `<GetCapabilities xmlns="http://www.onvif.org/ver10/device/wsdl">
    <Category>Media</Category>
  </GetCapabilities>`)

	resp, err := soapCall(xaddr, body)
	if err != nil {
		return "", err
	}

	// Walk the XML looking for Media element, then its XAddr child
	mediaURL := findNestedElement(resp, "Media", "XAddr")
	if mediaURL == "" {
		return "", fmt.Errorf("media XAddr not found in capabilities")
	}

	return mediaURL, nil
}

func getProfiles(mediaURL string, creds Credentials) ([]profile, error) {
	body := soapEnvelope(creds, `<GetProfiles xmlns="http://www.onvif.org/ver10/media/wsdl"/>`)

	resp, err := soapCall(mediaURL, body)
	if err != nil {
		return nil, err
	}

	return parseProfiles(resp)
}

func getStreamURI(mediaURL string, creds Credentials, profileToken string) (string, error) {
	body := soapEnvelope(creds, fmt.Sprintf(`<GetStreamUri xmlns="http://www.onvif.org/ver10/media/wsdl">
    <StreamSetup>
      <Stream xmlns="http://www.onvif.org/ver10/schema">RTP-Unicast</Stream>
      <Transport xmlns="http://www.onvif.org/ver10/schema">
        <Protocol>RTSP</Protocol>
      </Transport>
    </StreamSetup>
    <ProfileToken>%s</ProfileToken>
  </GetStreamUri>`, profileToken))

	resp, err := soapCall(mediaURL, body)
	if err != nil {
		return "", err
	}

	uri := findNestedElement(resp, "MediaUri", "Uri")
	if uri == "" {
		return "", fmt.Errorf("empty stream URI")
	}

	return uri, nil
}

// findNestedElement walks XML tokens looking for a parent element by local name,
// then returns the text content of the first child matching childLocal.
// This is namespace-agnostic, which handles ONVIF's varied namespace usage.
func findNestedElement(data []byte, parentLocal, childLocal string) string {
	decoder := xml.NewDecoder(bytes.NewReader(data))
	inParent := false

	for {
		tok, err := decoder.Token()
		if err != nil {
			return ""
		}

		switch t := tok.(type) {
		case xml.StartElement:
			if t.Name.Local == parentLocal {
				inParent = true
			}
			if inParent && t.Name.Local == childLocal {
				var content string
				if err := decoder.DecodeElement(&content, &t); err == nil {
					return strings.TrimSpace(content)
				}
			}
		case xml.EndElement:
			if t.Name.Local == parentLocal {
				inParent = false
			}
		}
	}
}

// parseProfiles walks XML tokens to extract profile info in a namespace-agnostic way.
func parseProfiles(data []byte) ([]profile, error) {
	decoder := xml.NewDecoder(bytes.NewReader(data))
	var profiles []profile

	for {
		tok, err := decoder.Token()
		if err != nil {
			break
		}

		start, ok := tok.(xml.StartElement)
		if !ok || start.Name.Local != "Profiles" {
			continue
		}

		var p profile
		for _, attr := range start.Attr {
			if attr.Name.Local == "token" {
				p.Token = attr.Value
			}
		}

		// Read inner elements of this Profiles block
		depth := 1
		for depth > 0 {
			inner, err := decoder.Token()
			if err != nil {
				break
			}
			switch t := inner.(type) {
			case xml.StartElement:
				depth++
				switch t.Name.Local {
				case "Name":
					if depth == 2 { // direct child
						var name string
						decoder.DecodeElement(&name, &t)
						p.Name = name
						depth-- // DecodeElement consumed the end element
					}
				case "VideoEncoderConfiguration":
					// Look for Encoding inside
					vecDepth := 1
					for vecDepth > 0 {
						vecTok, err := decoder.Token()
						if err != nil {
							break
						}
						switch vt := vecTok.(type) {
						case xml.StartElement:
							vecDepth++
							if vt.Name.Local == "Encoding" {
								var enc string
								decoder.DecodeElement(&enc, &vt)
								p.Video.Encoding = enc
								vecDepth--
							}
						case xml.EndElement:
							vecDepth--
						}
					}
					depth-- // VideoEncoderConfiguration end consumed
				}
			case xml.EndElement:
				depth--
			}
		}

		if p.Token != "" {
			profiles = append(profiles, p)
		}
	}

	return profiles, nil
}

func normalizeRTSPURI(rtspURI, xaddr string, creds Credentials) (string, error) {
	u, err := url.Parse(rtspURI)
	if err != nil {
		return "", err
	}

	if !u.IsAbs() {
		base, err := url.Parse(xaddr)
		if err != nil {
			return "", err
		}
		u.Scheme = "rtsp"
		u.Host = base.Host
	}

	u.User = url.UserPassword(creds.Username, creds.Password)

	return u.String(), nil
}

func soapCall(endpoint string, body []byte) ([]byte, error) {
	client := &http.Client{Timeout: 10 * time.Second}

	req, err := http.NewRequest("POST", endpoint, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/soap+xml; charset=utf-8")

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("SOAP call failed: %s: %s", resp.Status, string(data))
	}

	return data, nil
}

func soapEnvelope(creds Credentials, innerBody string) []byte {
	security := wsSecurityHeader(creds.Username, creds.Password)

	envelope := fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?>
<s:Envelope xmlns:s="http://www.w3.org/2003/05/soap-envelope">
  <s:Header>%s</s:Header>
  <s:Body>%s</s:Body>
</s:Envelope>`, security, innerBody)

	return []byte(envelope)
}

func wsSecurityHeader(username, password string) string {
	nonce := make([]byte, 16)
	copy(nonce, []byte(uuid.New().String())[:16])

	created := time.Now().UTC().Format(time.RFC3339Nano)

	h := sha1.New()
	h.Write(nonce)
	h.Write([]byte(created))
	h.Write([]byte(password))
	digest := h.Sum(nil)

	return fmt.Sprintf(`<Security xmlns="http://docs.oasis-open.org/wss/2004/01/oasis-200401-wss-wssecurity-secext-1.0.xsd" s:mustUnderstand="true">
      <UsernameToken>
        <Username>%s</Username>
        <Password Type="http://docs.oasis-open.org/wss/2004/01/oasis-200401-wss-username-token-profile-1.0#PasswordDigest">%s</Password>
        <Nonce EncodingType="http://docs.oasis-open.org/wss/2004/01/oasis-200401-wss-soap-message-security-1.0#Base64Binary">%s</Nonce>
        <Created xmlns="http://docs.oasis-open.org/wss/2004/01/oasis-200401-wss-wssecurity-utility-1.0.xsd">%s</Created>
      </UsernameToken>
    </Security>`,
		username,
		base64.StdEncoding.EncodeToString(digest),
		base64.StdEncoding.EncodeToString(nonce),
		created)
}

// FilterH264Profiles returns profiles that use H.264 encoding.
// If none are found, returns all profiles as fallback.
func FilterH264Profiles(streams []StreamInfo, profiles []profile) []StreamInfo {
	h264Tokens := make(map[string]bool)
	for _, p := range profiles {
		if strings.EqualFold(p.Video.Encoding, "H264") {
			h264Tokens[p.Token] = true
		}
	}

	if len(h264Tokens) == 0 {
		return streams
	}

	var filtered []StreamInfo
	for _, s := range streams {
		if h264Tokens[s.ProfileToken] {
			filtered = append(filtered, s)
		}
	}

	if len(filtered) == 0 {
		return streams
	}
	return filtered
}
