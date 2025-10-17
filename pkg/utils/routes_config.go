package utils

import (
	"errors"
	"fmt"
	"regexp"
)

type RouteType string

const (
	TCP   = RouteType("tcp")
	UDP   = RouteType("udp")
	HTTP  = RouteType("http")
	HTTPS = RouteType("https")
)

type ServiceConfig struct {
	Name    string        `yaml:"name" json:"name"`
	Enabled bool          `yaml:"enabled" json:"enabled"`
	Routes  []RouteConfig `yaml:"routes" json:"routes"`
}

type ProxyRule struct {
	Path       string `yaml:"path" json:"path"`
	TargetHost string `yaml:"target_host,omitempty" json:"target_host,omitempty"`
	TargetPort int    `yaml:"target_port,omitempty" json:"target_port,omitempty"`
	Rewrite    string `yaml:"rewrite,omitempty" json:"rewrite,omitempty"`
	StripPath  bool   `yaml:"strip_path,omitempty" json:"strip_path,omitempty"`
}

type ProxyHeaders struct {
	Add    map[string]string `yaml:"add,omitempty" json:"add,omitempty"`
	Remove []string          `yaml:"remove,omitempty" json:"remove,omitempty"`
	Set    map[string]string `yaml:"set,omitempty" json:"set,omitempty"`
}

type ProxySettings struct {
	Timeout         int           `yaml:"timeout,omitempty" json:"timeout,omitempty"`
	RetryAttempts   int           `yaml:"retry_attempts,omitempty" json:"retry_attempts,omitempty"`
	BufferRequests  bool          `yaml:"buffer_requests,omitempty" json:"buffer_requests,omitempty"`
	PreserveHost    bool          `yaml:"preserve_host,omitempty" json:"preserve_host,omitempty"`
	FollowRedirects bool          `yaml:"follow_redirects,omitempty" json:"follow_redirects,omitempty"`
	CustomHeaders   *ProxyHeaders `yaml:"custom_headers,omitempty" json:"custom_headers,omitempty"`
	Rules           []ProxyRule   `yaml:"rules,omitempty" json:"rules,omitempty"`
}

type RouteConfig struct {
	Type          RouteType      `yaml:"type" json:"type"`
	Private       bool           `yaml:"private" json:"private,omitempty"`
	BotProtect    bool           `yaml:"bot_protect" json:"bot_protect,omitempty"`
	Domain        string         `yaml:"domain,omitempty" json:"domain,omitempty"`
	Port          int            `yaml:"port,omitempty" json:"port,omitempty"`
	Machine       Machine        `yaml:"machine" json:"machine"`
	ProxySettings *ProxySettings `yaml:"proxy_settings,omitempty" json:"proxy_settings,omitempty"`
}

type Machine struct {
	NodeName string `yaml:"node" json:"node,omitempty"`
	Address  string `yaml:"address" json:"address"`
	Port     uint16 `yaml:"port" json:"port"`
}

func RouteComparison(v1, v2 RouteConfig) bool {
	if v1.Type != v2.Type {
		return false
	}
	if (v1.Machine.Address) != v2.Machine.Address {
		return false
	}
	if (v1.Machine.Port) != v2.Machine.Port {
		return false
	}
	switch v1.Type {
	case HTTP, HTTPS:
		if v1.Domain != v2.Domain {
			return false
		}
	case TCP, UDP:
		if v1.Port != v2.Port {
			return false
		}
	}
	return true
}

func ValidatePort(port int) error {
	if port < 0 || port > 65535 {
		return errors.New("invalid port: must be between 0 and 65535")
	}
	return nil
}

func ValidateHostname(hostname string) error {
	if len(hostname) > 255 {
		return errors.New("hostname is too long: must not exceed 255 characters")
	}

	// Regular expression for hostname validation
	hostnameRegex := `^([a-zA-Z0-9]([a-zA-Z0-9\-]{0,61}[a-zA-Z0-9])?\.)*[a-zA-Z0-9]([a-zA-Z0-9\-]{0,61}[a-zA-Z0-9])?$`
	matched, err := regexp.MatchString(hostnameRegex, hostname)
	if err != nil {
		return errors.New("error while validating hostname")
	}
	if !matched {
		return errors.New("invalid hostname format")
	}
	return nil
}

// ValidateDomain validates a domain name (with stricter rules).
func ValidateDomain(domain string) error {
	if len(domain) > 253 {
		return errors.New("domain is too long: must not exceed 253 characters")
	}

	if domain == "localhost" {
		return nil
	}
	// Regular expression for domain validation
	domainRegex := `^([a-zA-Z0-9]([a-zA-Z0-9\-]{0,61}[a-zA-Z0-9])?\.)+[a-zA-Z]{2,}$`
	matched, err := regexp.MatchString(domainRegex, domain)
	if err != nil {
		return errors.New("error while validating domain")
	}
	if !matched {
		return errors.New("invalid domain format")
	}
	return nil
}

func (cfg ServiceConfig) validate() error {
	for _, route := range cfg.Routes {
		if len(route.Machine.Address) == 0 {
			return fmt.Errorf("invalid config for route %s missing tailscale `machine.address`", cfg.Name)
		} else if err := ValidateHostname(route.Machine.Address); err != nil {
			return fmt.Errorf("invalid config for route %s `machine.address` %w", cfg.Name, err)
		}
		if (route.Machine.Port) == 0 {
			return fmt.Errorf("invalid config for route %s missing tailscale `machine.port`", cfg.Name)
		} else if err := ValidatePort(int(route.Machine.Port)); err != nil {
			return fmt.Errorf("invalid config for route %s `machine.port` %w", cfg.Name, err)
		}
		switch route.Type {
		case HTTP, HTTPS:
			if len(route.Domain) == 0 {
				return fmt.Errorf("invalid config for route %s missing `domain`", cfg.Name)
			} else if err := ValidateDomain(route.Domain); err != nil {
				return fmt.Errorf("invalid config for route %s `domian` %w", cfg.Name, err)
			}

		case TCP, UDP:
			if route.Port == 0 {
				return fmt.Errorf("invalid config for route %s missing `port`", cfg.Name)
			} else if err := ValidatePort(int(route.Port)); err != nil {
				return fmt.Errorf("invalid config for route %s `port` %w", cfg.Name, err)
			}
		default:
			return fmt.Errorf("invalid config for route %s missing or invalid `type` choose between [http,https,tcp,udp]", cfg.Name)
		}
	}
	return nil
}
