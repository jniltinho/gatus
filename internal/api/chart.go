package api

import (
	"bytes"
	"errors"
	"math"
	"net/http"
	"net/url"
	"sort"
	"time"

	"gatus/v5/internal/httpx"
	"gatus/v5/internal/storage/store"
	"gatus/v5/internal/storage/store/common"

	"github.com/TwiN/logr"
	"github.com/labstack/echo/v5"
	"github.com/wcharczuk/go-chart/v2"
	"github.com/wcharczuk/go-chart/v2/drawing"
)

const timeFormat = "3:04PM"

var (
	gridStyle = chart.Style{
		StrokeColor: drawing.Color{R: 119, G: 119, B: 119, A: 40},
		StrokeWidth: 1.0,
	}
	axisStyle = chart.Style{
		FontColor: drawing.Color{R: 119, G: 119, B: 119, A: 255},
	}
	transparentStyle = chart.Style{
		FillColor: drawing.Color{R: 255, G: 255, B: 255, A: 0},
	}
)

func ResponseTimeChart(c *echo.Context) error {
	duration := c.Param("duration")
	chartTimestampFormatter := chart.TimeValueFormatterWithFormat(timeFormat)
	var from time.Time
	switch duration {
	case "30d":
		from = time.Now().Truncate(time.Hour).Add(-30 * 24 * time.Hour)
		chartTimestampFormatter = chart.TimeDateValueFormatter
	case "7d":
		from = time.Now().Truncate(time.Hour).Add(-7 * 24 * time.Hour)
	case "24h":
		from = time.Now().Truncate(time.Hour).Add(-24 * time.Hour)
	default:
		return httpx.SendString(c, 400, "Durations supported: 30d, 7d, 24h")
	}
	key, err := url.QueryUnescape(c.Param("key"))
	if err != nil {
		return httpx.SendString(c, 400, "invalid key encoding")
	}
	hourlyAverageResponseTime, err := store.Get().GetHourlyAverageResponseTimeByKey(key, from, time.Now())
	if err != nil {
		if errors.Is(err, common.ErrEndpointNotFound) {
			return httpx.SendString(c, 404, err.Error())
		} else if errors.Is(err, common.ErrInvalidTimeRange) {
			return httpx.SendString(c, 400, err.Error())
		}
		return httpx.SendString(c, 500, err.Error())
	}
	if len(hourlyAverageResponseTime) == 0 {
		return httpx.SendString(c, 204, "")
	}
	series := chart.TimeSeries{
		Name: "Average response time per hour",
		Style: chart.Style{
			StrokeWidth: 1.5,
			DotWidth:    2.0,
		},
	}
	keys := make([]int, 0, len(hourlyAverageResponseTime))
	earliestTimestamp := int64(0)
	for hourlyTimestamp := range hourlyAverageResponseTime {
		keys = append(keys, int(hourlyTimestamp))
		if earliestTimestamp == 0 || hourlyTimestamp < earliestTimestamp {
			earliestTimestamp = hourlyTimestamp
		}
	}
	for earliestTimestamp > from.Unix() {
		earliestTimestamp -= int64(time.Hour.Seconds())
		keys = append(keys, int(earliestTimestamp))
	}
	sort.Ints(keys)
	var maxAverageResponseTime float64
	for _, key := range keys {
		averageResponseTime := float64(hourlyAverageResponseTime[int64(key)])
		if maxAverageResponseTime < averageResponseTime {
			maxAverageResponseTime = averageResponseTime
		}
		series.XValues = append(series.XValues, time.Unix(int64(key), 0))
		series.YValues = append(series.YValues, averageResponseTime)
	}
	graph := chart.Chart{
		Canvas:     transparentStyle,
		Background: transparentStyle,
		Width:      1280,
		Height:     300,
		XAxis: chart.XAxis{
			ValueFormatter: chartTimestampFormatter,
			GridMajorStyle: gridStyle,
			GridMinorStyle: gridStyle,
			Style:          axisStyle,
			NameStyle:      axisStyle,
		},
		YAxis: chart.YAxis{
			Name:           "Average response time",
			GridMajorStyle: gridStyle,
			GridMinorStyle: gridStyle,
			Style:          axisStyle,
			NameStyle:      axisStyle,
			Range: &chart.ContinuousRange{
				Min: 0,
				Max: math.Ceil(maxAverageResponseTime * 1.25),
			},
		},
		Series: []chart.Series{series},
	}
	// Rendered into a buffer: once the first byte is written the answer cannot become an error anymore
	var rendered bytes.Buffer
	if err := graph.Render(chart.SVG, &rendered); err != nil {
		logr.Errorf("[api.ResponseTimeChart] Failed to render response time chart: %s", err.Error())
		return httpx.SendString(c, 500, err.Error())
	}
	httpx.SetHeader(c, "Content-Type", "image/svg+xml")
	httpx.SetHeader(c, "Cache-Control", "no-cache, no-store")
	httpx.SetHeader(c, "Expires", "0")
	return httpx.Send(c, http.StatusOK, rendered.Bytes())
}

func ResponseTimeHistory(c *echo.Context) error {
	duration := c.Param("duration")
	var from time.Time
	switch duration {
	case "30d":
		from = time.Now().Truncate(time.Hour).Add(-30 * 24 * time.Hour)
	case "7d":
		from = time.Now().Truncate(time.Hour).Add(-7 * 24 * time.Hour)
	case "24h":
		from = time.Now().Truncate(time.Hour).Add(-24 * time.Hour)
	default:
		return httpx.SendString(c, 400, "Durations supported: 30d, 7d, 24h")
	}
	endpointKey, err := url.QueryUnescape(c.Param("key"))
	if err != nil {
		return httpx.SendString(c, 400, "invalid key encoding")
	}
	hourlyAverageResponseTime, err := store.Get().GetHourlyAverageResponseTimeByKey(endpointKey, from, time.Now())
	if err != nil {
		if errors.Is(err, common.ErrEndpointNotFound) {
			return httpx.SendString(c, 404, err.Error())
		}
		if errors.Is(err, common.ErrInvalidTimeRange) {
			return httpx.SendString(c, 400, err.Error())
		}
		return httpx.SendString(c, 500, err.Error())
	}
	if len(hourlyAverageResponseTime) == 0 {
		return httpx.JSON(c, 200, map[string]interface{}{
			"timestamps": []int64{},
			"values":     []int{},
		})
	}
	hourlyTimestamps := make([]int, 0, len(hourlyAverageResponseTime))
	earliestTimestamp := int64(0)
	for hourlyTimestamp := range hourlyAverageResponseTime {
		hourlyTimestamps = append(hourlyTimestamps, int(hourlyTimestamp))
		if earliestTimestamp == 0 || hourlyTimestamp < earliestTimestamp {
			earliestTimestamp = hourlyTimestamp
		}
	}
	for earliestTimestamp > from.Unix() {
		earliestTimestamp -= int64(time.Hour.Seconds())
		hourlyTimestamps = append(hourlyTimestamps, int(earliestTimestamp))
	}
	sort.Ints(hourlyTimestamps)
	timestamps := make([]int64, 0, len(hourlyTimestamps))
	values := make([]int, 0, len(hourlyTimestamps))
	for _, hourlyTimestamp := range hourlyTimestamps {
		timestamp := int64(hourlyTimestamp)
		averageResponseTime := hourlyAverageResponseTime[timestamp]
		timestamps = append(timestamps, timestamp*1000)
		values = append(values, averageResponseTime)
	}
	return httpx.JSON(c, http.StatusOK, map[string]interface{}{
		"timestamps": timestamps,
		"values":     values,
	})
}
