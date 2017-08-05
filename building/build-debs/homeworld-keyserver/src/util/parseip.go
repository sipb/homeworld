package util

import (
	"net"
	"strings"
	"fmt"
	"net/http"
	"strconv"
	"errors"
)

// returns IP address
func ParseRemoteAddress(remote_addr string) (net.IP, error) {
	index := strings.LastIndexByte(remote_addr, ':')
	if index == -1 {
		return nil, errors.New("Invalid request address (no colon)")
	}
	port, err := strconv.ParseUint(remote_addr[index+1:], 10, 16)
	if err != nil {
		return nil, err
	}
	if port < 0 || port > 65535 {
		return nil, fmt.Errorf("Invalid port %d", port)
	}
	var ip net.IP
	if remote_addr[0] == '[' && remote_addr[index-1] == ']' {
		ip = net.ParseIP(remote_addr[1:index-1])
		if ip.To4() != nil {
			return nil, fmt.Errorf("IP address was expected to be IPv6, but was IPv4")
		}
	} else {
		ip = net.ParseIP(remote_addr[:index])
		if ip.To4() == nil {
			return nil, fmt.Errorf("IP address was expected to be IPv4, but was IPv6")
		}
	}
	if ip == nil {
		return nil, fmt.Errorf("Invalid request address (invalid IP in '%s')", remote_addr)
	}
	return ip, nil
}

func ParseRemoteAddressFromRequest(req *http.Request) (net.IP, error) {
	return ParseRemoteAddress(req.RemoteAddr)
}
