package main

import (
	influxdb2 "github.com/influxdata/influxdb-client-go/v2"
	"github.com/influxdata/influxdb-client-go/v2/api"
	"github.com/kcz17/profiler/priority"
)

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

func (p *InfluxDBProfiler) Profile(sessionID string) priority.Priority {
	// TODO(kz): Remove hardcoding once tested.
	return priority.Low
}
