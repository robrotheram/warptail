package utils

import (
	"crypto/md5"
	"crypto/rand"
	"fmt"
	"log"
	"log/slog"
	"os"
	"reflect"

	"gopkg.in/yaml.v2"
)

type DashboardConfig struct {
	Enabled bool   `yaml:"enabled"`
	Token   string `yaml:"token"`
	Port    string `yaml:"port"`
}

type TailscaleConfig struct {
	AuthKey  string `yaml:"auth_key"`
	Hostname string `yaml:"hostnmae"`
}

type Config struct {
	Tailscale  TailscaleConfig  `yaml:"tailscale"`
	Dasboard   DashboardConfig  `yaml:"dashboard"`
	Kubernetes KubernetesConfig `yaml:"kubernetes,omitempty"`
	Services   []ServiceConfig  `yaml:"services"`
	Logging    LoggingConfig    `yaml:"logging"`
}

func ConfigHash(path string) [16]byte {
	data, err := os.ReadFile(path)
	if err != nil {
		log.Fatal(err)
	}
	return md5.Sum(data)
}

func LoadConfig(configPath string) Config {
	data, err := os.ReadFile(configPath)
	if err != nil {
		log.Fatalf("error: %v", err)
	}
	var config Config
	err = yaml.Unmarshal(data, &config)
	if err != nil {
		log.Fatalf("error: %v", err)
	}
	config.validate()
	setupLogger(config.Logging)
	return config
}

func (config *Config) validate() {
	if config.Dasboard.Enabled && config.Dasboard.Port == "" {
		config.Dasboard.Port = ":8081"
	}

	if config.Dasboard.Enabled && config.Dasboard.Token == "" {
		config.Dasboard.Token = tokenGenerator(24)
		slog.Info("New Dashboard Token", "token", config.Dasboard.Token)
	}

	if !IsEmptyStruct(config.Kubernetes) {
		if len(config.Kubernetes.Ingress.Name) == 0 {
			config.Kubernetes.Certificate.Name = "warptail-route-ingress"
			config.Kubernetes.Certificate.SecretName = "warptail-certificate"
		}
		if len(config.Kubernetes.Loadbalancer.Name) == 0 {
			config.Kubernetes.Certificate.Name = "warptail-route-loadbalancer"
		}
		if len(config.Kubernetes.Certificate.Name) == 0 {
			config.Kubernetes.Certificate.Name = "warptail-route-certificate"
		}
	}
}

func IsEmptyStruct(s interface{}) bool {
	return reflect.DeepEqual(s, reflect.Zero(reflect.TypeOf(s)).Interface())
}

func ContainsService(name string, configs []ServiceConfig) bool {
	for _, svc := range configs {
		if svc.Name == name {
			return true
		}
	}
	return false
}

func tokenGenerator(length int) string {
	b := make([]byte, length)
	rand.Read(b)
	return fmt.Sprintf("%x", b)
}
