package statuspage

import (
	"fmt"

	pageconfig "gatus/v5/config/statuspage"
	"gopkg.in/yaml.v3"
)

// Restore of the status pages of a backup of the administration (fork, see the adminbackup package)

// ManagedDefinition returns the stored definition and the version of the managed status page with the given slug
func ManagedDefinition(slug string) (definition string, version int64, exists bool) {
	snap := current.Load()
	if snap == nil {
		return "", 0, false
	}
	state := snap.managedStates[slug]
	if state == nil || state.Stored == nil {
		return "", 0, false
	}
	return state.Stored.Definition, state.Stored.Version, true
}

// NormalizeDefinition returns the definition of a status page as it is stored, to compare it with a stored definition:
// parsed with its default values, a missing enabled meaning disabled
func NormalizeDefinition(raw []byte) (*pageconfig.Page, []byte, error) {
	page, err := Parse(raw)
	if err != nil {
		return nil, nil, err
	}
	if page.Enabled == nil {
		disabled := false
		page.Enabled = &disabled
	}
	definition, err := yaml.Marshal(page)
	if err != nil {
		return nil, nil, err
	}
	return page, definition, nil
}

// ValidateRestore validates, without effects, a status page of a backup: its definition and its slug, which must not be
// used by the configuration file, even when a managed status page already uses it. It returns the normalized page, its
// definition and the selection warnings computed with refs.
func (s *Service) ValidateRestore(raw []byte, refs []EndpointRef) (*pageconfig.Page, []byte, []Warning, error) {
	// Fork: a backup assembled from the reads of the administration carries the hash of the credential masked. The
	// credential stored at the destination is merged before the validation, which would refuse the mask as an invalid
	// hash; a page that does not exist at the destination has no credential to keep and is refused as a masked secret.
	merged, err := mergeRestoredCredential(raw)
	if err != nil {
		return nil, nil, nil, err
	}
	page, definition, err := NormalizeDefinition(merged)
	if err != nil {
		return nil, nil, nil, err
	}
	if snap := current.Load(); snap != nil {
		if _, used := snap.configStates[page.Slug]; used {
			return nil, nil, nil, fmt.Errorf("%w: %s is used by the configuration file", ErrSlugInUse, page.Slug)
		}
	}
	return page, definition, selectionWarnings(page, refs), nil
}

// SelectionWarnings returns the groups and endpoint keys selected by the page without match among the endpoints that
// can be published
func SelectionWarnings(page *pageconfig.Page) []Warning {
	return selectionWarnings(page, Endpoints())
}
