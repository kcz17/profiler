package main

import (
	influxdb2 "github.com/influxdata/influxdb-client-go/v2"
	"github.com/influxdata/influxdb-client-go/v2/api"
	"github.com/kcz17/profiler/priority"
)

// Profiler only has one action: to process user history in the implementation's
// data store, returning priorities for sessions which will ultimately be set as
// cookies on the front-end.
type Profiler interface {
	Process() map[string]priority.Priority
}

type InfluxDBProfiler struct {
	client influxdb2.Client
	query  api.QueryAPI
	bucket string
}

func NewInfluxDBProfiler(addr, authToken, org, bucket string) *InfluxDBProfiler {
	options := influxdb2.DefaultOptions()
	client := influxdb2.NewClientWithOptions(addr, authToken, options)
	queryAPI := client.QueryAPI(org)

	return &InfluxDBProfiler{
		client: client,
		query:  queryAPI,
		bucket: bucket,
	}
}

func (p *InfluxDBProfiler) Process() map[string]priority.Priority {
	panic("To be implemented")
}
