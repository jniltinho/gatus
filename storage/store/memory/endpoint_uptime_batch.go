package memory

import (
	"time"

	"gatus/v5/config/endpoint"
	"gatus/v5/storage/store/common"
)

// GetUptimesByKeys returns the uptimes over the last 24 hours, 7 days and 30 days before now of the endpoints with the
// given keys. Like GetUptimeByKey, the hour in which each period starts is fully counted.
func (s *Store) GetUptimesByKeys(keys []string, now time.Time) (map[string]*common.EndpointUptimes, error) {
	uptimes := make(map[string]*common.EndpointUptimes, len(keys))
	periodStarts := [3]int64{
		now.Add(-24 * time.Hour).Truncate(time.Hour).Unix(),
		now.Add(-7 * 24 * time.Hour).Truncate(time.Hour).Unix(),
		now.Add(-30 * 24 * time.Hour).Truncate(time.Hour).Unix(),
	}
	end := now.Unix()
	s.RLock()
	defer s.RUnlock()
	for _, key := range keys {
		endpointStatus, ok := s.endpointCache.GetValue(key).(*endpoint.Status)
		if !ok || endpointStatus.Uptime == nil {
			continue
		}
		var totalExecutions, successfulExecutions [3]uint64
		for hourlyUnixTimestamp, hourlyStats := range endpointStatus.Uptime.HourlyStatistics {
			if hourlyStats == nil || hourlyUnixTimestamp > end {
				continue
			}
			for i, periodStart := range periodStarts {
				if hourlyUnixTimestamp >= periodStart {
					totalExecutions[i] += hourlyStats.TotalExecutions
					successfulExecutions[i] += hourlyStats.SuccessfulExecutions
				}
			}
		}
		if totalExecutions[2] == 0 {
			continue
		}
		uptimes[key] = &common.EndpointUptimes{
			Last24Hours: uptimeRatio(successfulExecutions[0], totalExecutions[0]),
			Last7Days:   uptimeRatio(successfulExecutions[1], totalExecutions[1]),
			Last30Days:  uptimeRatio(successfulExecutions[2], totalExecutions[2]),
		}
	}
	return uptimes, nil
}

func uptimeRatio(successfulExecutions, totalExecutions uint64) *float64 {
	if totalExecutions == 0 {
		return nil
	}
	ratio := float64(successfulExecutions) / float64(totalExecutions)
	return &ratio
}
