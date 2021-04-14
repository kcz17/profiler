package main

import (
	"fmt"
	influxdb2 "github.com/influxdata/influxdb-client-go/v2"
	"github.com/influxdata/influxdb-client-go/v2/api"
	"github.com/kcz17/profiler/priority"
	"time"
)

// Profiler only has one action: to process user history in the implementation's
// data store, returning priorities for sessions which will ultimately be set as
// cookies on the front-end.
type Profiler interface {
	Process() map[string]priority.Priority
}

type InfluxDBProfiler struct {
	client      influxdb2.Client
	asyncWriter api.WriteAPI
}

func NewInfluxDBProfiler(addr, authToken, org, bucket string) *InfluxDBProfiler {
	options := influxdb2.DefaultOptions()
	client := influxdb2.NewClientWithOptions(addr, authToken, options)
	writeAPI := client.WriteAPI(org, bucket)

	// Create a goroutine for reading and logging async write errors.
	errorsCh := writeAPI.Errors()
	go func() {
		for err := range errorsCh {
			fmt.Printf("[%s] influxdb2 async write error: %v\n", time.Now().Format(time.StampMilli), err)
		}
	}()

	return &InfluxDBProfiler{
		client:      client,
		asyncWriter: writeAPI,
	}
}

func (p *InfluxDBProfiler) Process() map[string]priority.Priority {
	panic("to be implemented")
}
