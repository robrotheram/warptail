package utils

import (
	"gopkg.in/yaml.v2"
	"strings"
	"testing"
)

func TestTailscaleHostnameCompatibility(t *testing.T) {
	for _, input := range []string{"hostname: proxy", "hostnmae: proxy", "hostname: proxy\nhostnmae: old"} {
		var config TailscaleConfig
		if err := yaml.Unmarshal([]byte(input), &config); err != nil {
			t.Fatal(err)
		}
		if config.Hostname != "proxy" {
			t.Fatalf("hostname = %q", config.Hostname)
		}
		data, err := yaml.Marshal(config)
		if err != nil {
			t.Fatal(err)
		}
		if strings.Contains(string(data), "hostnmae") || !strings.Contains(string(data), "hostname: proxy") {
			t.Fatalf("saved settings: %s", data)
		}
	}
}

func TestMachineAddresses(t *testing.T) {
	for _, address := range []string{"100.64.0.1", "fd7a:115c:a1e0::1", "service.tailnet.ts.net"} {
		cfg := ServiceConfig{Name: "service", Routes: []RouteConfig{{Type: TCP, Port: 1234, Machine: Machine{Address: address, Port: 8080}}}}
		if err := cfg.validate(); err != nil {
			t.Errorf("%s: %v", address, err)
		}
	}
}
