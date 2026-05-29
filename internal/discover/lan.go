package discover

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"sync"
	"time"
)

// Candidate is a device discovered on the LAN.
type Candidate struct {
	Host         string `json:"host"`
	SSHReachable bool   `json:"ssh_reachable"`
	SetupServer  bool   `json:"setup_server"`
	SerialNumber string `json:"serial_number,omitempty"`
	DeviceID     string `json:"device_id,omitempty"`
	HWVersion    string `json:"hw_version,omitempty"`
}

type setupIdentity struct {
	DeviceID     string `json:"device_id"`
	SerialNumber string `json:"serial_number"`
	HWVersion    string `json:"hw_version"`
	Serial       string `json:"serial"`
}

// ScanOptions configures LAN discovery.
type ScanOptions struct {
	ManualHosts []string
	Timeout     time.Duration
	// FullSubnet scans the local /24 (slow). When false, only ManualHosts and defaultHosts are probed.
	FullSubnet bool
}

// Scan finds Raspberry Pi hosts on the local network.
func Scan(ctx context.Context, opts ScanOptions) []Candidate {
	timeout := opts.Timeout
	if timeout == 0 {
		timeout = 2 * time.Second
	}

	hosts := uniqueHosts(append(defaultHosts(), opts.ManualHosts...))
	if opts.FullSubnet {
		if subnet, err := localSubnetHosts(); err == nil {
			hosts = uniqueHosts(append(hosts, subnet...))
		}
	}

	var (
		mu   sync.Mutex
		out  []Candidate
		wg   sync.WaitGroup
		sem  = make(chan struct{}, 32)
	)

	for _, host := range hosts {
		host := host
		wg.Add(1)
		go func() {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()

			c := probeHost(ctx, host, timeout)
			if !c.SSHReachable && !c.SetupServer {
				return
			}
			mu.Lock()
			out = append(out, c)
			mu.Unlock()
		}()
	}
	wg.Wait()
	if out == nil {
		return []Candidate{}
	}
	return out
}

func defaultHosts() []string {
	return []string{"raspberrypi.local"}
}

func uniqueHosts(in []string) []string {
	seen := make(map[string]struct{})
	out := make([]string, 0, len(in))
	for _, h := range in {
		h = stringsTrim(h)
		if h == "" {
			continue
		}
		if _, ok := seen[h]; ok {
			continue
		}
		seen[h] = struct{}{}
		out = append(out, h)
	}
	return out
}

func stringsTrim(s string) string {
	for len(s) > 0 && (s[0] == ' ' || s[0] == '\t') {
		s = s[1:]
	}
	for len(s) > 0 && (s[len(s)-1] == ' ' || s[len(s)-1] == '\t') {
		s = s[:len(s)-1]
	}
	return s
}

func localSubnetHosts() ([]string, error) {
	ifaces, err := net.Interfaces()
	if err != nil {
		return nil, err
	}

	var hosts []string
	for _, iface := range ifaces {
		if iface.Flags&net.FlagUp == 0 || iface.Flags&net.FlagLoopback != 0 {
			continue
		}
		addrs, err := iface.Addrs()
		if err != nil {
			continue
		}
		for _, addr := range addrs {
			ipNet, ok := addr.(*net.IPNet)
			if !ok || ipNet.IP.To4() == nil {
				continue
			}
			ip := ipNet.IP.To4()
			mask := ipNet.Mask
			if ones, _ := mask.Size(); ones != 24 {
				continue
			}
			base := net.IPv4(ip[0], ip[1], ip[2], 0)
			for i := 1; i < 255; i++ {
				h := net.IPv4(base[0], base[1], base[2], byte(i))
				if h.Equal(ip) {
					continue
				}
				hosts = append(hosts, h.String())
			}
		}
	}
	return hosts, nil
}

func probeHost(ctx context.Context, host string, timeout time.Duration) Candidate {
	c := Candidate{Host: host}

	resolved := host
	if addrs, err := net.DefaultResolver.LookupHost(ctx, host); err == nil && len(addrs) > 0 {
		resolved = addrs[0]
	}

	c.SSHReachable = tcpOpen(resolved, "22", timeout)

	client := &http.Client{Timeout: timeout}
	url := fmt.Sprintf("http://%s:4444/setup/identity", resolved)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return c
	}
	resp, err := client.Do(req)
	if err != nil {
		return c
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return c
	}

	var id setupIdentity
	if err := json.NewDecoder(resp.Body).Decode(&id); err != nil {
		return c
	}
	c.SetupServer = true
	c.DeviceID = id.DeviceID
	c.SerialNumber = id.SerialNumber
	if c.SerialNumber == "" {
		c.SerialNumber = id.Serial
	}
	c.HWVersion = id.HWVersion
	return c
}

func tcpOpen(host, port string, timeout time.Duration) bool {
	conn, err := net.DialTimeout("tcp", net.JoinHostPort(host, port), timeout)
	if err != nil {
		return false
	}
	_ = conn.Close()
	return true
}

// ResolveHost resolves mDNS / hostname to an IP for SSH.
func ResolveHost(host string) (string, error) {
	addrs, err := net.LookupHost(host)
	if err != nil {
		return host, nil
	}
	if len(addrs) > 0 {
		return addrs[0], nil
	}
	return host, nil
}
