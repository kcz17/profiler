package main

import (
	"fmt"
	"github.com/ilyakaznacheev/cleanenv"
	"github.com/kcz17/profiler/prioritystore"
	"log"
	"time"
)

type Config struct {
	// ProfilingInterval is how frequently sessions are profiled in seconds.
	ProfilingInterval int `env:"PROFILING_INTERVAL" env-default:"10"`

	///////////////////////////////////////////////////////////////////////////
	// Key-value store of session cookies. Currently, only Redis is available.
	///////////////////////////////////////////////////////////////////////////

	RedisAddr     string `env:"REDIS_ADDR"`
	RedisPassword string `env:"REDIS_PASSWORD"`
	RedisDB       int    `env:"REDIS_DB"`

	///////////////////////////////////////////////////////////////////////////
	// Store for session browsing history. Currently, only InfluxDB is available.
	///////////////////////////////////////////////////////////////////////////

	InfluxDBAddr   string `env:"INFLUXDB_ADDR"`
	InfluxDBToken  string `env:"INFLUXDB_TOKEN"`
	InfluxDBOrg    string `env:"INFLUXDB_ORG"`
	InfluxDBBucket string `env:"INFLUXDB_BUCKET"`
}

func main() {
	var config Config
	err := cleanenv.ReadEnv(&config)
	if err != nil {
		log.Fatalf("expected err == nil in envconfig.Process(); got err = %v", err)
	}

	priorityStore := prioritystore.NewRedisStore(config.RedisAddr, config.RedisPassword, config.RedisDB)
	profiler := NewInfluxDBProfiler(config.InfluxDBAddr, config.InfluxDBToken, config.InfluxDBOrg, config.InfluxDBBucket)

	for _ = range time.Tick(time.Duration(config.ProfilingInterval) * time.Second) {
		sessionPriorities := profiler.Process()
		for session, priority := range sessionPriorities {
			if err := priorityStore.Set(session, priority); err != nil {
				panic(fmt.Errorf("error encountered when setting priority %s for session %s: %w", priority.String(), session, err))
			}
		}
	}
}
