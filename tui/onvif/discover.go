package onvif

import (
	"encoding/xml"
	"fmt"
	"net"
	"net/url"
	"strings"
	"time"

	"github.com/google/uuid"
)

const (
	multicastAddr = "239.255.255.250:3702"
	probeTimeout  = 5 * time.Second
)

var probeTemplate = `<?xml version="1.0" encoding="UTF-8"?>
<s:Envelope xmlns:s="http://www.w3.org/2003/05/soap-envelope"
            xmlns:a="http://schemas.xmlsoap.org/ws/2004/08/addressing"
            xmlns:d="http://schemas.xmlsoap.org/ws/2005/04/discovery"
            xmlns:dn="http://www.onvif.org/ver10/network/wsdl">
  <s:Header>
    <a:Action>http://schemas.xmlsoap.org/ws/2005/04/discovery/Probe</a:Action>
    <a:MessageID>uuid:%s</a:MessageID>
    <a:ReplyTo>
      <a:Address>http://schemas.xmlsoap.org/ws/2004/08/addressing/role/anonymous</a:Address>
    </a:ReplyTo>
    <a:To>urn:schemas-xmlsoap-org:ws:2005:04:discovery</a:To>
  </s:Header>
  <s:Body>
    <d:Probe>
      <d:Types>dn:NetworkVideoTransmitter</d:Types>
    </d:Probe>
  </s:Body>
</s:Envelope>`

// Discover sends a WS-Discovery probe on all suitable network interfaces
// and returns any ONVIF cameras found on the LAN.
func Discover() ([]Camera, error) {
	msg := fmt.Sprintf(probeTemplate, uuid.New().String())

	multicast, err := net.ResolveUDPAddr("udp4", multicastAddr)
	if err != nil {
		return nil, fmt.Errorf("resolve multicast addr: %w", err)
	}

	localIPs, err := getLANIPs()
	if err != nil {
		return nil, fmt.Errorf("get LAN IPs: %w", err)
	}
	if len(localIPs) == 0 {
		return nil, fmt.Errorf("no suitable network interfaces found")
	}

	seen := make(map[string]bool)
	var cameras []Camera
	var lastErr error

	for _, ip := range localIPs {
		conn, err := net.ListenUDP("udp4", &net.UDPAddr{IP: ip, Port: 0})
		if err != nil {
			lastErr = fmt.Errorf("listen on %s: %w", ip, err)
			continue
		}

		if err := conn.SetDeadline(time.Now().Add(probeTimeout)); err != nil {
			_ = conn.Close()
			lastErr = fmt.Errorf("set deadline on %s: %w", ip, err)
			continue
		}

		if _, err := conn.WriteToUDP([]byte(msg), multicast); err != nil {
			_ = conn.Close()
			lastErr = fmt.Errorf("send probe on %s: %w", ip, err)
			continue
		}

		found, err := readResponses(conn)
		_ = conn.Close()
		if err != nil {
			lastErr = fmt.Errorf("read responses on %s: %w", ip, err)
			continue
		}

		for _, cam := range found {
			if !seen[cam.XAddr] {
				seen[cam.XAddr] = true
				cameras = append(cameras, cam)
			}
		}
	}

	if len(cameras) == 0 && lastErr != nil {
		return nil, lastErr
	}

	return cameras, nil
}

// getLANIPs returns all IPv4 addresses on up, non-loopback, multicast interfaces.
func getLANIPs() ([]net.IP, error) {
	ifaces, err := net.Interfaces()
	if err != nil {
		return nil, err
	}

	var ips []net.IP
	for _, iface := range ifaces {
		if iface.Flags&net.FlagUp == 0 || iface.Flags&net.FlagLoopback != 0 {
			continue
		}
		if iface.Flags&net.FlagMulticast == 0 {
			continue
		}

		addrs, err := iface.Addrs()
		if err != nil {
			continue
		}

		for _, a := range addrs {
			ipNet, ok := a.(*net.IPNet)
			if !ok || ipNet.IP.To4() == nil {
				continue
			}
			ips = append(ips, ipNet.IP)
		}
	}

	return ips, nil
}

// ProbeAddress sends an ONVIF probe directly to a specific IP address,
// used as a fallback when multicast discovery doesn't find the camera.
func ProbeAddress(ip string) ([]Camera, error) {
	msg := fmt.Sprintf(probeTemplate, uuid.New().String())

	target := net.JoinHostPort(ip, "3702")
	addr, err := net.ResolveUDPAddr("udp4", target)
	if err != nil {
		return nil, fmt.Errorf("resolve addr: %w", err)
	}

	conn, err := net.DialUDP("udp4", nil, addr)
	if err != nil {
		return nil, fmt.Errorf("dial: %w", err)
	}
	defer func() { _ = conn.Close() }()

	if err := conn.SetDeadline(time.Now().Add(probeTimeout)); err != nil {
		return nil, fmt.Errorf("set deadline: %w", err)
	}

	if _, err := conn.Write([]byte(msg)); err != nil {
		return nil, fmt.Errorf("send probe: %w", err)
	}

	return readResponses(conn)
}

func readResponses(conn *net.UDPConn) ([]Camera, error) {
	var cameras []Camera
	buf := make([]byte, 65535)

	for {
		n, _, err := conn.ReadFromUDP(buf)
		if err != nil {
			if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
				break
			}
			return cameras, err
		}

		cam, err := parseProbeMatch(buf[:n])
		if err != nil {
			continue
		}
		cameras = append(cameras, cam...)
	}

	return cameras, nil
}

func parseProbeMatch(data []byte) ([]Camera, error) {
	var resp probeMatch
	if err := xml.Unmarshal(data, &resp); err != nil {
		return nil, err
	}

	var cameras []Camera
	for _, match := range resp.Body.ProbeMatches.Matches {
		xaddrs := strings.Fields(match.XAddrs)
		for _, xaddr := range xaddrs {
			u, err := url.Parse(xaddr)
			if err != nil {
				continue
			}

			// Skip IPv6 link-local and non-IPv4 addresses
			host := u.Hostname()
			if ip := net.ParseIP(host); ip != nil && ip.To4() == nil {
				continue
			}

			port := u.Port()
			if port == "" {
				port = "80"
			}

			name := extractNameFromScopes(match.Scopes)
			if name == "" {
				name = host
			}

			cameras = append(cameras, Camera{
				Name:  name,
				IP:    u.Hostname(),
				Port:  port,
				XAddr: xaddr,
			})
		}
	}

	return cameras, nil
}

func extractNameFromScopes(scopes string) string {
	for _, scope := range strings.Fields(scopes) {
		if strings.Contains(scope, "onvif://www.onvif.org/name/") {
			parts := strings.SplitAfter(scope, "/name/")
			if len(parts) == 2 {
				name, err := url.PathUnescape(parts[1])
				if err != nil {
					return parts[1]
				}
				return name
			}
		}
	}
	return ""
}
