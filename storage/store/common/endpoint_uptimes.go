package common

import "time"

// EndpointUptimes is the uptime of an endpoint over the last 24 hours, 7 days and 30 days, between 0 and 1.
// A nil value means that there was no execution during the period.
type EndpointUptimes struct {
	Last24Hours *float64
	Last7Days   *float64
	Last30Days  *float64

	// AverageResponseTime24Hours, AverageResponseTime7Days and AverageResponseTime30Days are the average response times
	// in milliseconds over the same periods. A nil value means that there was no execution during the period.
	AverageResponseTime24Hours *int
	AverageResponseTime7Days   *int
	AverageResponseTime30Days  *int
}

// ResultSummary is the part of an endpoint result that can be shown on a public status page
type ResultSummary struct {
	Timestamp time.Time
	Success   bool
	Duration  time.Duration
}

// EndpointSummary is the latest results and the uptimes of an endpoint, for the public status pages
type EndpointSummary struct {
	// Results are the latest results, from the oldest to the most recent
	Results []ResultSummary

	Uptimes EndpointUptimes
}
