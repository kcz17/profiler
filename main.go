package main

import (
	"fmt"
	"github.com/adjust/rmq/v3"
	"github.com/go-redis/redis/v7"
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
	RedisStoreDB  int    `env:"REDIS_STORE_DB"`

	///////////////////////////////////////////////////////////////////////////
	// Store for session ID queue. Currently shared with Redis above.
	///////////////////////////////////////////////////////////////////////////

	RedisQueueDB int `env:"REDIS_QUEUE_DB"`

	///////////////////////////////////////////////////////////////////////////
	// Store for session browsing history. Currently, only InfluxDB is available.
	///////////////////////////////////////////////////////////////////////////

	InfluxDBAddr   string `env:"INFLUXDB_ADDR"`
	InfluxDBToken  string `env:"INFLUXDB_TOKEN"`
	InfluxDBOrg    string `env:"INFLUXDB_ORG"`
	InfluxDBBucket string `env:"INFLUXDB_BUCKET"`
}

const prefetchLimit = 5
const pollDuration = 100 * time.Millisecond

const RedisQueueTag = "profiler service"
const RedisQueueName = "sessions"

func main() {
	var config Config
	err := cleanenv.ReadEnv(&config)
	if err != nil {
		log.Fatalf("expected err == nil in envconfig.Process(); got err = %v", err)
	}

	priorityStore := prioritystore.NewRedisStore(config.RedisAddr, config.RedisPassword, config.RedisStoreDB)
	profiler := NewInfluxDBProfiler(config.InfluxDBAddr, config.InfluxDBToken, config.InfluxDBOrg, config.InfluxDBBucket)

	// Set up a Redis queue to process incoming session IDs.
	errChan := make(chan error, 10)
	go logErrors(errChan)
	connection, err := rmq.OpenConnectionWithRedisClient(RedisQueueTag, redis.NewClient(&redis.Options{
		Addr:     config.RedisAddr,
		Password: config.RedisPassword,
		DB:       config.RedisQueueDB,
	}), errChan)
	if err != nil {
		panic(fmt.Errorf("unable to open connection with redis client: %w", err))
	}

	queue, err := connection.OpenQueue(RedisQueueName)
	if err != nil {
		panic(fmt.Errorf("unable to open queue with redis client: %w", err))
	}

	if err := queue.StartConsuming(prefetchLimit, pollDuration); err != nil {
		panic(fmt.Errorf("unable to start consuming queue: %w", err))
	}

	_, err = queue.AddConsumerFunc(RedisQueueTag, func(delivery rmq.Delivery) {
		sessionID := delivery.Payload()
		log.Printf("incoming request for session ID %s", sessionID)
		priority := profiler.Profile(sessionID)
		log.Printf("assigned priority %s to session ID %s", priority.String(), sessionID)
		if err := priorityStore.Set(sessionID, priority); err != nil {
			log.Printf("unexpected error when setting priority %s for session ID %s; err = %s", priority.String(), sessionID, err)
			if err := delivery.Reject(); err != nil {
				log.Printf("unable to reject delivery; err = %s", err)
			}
			return
		}

		if err := delivery.Ack(); err != nil {
			log.Printf("unable to ack delivery; err = %s", err)
		}
	})
	if err != nil {
		panic(fmt.Errorf("unable to add consumer func: %w", err))
	}

	cleaner := rmq.NewCleaner(connection)
	for range time.Tick(time.Second) {
		_, err := cleaner.Clean()
		if err != nil {
			log.Printf("failed to clean: %s", err)
			continue
		}
	}
}

func logErrors(errChan <-chan error) {
	for err := range errChan {
		switch err := err.(type) {
		case *rmq.HeartbeatError:
			if err.Count == rmq.HeartbeatErrorLimit {
				log.Print("heartbeat error (limit): ", err)
			} else {
				log.Print("heartbeat error: ", err)
			}
		case *rmq.ConsumeError:
			log.Print("consume error: ", err)
		case *rmq.DeliveryError:
			log.Print("delivery error: ", err.Delivery, err)
		default:
			log.Print("other error: ", err)
		}
	}
}
