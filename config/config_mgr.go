package config

import (
	"fmt"
	"io/ioutil"
)

type configValidator interface {
	ValidateConfig() (*ProxyConfig, error)
}

type configMgr struct {
	cv configValidator
}

func NewConfigMgr(confVld configValidator) (*configMgr, error) {
	return &configMgr{cv: confVld}, nil
}

func (cm *configMgr) LoadConfig(configPath string) ([]byte, error) {
	// Read the configuration file
	configData, err := ioutil.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("error reading config file: %v", err)
	}
	return configData, nil
}

// change ProxyConfig and the corresponding adapters to be config.format .(json, .yaml, etc) agnostic
type ProxyConfig struct {
	ListenAddress     string        `json:"listen_address"`
	TLSEnabled        bool          `json:"tls_enabled"`
	TLSCert           TLSCertConfig `json:"tls_cert"`
	TunnellingEnabled bool          `json:"tunnelling_enabled"`
	Routes            []RouteConfig `json:"routes"`
}

type TLSCertConfig struct {
	Key string `json:"key"`
	Crt string `json:"crt"`
}

type RouteConfig struct {
	Path        string   `json:"path"`
	Method      string   `json:"method"`
	FilterChain []string `json:"filter_chain"`
	Connector   string   `json:"connector"`
}
