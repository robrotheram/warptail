package utils

type Loadbalancer struct {
	Name string `yaml:"name"`
}

type Certificate struct {
	Name       string `yaml:"name"`
	SecretName string `yaml:"secret_name"`
}

type Ingress struct {
	Name    string `yaml:"name"`
	Class   string `yaml:"class"`
	Service string `yaml:"service"`
}

type KubernetesConfig struct {
	Namespace    string       `yaml:"namespace"`
	Loadbalancer Loadbalancer `yaml:"loadbalancer"`
	Certificate  Certificate  `yaml:"certificate"`
	Ingress      Ingress      `yaml:"ingress"`
}
