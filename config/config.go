package config

import (
	"github.com/spf13/viper"
	"log"
)

type Config struct {
	Connections Connections `mapstructure:"connections" validate:"required"`
	Profiling   Profiling   `mapstructure:"profiling" validate:"required"`
	Rules       []Rule      `mapstructure:"rules" validate:"required"`
}

type Profiling struct {
	// ProfilingInterval is how frequently sessions are profiled in seconds.
	ProfilingInterval int `mapstructure:"interval" validate:"required"`
}

type Connections struct {
	Redis    Redis    `mapstructure:"redis"`
	InfluxDB InfluxDB `mapstructure:"influxdb"`
}

// Redis represents the key-value store of session cookies. Currently, only the
// Redis driver is available.
type Redis struct {
	Addr     string `mapstructure:"addr" validate:"required"`
	Password string `mapstructure:"pass" validate:"required"`
	StoreDB  int    `mapstructure:"storeDB" validate:"required"`
	// RedisQueueDBis the store for the session ID queue. Currently shared with
	// Redis above.
	QueueDB int `mapstructure:"queueDB" validate:"required"`
}

// InfluxDB represents the store for the session browsing history. Currently,
// only InfluxDB is available.
type InfluxDB struct {
	Addr          string `mapstructure:"addr"  validate:"required"`
	Token         string `mapstructure:"token"  validate:"required"`
	Org           string `mapstructure:"org"  validate:"required"`
	SessionBucket string `mapstructure:"sessionBucket"  validate:"required"`
	// InfluxDBLoggingBucket is the store for logging output. Currently shared
	// with InfluxDB above.
	LoggingBucket string `mapstructure:"loggingBucket"  validate:"required"`
}

type Rule struct {
	Description string `mapstructure:"description" validate:"required"`
	Method      MatchableMethod
	Path        string `mapstructure:"path" validate:"required"`
	Occurrences int    `mapstructure:"occurrences" validate:"required"`
	Result      string `mapstructure:"result" validate:"oneof=low high"`
}

type MatchableMethod struct {
	ShouldMatchAll bool `mapstructure:"sholdMatchAll" validate:"required_without=Method"`
	// Method must be set if ShouldMatchAll is false. If ShouldMatchAll is true,
	// Method is ignored.
	Method string `mapstructure:"method" validate:"required_without=ShouldMatchAll"`
}

func ReadConfig() *Config {
	viper.SetDefault("profiling.interval", 10)
	viper.SetDefault("rules", []Rule{})

	viper.AutomaticEnv()
	viper.SetConfigName("config.yaml")
	viper.AddConfigPath("/app")
	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); ok {
			log.Fatal("error: /app/config.yaml not found. Are you sure you have configured the ConfigMap?")
		} else {
			log.Fatalf("error when reading config file at /app/config.yaml: err = %s", err)
		}
	}

	var config Config
	if err := viper.Unmarshal(&config); err != nil {
		log.Fatalf("error occured while reading configuration file: err = %s", err)
	}

	return &config
}
