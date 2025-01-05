package utils

import (
	"context"
	"crypto/md5"
	"log"
	"os"
	"reflect"
	"warptail/pkg/migrations"

	"github.com/uptrace/bun"
	"github.com/uptrace/bun/migrate"
	"gopkg.in/yaml.v2"
)

type AuthenticationProvider struct {
	Name        string `yaml:"name"`
	Type        string `yaml:"type"`
	ClientID    string `yaml:"clientID,omitempty"`
	ProviderURL string `yaml:"providerURL,omitempty"`
	BaseURL     string `yaml:"baseURL,omitempty"`
	Secret      string `yaml:"session_secret,omitempty"`
}

type AuthenticationConfig struct {
	BaseURL  string                 `yaml:"baseURL"`
	Secret   string                 `yaml:"secretKey"`
	Provider AuthenticationProvider `yaml:"provider"`
}

type TailscaleConfig struct {
	AuthKey  string `yaml:"auth_key"`
	Hostname string `yaml:"hostnmae"`
}

type Config struct {
	Tailscale   TailscaleConfig   `yaml:"tailscale"`
	Database    DatabaseConfig    `yaml:"database"`
	Application ApplicationConfig `yaml:"application"`
	Kubernetes  KubernetesConfig  `yaml:"kubernetes,omitempty"`
	Services    []ServiceConfig   `yaml:"services"`
	Logging     LoggingConfig     `yaml:"logging"`
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

func NewDB(config Config) *bun.DB {

	db, err := NewDatabase(config.Database)
	if err != nil {
		panic(err)
	}
	migrator := migrate.NewMigrator(db, migrations.Migrations)
	err = migrator.Init(context.Background())
	if err != nil {
		panic(err)
	}
	group, err := migrator.Migrate(context.Background())
	if err != nil {
		panic(err)
	}

	if group.ID == 0 {
		Logger.Info("there are no new migrations to run")
		return db
	}

	Logger.Info("migrated to", "group", group)

	return db
}
