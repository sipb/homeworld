package util

import (
	"testing"
	"net"
	"net/http"
	"encoding/json"
	"context"
	"io/ioutil"
	"fmt"
)

func TestParseSimpleAddresses(t *testing.T) {
	addresses := []struct {
		text     string
		expected net.IP
	}{
		{"127.0.0.1:80", net.IPv4(127, 0, 0, 1)},
		{"10.15.40.2:20557", net.IPv4(10, 15, 40, 2)},
		{"8.8.8.8:0", net.IPv4(8, 8, 8, 8)},
		{"[2001:0db8:85a3:0000:0000:8a2e:0370:7334]:443", net.IP{0x20, 0x01, 0x0d, 0xb8, 0x85, 0xa3, 0x00, 0x00, 0x00, 0x00, 0x8a, 0x2e, 0x03, 0x70, 0x73, 0x34}},
		{"[fe80::5]:443", net.IP{0xfe, 0x80, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 5}},
		{"[::1]:123", net.IP{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 1}},
	}
	for _, test := range addresses {
		address, err := ParseRemoteAddress(test.text)
		if err != nil {
			t.Errorf("Should not have gotten error %s for %s", err, test.text)
		}
		if !address.Equal(test.expected) {
			t.Errorf("Mismatched address received (%s instead of %s)", address, test.expected)
		}
	}
}

func TestInvalidAddresses(t *testing.T) {
	addresses := []string{
		"127.0.0:80",
		"10.15.40.2:",
		"8.8.8.8",
		":56",
		"[2001:0db8:85a3:0000:0000:8a2e:0370:7334]",
		"[2001:0db8:85a3:0000:0000:8a2e:0370:7334]:",
		"2001:0db8:85a3:0000:0000:8a2e:0370:7334:443",
		"[127.0.0.1]:123",
		"82",
		"homeworld.mit.edu",
		"homeworld.mit.edu:80",
		"[fe80:::145",
		"fe80::]:145",
		"10.15.40.2:123:456",
		"192.168.0.1:hello",
		"nothing:55",
	}
	for _, test := range addresses {
		_, err := ParseRemoteAddress(test)
		if err == nil {
			t.Errorf("Should have gotten error for %s", test)
		}
	}
}

func TestLocalHTTPConnection(t *testing.T) {
	testConnection("127.0.0.8:50158", net.IPv4(127, 0, 0, 1), t)
	testConnection("[::1]:50158", net.ParseIP("::1"), t)
}

func testConnection(listenAddr string, expectIP net.IP, t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/test", func(writer http.ResponseWriter, req *http.Request) {
		ip, err := ParseRemoteAddressFromRequest(req)
		if err != nil {
			t.Errorf("Could not parse remote address that should have been parsable (%s): %s", req.RemoteAddr, err)
		} else {
			data, err := json.Marshal(ip)
			if err != nil {
				t.Errorf("Should have been able to marshal IP")
			} else {
				writer.Write(data)
			}
		}
	})
	server := &http.Server{
		Addr:    listenAddr,
		Handler: mux,
	}
	listener, err := net.Listen("tcp", server.Addr)
	if err != nil {
		t.Fatal(err)
	}
	go func() {
		err := server.Serve(listener)
		if err != nil && err != http.ErrServerClosed {
			t.Error(err)
		}
	}()
	defer func() {
		err := server.Shutdown(context.Background())
		if err != nil {
			t.Error(err)
		}
	}()

	resp, err := http.Get(fmt.Sprintf("http://%s/test", listenAddr))
	if err != nil {
		t.Errorf("Cannot fetch test endpoint: %s", err)
	} else {
		defer resp.Body.Close()
		body, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			t.Error(err)
		} else if len(body) == 0 {
			t.Errorf("Failed to get result from request")
		} else {
			ip := &net.IP{}
			err := json.Unmarshal(body, ip)
			if err != nil {
				t.Error(err)
			}
			if !ip.Equal(expectIP) {
				t.Errorf("Mismatch on IP address %v", expectIP)
			}
		}
	}
}
