package config

import (
	"fmt"
	"github.com/go-playground/validator/v10"
	"github.com/spf13/viper"
	"log"
	"os"
	"reflect"
	"strings"
)

type Config struct {
	Connections Connections `mapstructure:"connections" validate:"required"`
	Profiling   Profiling   `mapstructure:"profiling" validate:"required"`
	Rules       []Rule      `mapstructure:"rules" validate:"required"`
}

type Profiling struct {
	// ProfilingInterval is how frequently sessions are profiled in seconds.
	ProfilingInterval *int `mapstructure:"interval" validate:"required"`
}

type Connections struct {
	Redis    Redis    `mapstructure:"redis" validate:"required"`
	InfluxDB InfluxDB `mapstructure:"influxdb" validate:"required"`
}

// Redis represents the key-value store of session cookies. Currently, only the
// Redis driver is available.
type Redis struct {
	Addr     *string `mapstructure:"addr" validate:"required"`
	Password *string `mapstructure:"pass" validate:"required"`
	StoreDB  *int    `mapstructure:"storeDB" validate:"required"`
	// QueueDB is the store for the session ID queue. Currently shared with
	// Redis above.
	QueueDB *int `mapstructure:"queueDB" validate:"required"`
}

// InfluxDB represents the store for the session browsing history. Currently,
// only InfluxDB is available.
type InfluxDB struct {
	Addr          *string `mapstructure:"addr"  validate:"required"`
	Token         *string `mapstructure:"token"  validate:"required"`
	Org           *string `mapstructure:"org"  validate:"required"`
	SessionBucket *string `mapstructure:"sessionBucket"  validate:"required"`
	// LoggingBucket is the store for logging output. Currently shared
	// with InfluxDB above.
	LoggingBucket *string `mapstructure:"loggingBucket"  validate:"required"`
}

type Rule struct {
	Description *string `mapstructure:"description" validate:"required"`
	Method      MatchableMethod
	Path        *string `mapstructure:"path" validate:"required"`
	Occurrences *int    `mapstructure:"occurrences" validate:"required"`
	Result      *string `mapstructure:"result" validate:"oneof=low high"`
}

type MatchableMethod struct {
	ShouldMatchAll *bool `mapstructure:"shouldMatchAll" validate:"required_without=Method"`
	// Method must be set if ShouldMatchAll is false. If ShouldMatchAll is true,
	// Method is ignored.
	Method *string `mapstructure:"method" validate:"required_without=ShouldMatchAll"`
}

func setDefaults() {
	viper.SetDefault("Profiling.Interval", 10)
}

func ReadConfig() *Config {
	// Dots are not valid identifiers for environment variables.
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	viper.AutomaticEnv()
	setDefaults()

	viper.SetConfigType("yaml")
	viper.SetConfigName("config")
	viper.AddConfigPath(".")
	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); ok {
			log.Fatalf("error: /app/config.yaml not found. Are you sure you have configured the ConfigMap?\nerr = %s", err)
		} else {
			log.Fatalf("error when reading config file at /app/config.yaml: err = %s", err)
		}
	}

	var config Config
	bindEnvs(config)
	if err := viper.Unmarshal(&config); err != nil {
		log.Fatalf("error occurred while reading configuration file: err = %s", err)
	}

	validate := validator.New()
	err := validate.Struct(&config)
	if err != nil {
		if _, ok := err.(*validator.InvalidValidationError); ok {
			log.Printf("unable to validate config: err = %s", err)
		}

		log.Printf("encountered validation errors:\n")

		for _, err := range err.(validator.ValidationErrors) {
			fmt.Printf("\t%s\n", err.Error())
		}

		fmt.Println("Check your configuration file and try again.")
		os.Exit(1)
	}

	return &config
}

// bindEnvs binds all environment variables automatically.
// See: https://github.com/spf13/viper/issues/188#issuecomment-399884438
func bindEnvs(iface interface{}, parts ...string) {
	ifv := reflect.ValueOf(iface)
	ift := reflect.TypeOf(iface)
	for i := 0; i < ift.NumField(); i++ {
		v := ifv.Field(i)
		t := ift.Field(i)
		tv, ok := t.Tag.Lookup("mapstructure")
		if !ok {
			continue
		}
		switch v.Kind() {
		case reflect.Struct:
			bindEnvs(v.Interface(), append(parts, tv)...)
		default:
			_ = viper.BindEnv(strings.Join(append(parts, tv), "."))
		}
	}
}
