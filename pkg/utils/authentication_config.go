package utils

import "fmt"

type OIDCProvider struct {
	Name         string `yaml:"name"`
	ClientID     string `yaml:"client_id"`
	ClientSecret string `yaml:"client_secret"`
	IssuerURL    string `yaml:"issuer_url"`
	RedirectURL  string `yaml:"redirect_url,omitempty"`
}

func (oidc *OIDCProvider) validate() error {
	if oidc.ClientID == "" {
		return fmt.Errorf("client_id is required for OIDC provider")
	}
	if oidc.IssuerURL == "" {
		return fmt.Errorf("issuer_url is required for OIDC provider")
	}
	return nil
}

type BasicProvider struct {
	Email string `yaml:"email,omitempty"`
}

func (basic *BasicProvider) validate() error {
	if basic.Email == "" {
		return fmt.Errorf("email is required for basic provider")
	}
	return nil
}

type AuthenticationProvider struct {
	OIDC  *OIDCProvider  `yaml:"oidc,omitempty"`
	Basic *BasicProvider `yaml:"basic,omitempty"`
}

func (provider *AuthenticationProvider) validate() error {
	if provider.OIDC != nil {
		return provider.OIDC.validate()
	}
	if provider.Basic != nil {
		return provider.Basic.validate()
	}
	return fmt.Errorf("no valid authentication provider configured")
}

type AuthenticationConfig struct {
	BaseURL       string                 `yaml:"baseURL"`
	SessionSecret string                 `yaml:"session_secret"`
	Provider      AuthenticationProvider `yaml:"provider"`
}
