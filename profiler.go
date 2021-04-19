package main

import (
	"context"
	"fmt"
	influxdb2 "github.com/influxdata/influxdb-client-go/v2"
	"github.com/influxdata/influxdb-client-go/v2/api"
	"github.com/kcz17/profiler/priority"
	"strings"
)

// MinSessionRequests specifies how many session requests are required before
// a session can be profiled. This is set to prevent sessions from being
// profiled too early.
const MinSessionRequests = 10

type SessionRequest struct {
	Method string
	Path   string
}

type InfluxDBProfiler struct {
	rules  OrderedRules
	client influxdb2.Client
	query  api.QueryAPI
	bucket string
}

func NewInfluxDBProfiler(rules OrderedRules, addr, authToken, org, bucket string) *InfluxDBProfiler {
	options := influxdb2.DefaultOptions()
	client := influxdb2.NewClientWithOptions(addr, authToken, options)
	queryAPI := client.QueryAPI(org)

	return &InfluxDBProfiler{
		rules:  rules,
		client: client,
		query:  queryAPI,
		bucket: bucket,
	}
}

func (p *InfluxDBProfiler) Profile(sessionID string) (priority.Priority, error) {
	escapedSessionID := strings.ReplaceAll(sessionID, "\"", "%22")
	result, err := p.query.Query(
		context.Background(),
		`from(bucket: "session_history")
		  |> range(start: -1h)
		  |> filter(fn: (r) => 
			r["_measurement"] == "request" and
			r["session_id"] == "`+escapedSessionID+`"
		  )
		  |> pivot(
			rowKey:["_time"],
			columnKey: ["_field"],
			valueColumn: "_value"
		  )
		  |> yield()`)
	if err != nil {
		return priority.Unknown, fmt.Errorf("influxdb query for session ID %s expected nil err; got err = %w", sessionID, err)
	}

	sessionRequests, err := queryTableResultToSessionRequests(result)
	if err != nil {
		return priority.Unknown, fmt.Errorf("could extract session requests from query table result for session ID %s: err = %w", sessionID, err)
	}

	if len(sessionRequests) < MinSessionRequests {
		return priority.Unknown, nil
	}

	// Match the list of requests to the rules by order of rule priority
	// descending. O(mn), m = no. rules, n = no. session requests.
	for _, rule := range p.rules {
		occurrences := 0

		for _, req := range sessionRequests {
			if rule.IsMatch(req.Method, req.Path) {
				occurrences++
			}

			if occurrences >= rule.Occurrences {
				return rule.Result, nil
			}
		}
	}

	return priority.Unknown, nil
}

func queryTableResultToSessionRequests(result *api.QueryTableResult) ([]SessionRequest, error) {
	var requests []SessionRequest
	for result.Next() {
		var request SessionRequest

		methodField := result.Record().ValueByKey("method")
		if methodField == nil {
			return nil, fmt.Errorf("expected non-nil method value in table; got method = nil with values:\n%s", result.Record().Values())
		}
		request.Method = methodField.(string)

		pathField := result.Record().ValueByKey("path")
		if pathField == nil {
			return nil, fmt.Errorf("expected non-nil path value in table; got path = nil with values:\n%s", result.Record().Values())
		}

		// Prepend leading slash if it does not exist.
		if len(pathField.(string)) == 0 || pathField.(string)[0] != '/' {
			request.Path = "/" + pathField.(string)
		} else {
			request.Path = pathField.(string)
		}

		requests = append(requests, request)
	}

	if result.Err() != nil {
		return nil, fmt.Errorf("result.Err() expected nil err; got err = %w", result.Err())
	}

	return requests, nil
}
