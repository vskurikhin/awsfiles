package config

import (
	"fmt"
	"log/slog"
	"net/url"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/vskurikhin/awsfiles/pkg/tool"
)

type Config struct {
	AccessKeyID        string `mapstructure:"access_key_id"`
	Address            string `mapstructure:"address"`
	Bucket             string `mapstructure:"bucket"`
	BufferSize         int    `mapstructure:"buffer_size"`
	CAFile             string `mapstructure:"ca_file"`
	Debug              bool   `mapstructure:"debug"`
	InsecureSkipVerify bool   `mapstructure:"insecure_skip_verify"`
	Key                string `mapstructure:"key"`
	S3Host             string `mapstructure:"s3_host"`
	SecretAccessKey    string `mapstructure:"secret_access_key"`
	ServerName         string `mapstructure:"server_name"`
	Size               int    `mapstructure:"size"`
	Verbose            bool   `mapstructure:"verbose"`
	ssl                bool
}

func MakeConfig(cmd *cobra.Command) Config {
	var cfg Config
	err := viper.Unmarshal(&cfg)
	if err != nil && tool.IsDebug(cmd) {
		slog.Error("Error binding flag", "error", err)
	}
	if cfg.S3Host != "" {
		var u *url.URL
		u, err = url.Parse(cfg.S3Host)
		if err != nil && tool.IsDebug(cmd) {
			slog.Error("Error binding flag", "error", err)
		}
		if u != nil && cfg.Address == "" {
			cfg.Address = u.Host
		}
		if u != nil && u.Scheme == "http" {
			if !strings.Contains(cfg.Address, ":") {
				cfg.Address = cfg.Address + ":80"
			}
		}
		if u != nil && u.Scheme == "https" {
			cfg.ssl = true
			if !strings.Contains(cfg.Address, ":") {
				cfg.Address = cfg.Address + ":443"
			}
			//goland:noinspection HttpUrlsUsage
			cfg.S3Host = "http://" + u.Host
		}
		if u != nil && cfg.ServerName == "" && len(u.Host) > 0 {
			a := strings.Split(u.Host, ":")
			if len(a) > 0 {
				cfg.ServerName = a[0]
			}
		}
	}
	if tool.IsDebug(cmd) {
		slog.Debug("variable:", "config", fmt.Sprintf("%+v", cfg))
	}
	return cfg
}

func (c Config) Ssl() bool {
	return c.ssl
}
