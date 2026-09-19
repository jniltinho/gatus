// Package test holds the helpers shared by the tests of the other packages.
package test

import "net/http"

// MockRoundTripper is an http.RoundTripper backed by a function, used to answer the requests of a client under test
// without a network.
type MockRoundTripper func(r *http.Request) *http.Response

// RoundTrip calls the function and never returns an error.
func (f MockRoundTripper) RoundTrip(r *http.Request) (*http.Response, error) {
	return f(r), nil
}
