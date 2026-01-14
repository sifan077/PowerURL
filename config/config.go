package config

import (
	"fmt"
	"os"
	"strings"

	"github.com/joho/godotenv"
	"github.com/spf13/viper"
)

type Config struct {
	// PostgreSQL
	Postgres PostgresConfig `mapstructure:"postgres"`

	// Redis
	Redis RedisConfig `mapstructure:"redis"`

	// NATS
	NATS NATSConfig `mapstructure:"nats"`

	// Prometheus
	Prometheus PrometheusConfig `mapstructure:"prometheus"`

	// Grafana
	Grafana GrafanaConfig `mapstructure:"grafana"`
}

type PostgresConfig struct {
	Host     string `mapstructure:"host"`
	User     string `mapstructure:"user"`
	Password string `mapstructure:"password"`
	Database string `mapstructure:"database"`
	Port     int    `mapstructure:"port"`
	SSLMode  string `mapstructure:"sslmode"`
}

type RedisConfig struct {
	Host     string `mapstructure:"host"`
	Port     int    `mapstructure:"port"`
	Password string `mapstructure:"password"`
	DB       int    `mapstructure:"db"`
}

type NATSConfig struct {
	Host        string `mapstructure:"host"`
	Port        int    `mapstructure:"port"`
	User        string `mapstructure:"user"`
	Password    string `mapstructure:"password"`
	MonitorPort int    `mapstructure:"monitor_port"`
}

type PrometheusConfig struct {
	Port           int    `mapstructure:"port"`
	Retention      string `mapstructure:"retention"`
	ScrapeInterval string `mapstructure:"scrape_interval"`
	Target         string `mapstructure:"target"`
}

type GrafanaConfig struct {
	Port          int    `mapstructure:"port"`
	AdminUser     string `mapstructure:"admin_user"`
	AdminPassword string `mapstructure:"admin_password"`
}

func Load() (*Config, error) {
	// Load local .env for development (ignored when missing).
	if err := godotenv.Load(".env"); err != nil && !os.IsNotExist(err) {
		return nil, fmt.Errorf("failed to load .env file: %w", err)
	}

	v := viper.New()

	// Search for config/config.yaml (plus root for overrides).
	v.SetConfigName("config")
	v.SetConfigType("yaml")
	v.AddConfigPath(".")
	v.AddConfigPath("./config")

	// Allow environment variables to override YAML entries.
	v.SetEnvPrefix("")
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	v.AutomaticEnv()

	if err := v.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return nil, fmt.Errorf("failed to read config file: %w", err)
		}
	}

	// Preserve legacy env variable names.
	bindEnvVars(v)

	var cfg Config
	if err := v.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	return &cfg, nil
}

func bindEnvVars(v *viper.Viper) {
	// PostgreSQL
	v.BindEnv("postgres.host", "PG_HOST")
	v.BindEnv("postgres.user", "PG_USER")
	v.BindEnv("postgres.password", "PG_PASSWORD")
	v.BindEnv("postgres.database", "PG_DB")
	v.BindEnv("postgres.port", "PG_PORT")
	v.BindEnv("postgres.sslmode", "PG_SSLMODE")

	// Redis
	v.BindEnv("redis.host", "REDIS_HOST")
	v.BindEnv("redis.port", "REDIS_PORT")
	v.BindEnv("redis.password", "REDIS_PASSWORD")
	v.BindEnv("redis.db", "REDIS_DB")

	// NATS
	v.BindEnv("nats.host", "NATS_HOST")
	v.BindEnv("nats.port", "NATS_PORT")
	v.BindEnv("nats.user", "NATS_USER")
	v.BindEnv("nats.password", "NATS_PASSWORD")
	v.BindEnv("nats.monitor_port", "NATS_MONITOR_PORT")

	// Prometheus
	v.BindEnv("prometheus.port", "PROM_PORT")
	v.BindEnv("prometheus.retention", "PROM_RETENTION")
	v.BindEnv("prometheus.scrape_interval", "PROM_SCRAPE_INTERVAL")
	v.BindEnv("prometheus.target", "PROM_TARGET")

	// Grafana
	v.BindEnv("grafana.port", "GRAFANA_PORT")
	v.BindEnv("grafana.admin_user", "GF_SECURITY_ADMIN_USER")
	v.BindEnv("grafana.admin_password", "GF_SECURITY_ADMIN_PASSWORD")
}
