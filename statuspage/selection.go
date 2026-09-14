package statuspage

import (
	"sort"
	"strings"

	pageconfig "gatus/v5/config/statuspage"
)

// EndpointRef identifies an endpoint that can be published on a status page. Only fields that the watchdog never
// changes are copied, so that monitored endpoints are never read concurrently with their evaluation.
type EndpointRef struct {
	Key   string
	Name  string
	Group string
}

// Section is a group of endpoints shown on a status page. Endpoints without group are in a section with an empty name.
type Section struct {
	Group     string
	Endpoints []EndpointRef
}

// Selection is the ordered list of sections of a status page
type Selection struct {
	Sections []Section

	// Truncated is whether endpoints were left out because the page selects more than pageconfig.MaximumEndpoints
	Truncated bool
}

// Keys returns the keys of the selected endpoints, in display order
func (selection Selection) Keys() []string {
	var keys []string
	for _, section := range selection.Sections {
		for _, ref := range section.Endpoints {
			keys = append(keys, ref.Key)
		}
	}
	return keys
}

// Select returns the endpoints of refs selected by the page, by group or by key, in display order:
//   - sections follow the order of page.Groups, then the groups only reached through page.Endpoints in alphabetical
//     order, then the endpoints without group;
//   - endpoints are ordered by name, ignoring case, within each section;
//   - at most pageconfig.MaximumEndpoints endpoints are kept.
func Select(page *pageconfig.Page, refs []EndpointRef) Selection {
	groupOrder := make(map[string]int, len(page.Groups))
	for i, group := range page.Groups {
		groupOrder[group] = i
	}
	selectedKeys := make(map[string]struct{}, len(page.Endpoints))
	for _, key := range page.Endpoints {
		selectedKeys[key] = struct{}{}
	}
	endpointsByGroup := make(map[string][]EndpointRef)
	for _, ref := range refs {
		group := pageconfig.NormalizeGroup(ref.Group)
		_, byGroup := groupOrder[group]
		_, byKey := selectedKeys[ref.Key]
		if !byGroup && !byKey {
			continue
		}
		endpointsByGroup[group] = append(endpointsByGroup[group], ref)
	}
	groups := make([]string, 0, len(endpointsByGroup))
	for group := range endpointsByGroup {
		groups = append(groups, group)
	}
	sort.Slice(groups, func(i, j int) bool {
		return groupLess(groups[i], groups[j], groupOrder)
	})
	var selection Selection
	remaining := pageconfig.MaximumEndpoints
	for _, group := range groups {
		endpoints := endpointsByGroup[group]
		sort.Slice(endpoints, func(i, j int) bool {
			nameI, nameJ := strings.ToLower(endpoints[i].Name), strings.ToLower(endpoints[j].Name)
			if nameI != nameJ {
				return nameI < nameJ
			}
			return endpoints[i].Key < endpoints[j].Key
		})
		if remaining == 0 {
			selection.Truncated = true
			break
		}
		if len(endpoints) > remaining {
			endpoints = endpoints[:remaining]
			selection.Truncated = true
		}
		remaining -= len(endpoints)
		selection.Sections = append(selection.Sections, Section{Group: group, Endpoints: endpoints})
	}
	return selection
}

// groupLess orders the groups of page.Groups first, then the other named groups alphabetically, then the empty group
func groupLess(a, b string, groupOrder map[string]int) bool {
	orderA, selectedA := groupOrder[a]
	orderB, selectedB := groupOrder[b]
	switch {
	case selectedA && selectedB:
		return orderA < orderB
	case selectedA != selectedB:
		return selectedA
	case (a == "") != (b == ""):
		return b == ""
	case strings.ToLower(a) != strings.ToLower(b):
		return strings.ToLower(a) < strings.ToLower(b)
	default:
		return a < b
	}
}
