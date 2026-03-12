package config

import "time"

type Config struct {
	BrasilAPIURL string
	ViaCEPURL    string
	Timeout      time.Duration
	Port         string
}

func NewDefaultConfig() *Config {
	return &Config{
		BrasilAPIURL: "https://brasilapi.com.br/api/cep/v1/%s",
		ViaCEPURL:    "http://viacep.com.br/ws/%s/json/",
		Timeout:      1 * time.Second,
		Port:         ":8080",
	}
}
