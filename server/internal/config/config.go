package config

import (
	"os"
	"strings"

	"github.com/spf13/viper"
)

type Config struct {
	AppEnv string `mapstructure:"app_env"`
	HTTP   struct {
		Addr string `mapstructure:"addr"`
	} `mapstructure:"http"`
	Postgres struct {
		DSN string `mapstructure:"dsn"`
	} `mapstructure:"postgres"`
	Redis struct {
		Addr      string `mapstructure:"addr"`
		Namespace string `mapstructure:"namespace"`
	} `mapstructure:"redis"`
	Build struct {
		Version   string `mapstructure:"version"`
		GitCommit string `mapstructure:"git_commit"`
	} `mapstructure:"build"`
}

func Load(path string) (*Config, error) {
	v := viper.New()

	// Defaults
	v.SetDefault("app_env", "development")
	v.SetDefault("http.addr", ":8080")

	// Env vars
	v.AutomaticEnv()
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))

	// Explicit bindings for test compatibility
	_ = v.BindEnv("app_env", "APP_ENV")
	_ = v.BindEnv("http.addr", "HTTP_ADDR")
	_ = v.BindEnv("postgres.dsn", "DATABASE_DSN")
	_ = v.BindEnv("redis.addr", "REDIS_ADDR")
	_ = v.BindEnv("redis.namespace", "REDIS_NAMESPACE")
	_ = v.BindEnv("build.version", "APP_VERSION")
	_ = v.BindEnv("build.git_commit", "GIT_COMMIT")

	// Config file
	if path != "" {
		if _, err := os.Stat(path); err == nil {
			v.SetConfigFile(path)
			if err := v.ReadInConfig(); err != nil {
				return nil, err
			}
		}
	}

	var cfg Config
	if err := v.Unmarshal(&cfg); err != nil {
		return nil, err
	}

	return &cfg, nil
}
