package main

import (
	"fmt"
	"github.com/adjust/rmq/v3"
	"github.com/go-redis/redis/v7"
	"github.com/kcz17/profiler/config"
	"github.com/kcz17/profiler/priority"
	"github.com/kcz17/profiler/prioritystore"
	"log"
	"time"
)

const prefetchLimit = 5
const pollDuration = 100 * time.Millisecond

const RedisQueueTag = "profiler service"
const RedisQueueName = "sessions"

func main() {
	conf := config.ReadConfig()

	orderedRules := configRulesToOrderedRules(conf.Rules)

	logger := NewInfluxDBLogger(
		*conf.Connections.InfluxDB.Addr,
		*conf.Connections.InfluxDB.Token,
		*conf.Connections.InfluxDB.Org,
		*conf.Connections.InfluxDB.LoggingBucket,
	)
	priorityStore := prioritystore.NewRedisStore(
		*conf.Connections.Redis.Addr,
		*conf.Connections.Redis.Password,
		*conf.Connections.Redis.StoreDB,
	)
	profiler := NewInfluxDBProfiler(
		orderedRules,
		*conf.Connections.InfluxDB.Addr,
		*conf.Connections.InfluxDB.Token,
		*conf.Connections.InfluxDB.Org,
		*conf.Connections.InfluxDB.SessionBucket,
	)

	// Set up a Redis queue to process incoming session IDs.
	errChan := make(chan error, 10)
	go logErrors(errChan)
	connection, err := rmq.OpenConnectionWithRedisClient(RedisQueueTag, redis.NewClient(&redis.Options{
		Addr:     *conf.Connections.Redis.Addr,
		Password: *conf.Connections.Redis.Password,
		DB:       *conf.Connections.Redis.QueueDB,
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

		profiledPriority, err := profiler.Profile(sessionID)
		if err != nil {
			log.Printf("unexpected error when profiling session ID %s; err = %s", sessionID, err)
			return
		}

		logger.LogProfile(sessionID, profiledPriority)
		log.Printf("assigned priority %s to session ID %s", profiledPriority.String(), sessionID)
		if err := priorityStore.Set(sessionID, profiledPriority); err != nil {
			log.Printf("unexpected error when setting priority %s for session ID %s; err = %s", profiledPriority.String(), sessionID, err)
			if err := delivery.Reject(); err != nil {
				log.Printf("unable to reject delivery; err = %s", err)
			}
			return
		}

		if err := delivery.Ack(); err != nil {
			log.Printf("unable to ack delivery for session ID %s; err = %s", sessionID, err)
		}
	})
	if err != nil {
		panic(fmt.Errorf("unable to add consumer func: %w", err))
	}

	log.Printf("Profiler started with following rules:\n%s", orderedRules.String())

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

func configRulesToOrderedRules(configRules []config.Rule) OrderedRules {
	var rules OrderedRules

	for _, configRule := range configRules {
		rule := Rule{
			Description: *configRule.Description,
			Path:        *configRule.Path,
			Occurrences: *configRule.Occurrences,
		}

		if configRule.Method.ShouldMatchAll != nil && *configRule.Method.ShouldMatchAll {
			rule.Method.ShouldMatchAll = true
		} else {
			rule.Method.ShouldMatchAll = false
			rule.Method.Method = *configRule.Method.Method
		}

		if *configRule.Result == "high" {
			rule.Result = priority.High
		} else {
			rule.Result = priority.Low
		}

		rules = append(rules, rule)
	}

	return rules
}
