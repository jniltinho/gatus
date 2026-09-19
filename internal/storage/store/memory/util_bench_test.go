package memory

import (
	"testing"

	"gatus/v5/internal/config/endpoint"
	"gatus/v5/internal/storage"
	"gatus/v5/internal/storage/store/common/paging"
)

func BenchmarkShallowCopyEndpointStatus(b *testing.B) {
	ep := &testEndpoint
	status := endpoint.NewStatus(ep.Group, ep.Name)
	for range storage.DefaultMaximumNumberOfResults {
		AddResult(status, &testSuccessfulResult, storage.DefaultMaximumNumberOfResults, storage.DefaultMaximumNumberOfEvents)
	}
	for b.Loop() {
		ShallowCopyEndpointStatus(status, paging.NewEndpointStatusParams().WithResults(1, 20))
	}
	b.ReportAllocs()
}
