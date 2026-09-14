package statuspage

import (
	"time"

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

	// StatusUnknown means that there is no result
	StatusUnknown = "unknown"
)

// Payload is the public representation of a status page. It only has the fields that can be published: no key, URL,
// hostname, IP address, HTTP status, error, condition or event.
type Payload struct {
	Slug        string         `json:"slug"`
	Title       string         `json:"title"`
	Description string         `json:"description"`
	Status      string         `json:"status"`
	UpdatedAt   time.Time      `json:"updatedAt"`
	Truncated   bool           `json:"truncated"`
	Groups      []GroupPayload `json:"groups"`
}

// GroupPayload is a section of a public status page. Endpoints without group are in a group with an empty name.
type GroupPayload struct {
	Name      string            `json:"name"`
	Status    string            `json:"status"`
	Endpoints []EndpointPayload `json:"endpoints"`
}

// EndpointPayload is the public representation of an endpoint
type EndpointPayload struct {
	Name    string          `json:"name"`
	Status  string          `json:"status"`
	Uptime  UptimePayload   `json:"uptime"`
	Results []ResultPayload `json:"results"`
}

// UptimePayload is the uptime of an endpoint, between 0 and 1, or null without execution during the period
type UptimePayload struct {
	Last24Hours *float64 `json:"24h"`
	Last7Days   *float64 `json:"7d"`
	Last30Days  *float64 `json:"30d"`
}

// ResultPayload is the public representation of a result
type ResultPayload struct {
	Timestamp  time.Time `json:"timestamp"`
	Success    bool      `json:"success"`
	DurationMs int64     `json:"durationMs"`
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
		Groups:      make([]GroupPayload, 0, len(selection.Sections)),
	}
	var pageStatuses []string
	for _, section := range selection.Sections {
		group := GroupPayload{Name: section.Group, Endpoints: make([]EndpointPayload, 0, len(section.Endpoints))}
		groupStatuses := make([]string, 0, len(section.Endpoints))
		for _, ref := range section.Endpoints {
			endpointPayload := buildEndpointPayload(ref, summaries[ref.Key])
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

func buildEndpointPayload(ref EndpointRef, summary *common.EndpointSummary) EndpointPayload {
	endpointPayload := EndpointPayload{Name: ref.Name, Status: StatusUnknown, Results: []ResultPayload{}}
	if summary == nil {
		return endpointPayload
	}
	for _, result := range summary.Results {
		endpointPayload.Results = append(endpointPayload.Results, ResultPayload{
			Timestamp:  result.Timestamp.UTC(),
			Success:    result.Success,
			DurationMs: result.Duration.Milliseconds(),
		})
	}
	if numberOfResults := len(summary.Results); numberOfResults > 0 {
		if summary.Results[numberOfResults-1].Success {
			endpointPayload.Status = StatusUp
		} else {
			endpointPayload.Status = StatusDown
		}
	}
	endpointPayload.Uptime = UptimePayload{
		Last24Hours: summary.Uptimes.Last24Hours,
		Last7Days:   summary.Uptimes.Last7Days,
		Last30Days:  summary.Uptimes.Last30Days,
	}
	return endpointPayload
}

// aggregateStatus returns the status of a group or a page from the statuses of its endpoints, ignoring unknown ones
func aggregateStatus(endpointStatuses []string) string {
	var up, down int
	for _, status := range endpointStatuses {
		switch status {
		case StatusUp:
			up++
		case StatusDown:
			down++
		}
	}
	switch {
	case up == 0 && down == 0:
		return StatusUnknown
	case down == 0:
		return StatusOperational
	case up == 0:
		return StatusDown
	default:
		return StatusDegraded
	}
}
