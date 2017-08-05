package util

import (
	"net"
	"strings"
	"fmt"
	"net/http"
)

// returns IP address
func ParseRemoteAddress(remote_addr string) (net.IP, error) {
	parts := strings.Split(remote_addr, ":")
	if len(parts) != 2 {
		return nil, fmt.Errorf("Invalid request address (colon count mismatch of %d)", len(parts) - 1)
	}
	ip := net.ParseIP(parts[0])
	if ip == nil {
		return nil, fmt.Errorf("Invalid request address (invalid IP of '%s')", parts[0])
	}
	return ip, nil
}

func ParseRemoteAddressFromRequest(req *http.Request) (net.IP, error) {
	return ParseRemoteAddress(req.RemoteAddr)
}
