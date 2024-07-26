package adapters

import (
	"encoding/json"
	"fmt"

	config "github.com/LamineKouissi/LHP/config"
	"github.com/xeipuuv/gojsonschema"
)

type jsonValidator struct {
	configData []byte
}

func NewjsonValidator(config []byte) (*jsonValidator, error) {
	return &jsonValidator{configData: config}, nil
}

func (jv *jsonValidator) ValidateConfig() (*config.ProxyConfig, error) {

	// Load the JSON schema
	schemaLoader := gojsonschema.NewStringLoader(`
		{
			"$schema": "http://json-schema.org/draft-07/schema#",
			"type": "object",
			"properties": {
				"listen_address": {
				"type": "string"
				},
				"tls_enabled": {
				"type": "boolean"
				},
				"tls_cert": {
				"type": "object",
				"properties": {
					"key": {
					"type": "string"
					},
					"crt": {
					"type": "string"
					}
				},
				"required": ["key", "crt"]
				},
				"tunnelling_enabled": {
				"type": "boolean"
				},
				"routes": {
				"type": "array",
				"items": {
					"type": "object",
					"properties": {
					"path": {
						"type": "string"
					},
					"method": {
						"type": "string",
						"enum": ["GET", "POST", "PUT", "DELETE"]
					},
					"filter_chain": {
						"type": "array",
						"items": {
						"type": "string"
						}
					},
					"connector": {
						"type": "string"
					}
					},
					"required": ["path", "method", "filter_chain", "connector"]
				}
				}
			},
			"required": ["listen_address", "tls_enabled", "tls_cert", "tunnelling_enabled", "routes"]
		}`)

	// Load the configuration data
	documentLoader := gojsonschema.NewBytesLoader(jv.configData)

	// Validate the configuration against the schema
	result, err := gojsonschema.Validate(schemaLoader, documentLoader)
	if err != nil {
		return nil, fmt.Errorf("error validating config: %v", err)
	}

	if !result.Valid() {
		// Collect and return validation errors
		var errors []string
		for _, desc := range result.Errors() {
			errors = append(errors, desc.String())
		}
		return nil, fmt.Errorf("config validation failed: %v", errors)
	}

	// Parse the configuration into the ProxyConfig struct
	var config config.ProxyConfig
	err = json.Unmarshal(jv.configData, &config)
	if err != nil {
		return nil, fmt.Errorf("error parsing config: %v", err)
	}

	return &config, nil
}
