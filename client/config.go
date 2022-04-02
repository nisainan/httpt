package client

import (
	"fmt"
	"github.com/spf13/viper"
	"github.com/urfave/cli"
)

type Config struct {
	Mode       string `json:"mode" yaml:"mode"`
	LocalAddr  string `json:"local_addr" yaml:"local-addr" mapstructure:"local-addr"`    // Addr 52.33.220.110:443
	ServerAddr string `json:"server_addr" yaml:"server-addr" mapstructure:"server-addr"` // Addr 52.33.220.110:443
	Sni        string `json:"sni" yaml:"sni"`                                            // Sni sfrolov.io
	SecretLink string `json:"secret_link" yaml:"secret-link" mapstructure:"secret-link"` // SecretLink ws uri
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
	return
}
