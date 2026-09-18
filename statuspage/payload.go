package statuspage

import (
	"strconv"
	"strings"
	"time"

	"gatus/v5/config/endpoint"
	pageconfig "gatus/v5/config/statuspage"
	"gatus/v5/storage/store/common"
)

const (
	// StatusOperational means that every endpoint with results is up
	StatusOperational = "operational"

	// StatusDegraded means that some endpoints with results are up and others are down
	StatusDegraded = "degraded"

	// StatusDown means that every endpoint with results is down, or that the last result of an endpoint failed
	StatusDown = "down"

	// StatusUp means that the last result of an endpoint succeeded
	StatusUp = "up"

	// StatusPending means that the last result of an endpoint is pending (fork)
	StatusPending = "pending"

	// StatusUnknown means that there is no result
	StatusUnknown = "unknown"
)

// Payload is the public representation of a status page. It only has the fields that can be published: no key, URL,
// hostname, IP address, HTTP status, error, condition or event.
type Payload struct {
	Slug        string                    `json:"slug"`
	Title       string                    `json:"title"`
	Description string                    `json:"description"`
	Status      string                    `json:"status"`
	UpdatedAt   time.Time                 `json:"updatedAt"`
	Truncated   bool                      `json:"truncated"`
	Featured    []FeaturedEndpointPayload `json:"featured"`
	Groups      []GroupPayload            `json:"groups"`
}

// GroupPayload is a section of a public status page. Endpoints without group are in a group with an empty name.
type GroupPayload struct {
	Name      string            `json:"name"`
	Status    string            `json:"status"`
	Endpoints []EndpointPayload `json:"endpoints"`
}

// FeaturedEndpointPayload is a featured endpoint, with the name of its group
type FeaturedEndpointPayload struct {
	EndpointPayload
	Group string `json:"group"`
}

// EndpointPayload is the public representation of an endpoint
type EndpointPayload struct {
	Name         string              `json:"name"`
	Status       string              `json:"status"`
	Uptime       UptimePayload       `json:"uptime"`
	ResponseTime ResponseTimePayload `json:"responseTime"`
	Results      []ResultPayload     `json:"results"`

	// CertificateExpiresInDays is the number of whole days until the TLS certificate of the endpoint expires, negative
	// once it expired. It is only set when the page shows the certificate expiration and a published result has a
	// certificate (fork).
	CertificateExpiresInDays *int `json:"certificateExpiresInDays,omitempty"`

	// CertificateExpiresAt is the instant of that expiration, from the same result, in UTC. The details page shows it
	// next to the number of days; the rows of the lists show only the days (fork).
	CertificateExpiresAt *time.Time `json:"certificateExpiresAt,omitempty"`
}

// UptimePayload is the uptime of an endpoint, between 0 and 1, or null without execution during the period
type UptimePayload struct {
	Last24Hours *float64 `json:"24h"`
	Last7Days   *float64 `json:"7d"`
	Last30Days  *float64 `json:"30d"`
}

// ResponseTimePayload is the average response time of an endpoint in milliseconds, or null without execution during
// the period
type ResponseTimePayload struct {
	Last24Hours *int `json:"24h"`
	Last7Days   *int `json:"7d"`
	Last30Days  *int `json:"30d"`
}

// ResultPayload is the public representation of a result
type ResultPayload struct {
	Timestamp  time.Time `json:"timestamp"`
	Success    bool      `json:"success"`
	DurationMs int64     `json:"durationMs"`

	// Pending is whether the result is pending (fork)
	Pending bool `json:"pending,omitempty"`

	// Message and Origin are only set on the details page of an endpoint of a page that shows messages (fork)
	Message string `json:"message,omitempty"`
	Origin  string `json:"origin,omitempty"`
}

// EndpointDetailsPayload is the public representation of an endpoint of a status page on its details page. Like
// Payload, it has no key, URL, hostname, IP address, HTTP status, error or condition; its events only have a type and a
// timestamp.
type EndpointDetailsPayload struct {
	Page PageReferencePayload `json:"page"`
	EndpointPayload
	Group     string         `json:"group"`
	UpdatedAt time.Time      `json:"updatedAt"`
	Events    []EventPayload `json:"events"`
}

// PageReferencePayload identifies the status page of an endpoint details page
type PageReferencePayload struct {
	Slug  string `json:"slug"`
	Title string `json:"title"`

	// ShowMessages is whether the details page shows the messages of the results (fork)
	ShowMessages bool `json:"showMessages"`
}

// EventPayload is the public representation of an event: START, HEALTHY or UNHEALTHY
type EventPayload struct {
	Type      string    `json:"type"`
	Timestamp time.Time `json:"timestamp"`
}

// BuildPayload builds the public representation of a page from its selection and the summaries of its endpoints. An
// endpoint without summary (not in the store yet) is unknown.
func BuildPayload(page *pageconfig.Page, selection Selection, summaries map[string]*common.EndpointSummary, now time.Time) *Payload {
	payload := &Payload{
		Slug:        page.Slug,
		Title:       page.Title,
		Description: page.Description,
		UpdatedAt:   now.UTC(),
		Truncated:   selection.Truncated,
		Featured:    make([]FeaturedEndpointPayload, 0, len(selection.Featured)),
		Groups:      make([]GroupPayload, 0, len(selection.Sections)),
	}
	var pageStatuses []string
	for _, ref := range selection.Featured {
		endpointPayload := buildEndpointPayload(page, ref, summaries[ref.Key], now, false)
		payload.Featured = append(payload.Featured, FeaturedEndpointPayload{EndpointPayload: endpointPayload, Group: ref.Group})
		pageStatuses = append(pageStatuses, endpointPayload.Status)
	}
	for _, section := range selection.Sections {
		group := GroupPayload{Name: section.Group, Endpoints: make([]EndpointPayload, 0, len(section.Endpoints))}
		groupStatuses := make([]string, 0, len(section.Endpoints))
		for _, ref := range section.Endpoints {
			endpointPayload := buildEndpointPayload(page, ref, summaries[ref.Key], now, false)
			group.Endpoints = append(group.Endpoints, endpointPayload)
			groupStatuses = append(groupStatuses, endpointPayload.Status)
		}
		group.Status = aggregateStatus(groupStatuses)
		payload.Groups = append(payload.Groups, group)
		pageStatuses = append(pageStatuses, groupStatuses...)
	}
	payload.Status = aggregateStatus(pageStatuses)
	return payload
}

// BuildEndpointDetailsPayload builds the public representation of an endpoint of a page for its details page, from its
// summary and its events, from the oldest to the most recent. Events of an unknown type are left out.
func BuildEndpointDetailsPayload(page *pageconfig.Page, ref EndpointRef, summary *common.EndpointSummary, events []*endpoint.Event, now time.Time) *EndpointDetailsPayload {
	payload := &EndpointDetailsPayload{
		Page:            PageReferencePayload{Slug: page.Slug, Title: page.Title, ShowMessages: page.ShowMessages},
		EndpointPayload: buildEndpointPayload(page, ref, summary, now, page.ShowMessages),
		Group:           ref.Group,
		UpdatedAt:       now.UTC(),
		Events:          make([]EventPayload, 0, len(events)),
	}
	for _, event := range events {
		switch event.Type {
		case endpoint.EventStart, endpoint.EventHealthy, endpoint.EventUnhealthy:
			payload.Events = append(payload.Events, EventPayload{Type: string(event.Type), Timestamp: event.Timestamp.UTC()})
		}
	}
	return payload
}

// buildEndpointPayload builds the public representation of an endpoint. The messages and the origins of its results are
// only set with withMessages, which is only used by the details page of a page that shows messages.
func buildEndpointPayload(page *pageconfig.Page, ref EndpointRef, summary *common.EndpointSummary, now time.Time, withMessages bool) EndpointPayload {
	endpointPayload := EndpointPayload{Name: ref.Name, Status: StatusUnknown, Results: []ResultPayload{}}
	if summary == nil {
		return endpointPayload
	}
	for _, result := range summary.Results {
		resultPayload := ResultPayload{
			Timestamp:  result.Timestamp.UTC(),
			Success:    result.Success,
			DurationMs: result.Duration.Milliseconds(),
			Pending:    result.Pending,
		}
		if withMessages {
			resultPayload.Message, resultPayload.Origin = publicMessage(result), result.Origin
		}
		endpointPayload.Results = append(endpointPayload.Results, resultPayload)
	}
	if page.ShowCertificateExpiration {
		endpointPayload.CertificateExpiresInDays, endpointPayload.CertificateExpiresAt = certificateExpiration(summary.Results, now)
	}
	if numberOfResults := len(summary.Results); numberOfResults > 0 {
		switch lastResult := summary.Results[numberOfResults-1]; {
		case lastResult.Success:
			endpointPayload.Status = StatusUp
		case lastResult.Pending:
			endpointPayload.Status = StatusPending
		default:
			endpointPayload.Status = StatusDown
		}
	}
	endpointPayload.Uptime, endpointPayload.ResponseTime = UptimePayloads(summary.Uptimes)
	return endpointPayload
}

// UptimePayloads returns the uptimes and the average response times of an endpoint, with nil values for the periods
// without execution
func UptimePayloads(uptimes common.EndpointUptimes) (UptimePayload, ResponseTimePayload) {
	return UptimePayload{Last24Hours: uptimes.Last24Hours, Last7Days: uptimes.Last7Days, Last30Days: uptimes.Last30Days},
		ResponseTimePayload{
			Last24Hours: uptimes.AverageResponseTime24Hours,
			Last7Days:   uptimes.AverageResponseTime7Days,
			Last30Days:  uptimes.AverageResponseTime30Days,
		}
}

// publicMessage returns the message of a result that can be published by a page that shows messages (fork): the message
// of a push or of the heartbeat; the message of the heartbeat found in the errors of a result stored before results had
// a message; or the HTTP status of a check. The other errors are never published.
func publicMessage(result common.ResultSummary) string {
	if len(result.Message) > 0 {
		return result.Message
	}
	for _, resultError := range result.Errors {
		if strings.HasPrefix(resultError, endpoint.HeartbeatMessagePrefix) {
			return resultError
		}
	}
	if result.HTTPStatus > 0 {
		return "HTTP " + strconv.Itoa(result.HTTPStatus)
	}
	return ""
}

// aggregateStatus returns the status of a group or a page from the statuses of its endpoints, ignoring unknown ones:
// operational when all are up, down when all are down, and degraded otherwise, including when some are pending (fork)
func aggregateStatus(endpointStatuses []string) string {
	var up, down, pending int
	for _, status := range endpointStatuses {
		switch status {
		case StatusUp:
			up++
		case StatusDown:
			down++
		case StatusPending:
			pending++
		}
	}
	switch {
	case up == 0 && down == 0 && pending == 0:
		return StatusUnknown
	case down == 0 && pending == 0:
		return StatusOperational
	case up == 0 && pending == 0:
		return StatusDown
	default:
		return StatusDegraded
	}
}
