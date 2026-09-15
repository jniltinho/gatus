// Package push resolves the pushes received through /api/push (fork): the push token of an endpoint identifies it,
// and a global push key accepts push for every endpoint that receives push
package push

import (
	"crypto/sha256"
	"crypto/subtle"
	"strings"

	"gatus/v5/config"
	"gatus/v5/config/endpoint"
	pushconfig "gatus/v5/config/push"
	"github.com/TwiN/logr"
)

// Target is an endpoint that receives a push
type Target struct {
	// Key is the key of the endpoint
	Key string

	// External is the external endpoint that receives the push, or nil for an active endpoint monitored by Gatus
	External *endpoint.ExternalEndpoint
}

// Resolver resolves the endpoint of a push. It is immutable once created.
type Resolver struct {
	// targets are the enabled endpoints that receive push, by key
	targets map[string]Target

	// tokens are the push tokens used by exactly one endpoint, with the key of that endpoint
	tokens map[string]string

	// endpointTokens are the push tokens of the endpoints, by key, including the tokens used by more than one endpoint
	endpointTokens map[string]string

	// globalKeys are the names of the global push keys, by hash of their token
	globalKeys map[[sha256.Size]byte]string
}

// NewResolver returns the resolver of the endpoints of cfg that receive push: the external endpoints and the endpoints
// listed in push.endpoints
func NewResolver(cfg *config.Config) *Resolver {
	resolver := &Resolver{
		targets:        make(map[string]Target),
		tokens:         make(map[string]string),
		endpointTokens: make(map[string]string),
		globalKeys:     make(map[[sha256.Size]byte]string),
	}
	ambiguous := make(map[string]struct{})
	addToken := func(token, key string) {
		resolver.endpointTokens[key] = token
		if _, isAmbiguous := ambiguous[token]; isAmbiguous {
			return
		}
		if other, used := resolver.tokens[token]; used {
			logr.Warnf("[push.NewResolver] The endpoints with key=%s and key=%s have the same push token: /api/push/<token> ignores it, use /api/push/<token>/<endpoint-key>", other, key)
			delete(resolver.tokens, token)
			ambiguous[token] = struct{}{}
			return
		}
		resolver.tokens[token] = key
	}
	for _, externalEndpoint := range cfg.ExternalEndpoints {
		if !externalEndpoint.IsEnabled() {
			continue
		}
		key := externalEndpoint.Key()
		resolver.targets[key] = Target{Key: key, External: externalEndpoint}
		if len(externalEndpoint.Token) > 0 {
			addToken(externalEndpoint.Token, key)
		}
	}
	if cfg.Push != nil {
		for _, pushEndpoint := range cfg.Push.Endpoints {
			ep := cfg.GetEndpointByKey(pushEndpoint.Key)
			if ep == nil || !ep.IsEnabled() {
				continue
			}
			key := ep.Key()
			resolver.targets[key] = Target{Key: key}
			if len(pushEndpoint.Token) > 0 {
				addToken(pushEndpoint.Token, key)
			}
		}
		for _, globalKey := range cfg.Push.Keys {
			resolver.globalKeys[globalKey.Hash()] = globalKey.Name
		}
	}
	return resolver
}

// Resolve returns the endpoint of a push to /api/push/<token>, or to /api/push/<token>/<endpointKey> when endpointKey
// is not empty. The second form accepts the push token of the endpoint or a global push key. The second return value is
// the name of the global key used, if any.
func (resolver *Resolver) Resolve(token, endpointKey string) (Target, string, bool) {
	if len(token) == 0 {
		return Target{}, "", false
	}
	if len(endpointKey) == 0 {
		key, exists := resolver.tokens[token]
		if !exists {
			return Target{}, "", false
		}
		return resolver.targets[key], "", true
	}
	target, exists := resolver.targets[strings.ToLower(endpointKey)]
	if !exists {
		return Target{}, "", false
	}
	if endpointToken := resolver.endpointTokens[target.Key]; len(endpointToken) > 0 && subtle.ConstantTimeCompare([]byte(endpointToken), []byte(token)) == 1 {
		return target, "", true
	}
	if name, isGlobalKey := resolver.globalKeys[pushconfig.HashToken(token)]; isGlobalKey {
		return target, name, true
	}
	return Target{}, "", false
}
