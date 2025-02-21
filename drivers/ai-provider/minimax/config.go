package minimax

import (
	"fmt"

	"github.com/eolinker/eosc"
)

type Config struct {
	APIKey string `json:"minimax_api_key"`
}

func checkConfig(v interface{}) (*Config, error) {
	conf, ok := v.(*Config)
	if !ok {
		return nil, eosc.ErrorConfigType
	}
	if conf.APIKey == "" {
		return nil, fmt.Errorf("minimax_api_key is required")
	}
	return conf, nil
}
