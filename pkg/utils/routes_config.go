package utils

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

type RouteConfig struct {
	Type    RouteType `yaml:"type" json:"type"`
	Private bool      `yaml:"private" json:"private"`
	Domain  string    `yaml:"domain,omitempty" json:"domain,omitempty"`
	Port    int       `yaml:"port,omitempty" json:"port,omitempty"`
	Machine Machine   `yaml:"machine" json:"machine"`
}

type Machine struct {
	Address string `yaml:"address" json:"address"`
	Port    uint16 `yaml:"port" json:"port"`
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
