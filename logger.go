package main

import (
	"fmt"
	influxdb2 "github.com/influxdata/influxdb-client-go/v2"
	"github.com/influxdata/influxdb-client-go/v2/api"
	"github.com/kcz17/profiler/priority"
	"time"
)

// influxDBLogger logs the output to an external InfluxDB instance.
type influxDBLogger struct {
	client      influxdb2.Client
	asyncWriter api.WriteAPI
}

func NewInfluxDBLogger(baseURL, authToken, org, bucket string) *influxDBLogger {
	options := influxdb2.DefaultOptions()
	options.WriteOptions().SetBatchSize(1000)
	options.WriteOptions().SetFlushInterval(250)

	client := influxdb2.NewClientWithOptions(baseURL, authToken, options)
	writeAPI := client.WriteAPI(org, bucket)

	// Create a goroutine for reading and logging async write errors.
	errorsCh := writeAPI.Errors()
	go func() {
		for err := range errorsCh {
			fmt.Printf("[%s] influxdb2 logging async write error: %v\n", time.Now().Format(time.StampMilli), err)
		}
	}()

	return &influxDBLogger{
		client:      client,
		asyncWriter: writeAPI,
	}
}

func (l *influxDBLogger) LogProfile(sessionID string, priority priority.Priority) {
	p := influxdb2.NewPointWithMeasurement("session_priority").
		AddTag("session_id", sessionID).
		AddField("priority", priority.String()).
		SetTime(time.Now())
	l.asyncWriter.WritePoint(p)
}
