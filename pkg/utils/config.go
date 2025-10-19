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

type TailscaleConfig struct {
	AuthKey  string `yaml:"auth_key"`
	Hostname string `yaml:"hostnmae"`
}

type Config struct {
	Tailscale          TailscaleConfig          `yaml:"tailscale"`
	Database           DatabaseConfig           `yaml:"database"`
	Application        ApplicationConfig        `yaml:"application"`
	Authentication     AuthenticationConfig     `yaml:"authentication"`
	CertificateManager CertificateManagerConfig `yaml:"acme,omitempty"`
	Kubernetes         KubernetesConfig         `yaml:"kubernetes,omitempty"`
	Services           []ServiceConfig          `yaml:"services"`
	Logging            LoggingConfig            `yaml:"logging"`
}

func (app *Config) UseHTTPS() bool {
	return !IsEmptyStruct(app.CertificateManager) && app.CertificateManager.Enabled
}

var ConfigPath = os.Getenv("CONFIG_PATH")

func init() {
	if len(ConfigPath) == 0 {
		ConfigPath = "config.yaml"
	}
}

func ConfigHash(path string) [16]byte {
	data, err := os.ReadFile(path)
	if err != nil {
		log.Fatal(err)
	}
	return md5.Sum(data)
}

func LoadConfig(configPath string) (Config, error) {
	data, err := os.ReadFile(configPath)
	if err != nil {
		log.Fatalf("error: %v", err)
	}
	var config Config
	err = yaml.Unmarshal(data, &config)
	if err != nil {
		log.Fatalf("error: %v", err)
	}
	setupLogger(config.Logging)
	if err := config.validate(); err != nil {
		return config, err
	}
	return config, nil
}

func (config *Config) validate() error {
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

	if err := config.Authentication.Provider.validate(); err != nil {
		return err
	}

	for _, svc := range config.Services {
		if err := svc.validate(); err != nil {
			return err
		}
	}
	return nil
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
