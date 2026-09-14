package common

// EndpointUptimes is the uptime of an endpoint over the last 24 hours, 7 days and 30 days, between 0 and 1.
// A nil value means that there was no execution during the period.
type EndpointUptimes struct {
	Last24Hours *float64
	Last7Days   *float64
	Last30Days  *float64
}
