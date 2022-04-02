package server

import (
	"fmt"
	"github.com/spf13/viper"
	"github.com/urfave/cli"
)

type Config struct {
	EnableTLS  bool   `json:"enable_tls" yaml:"enable-tls" mapstructure:"enable-tls"`
	Cert       string `json:"cert" yaml:"cert"`                                          // Cert CertDir
	Key        string `json:"key" yaml:"key"`                                            // Key KeyDir
	Addr       string `json:"addr" yaml:"addr"`                                          // Addr 52.33.220.110:443
	SecretLink string `json:"secret_link" yaml:"secret-link" mapstructure:"secret-link"` // SecretLink ws uri
	Upstream   string `json:"upstream" yaml:"upstream"`
}

func NewConfig(c *cli.Context) (config *Config) {
	v := viper.New()
	v.SetConfigFile(c.String("config"))
	if err := v.ReadInConfig(); err != nil {
		panic(fmt.Sprintf("fatal error config file: %s", err))
	}
	if err := v.Unmarshal(&config); err != nil {
		panic(fmt.Sprintf("fatal error unmarshal config: %s", err))
	}
	if config.EnableTLS && (len(config.Cert) == 0 || len(config.Key) == 0) {
		panic("cert and key must be provided when enable tls")
	}
	return
}
