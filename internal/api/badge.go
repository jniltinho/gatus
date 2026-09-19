package api

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"strconv"
	"strings"
	"time"

	"gatus/v5/internal/config"
	"gatus/v5/internal/config/endpoint/ui"
	"gatus/v5/internal/httpx"
	"gatus/v5/internal/managedendpoint"
	"gatus/v5/internal/storage/store"
	"gatus/v5/internal/storage/store/common"
	"gatus/v5/internal/storage/store/common/paging"

	"github.com/labstack/echo/v5"
)

const (
	badgeColorHexAwesome  = "#40cc11"
	badgeColorHexGreat    = "#94cc11"
	badgeColorHexGood     = "#ccd311"
	badgeColorHexPassable = "#ccb311"
	badgeColorHexBad      = "#cc8111"
	badgeColorHexVeryBad  = "#c7130a"
)

const (
	HealthStatusUp      = "up"
	HealthStatusDown    = "down"
	HealthStatusUnknown = "?"
)

var (
	badgeColors = []string{badgeColorHexAwesome, badgeColorHexGreat, badgeColorHexGood, badgeColorHexPassable, badgeColorHexBad}
)

// UptimeBadge handles the automatic generation of badge based on the group name and endpoint name passed.
//
// Valid values for :duration -> 30d, 7d, 24h, 1h
func UptimeBadge(c *echo.Context) error {
	duration := c.Param("duration")
	var from time.Time
	switch duration {
	case "30d":
		from = time.Now().Add(-30 * 24 * time.Hour)
	case "7d":
		from = time.Now().Add(-7 * 24 * time.Hour)
	case "24h":
		from = time.Now().Add(-24 * time.Hour)
	case "1h":
		from = time.Now().Add(-2 * time.Hour) // Because uptime metrics are stored by hour, we have to cheat a little
	default:
		return httpx.SendString(c, 400, "Durations supported: 30d, 7d, 24h, 1h")
	}
	key, err := url.QueryUnescape(c.Param("key"))
	if err != nil {
		return httpx.SendString(c, 400, "invalid key encoding")
	}
	uptime, err := store.Get().GetUptimeByKey(key, from, time.Now())
	if err != nil {
		if errors.Is(err, common.ErrEndpointNotFound) {
			return httpx.SendString(c, 404, err.Error())
		} else if errors.Is(err, common.ErrInvalidTimeRange) {
			return httpx.SendString(c, 400, err.Error())
		}
		return httpx.SendString(c, 500, err.Error())
	}
	httpx.SetHeader(c, "Content-Type", "image/svg+xml")
	setBadgeCacheControl(c)
	httpx.SetHeader(c, "Expires", "0")
	return httpx.Send(c, 200, generateUptimeBadgeSVG(duration, uptime))
}

// ResponseTimeBadge handles the automatic generation of badge based on the group name and endpoint name passed.
//
// Valid values for :duration -> 30d, 7d, 24h, 1h
func ResponseTimeBadge(cfg *config.Config) echo.HandlerFunc {
	return func(c *echo.Context) error {
		duration := c.Param("duration")
		var from time.Time
		switch duration {
		case "30d":
			from = time.Now().Add(-30 * 24 * time.Hour)
		case "7d":
			from = time.Now().Add(-7 * 24 * time.Hour)
		case "24h":
			from = time.Now().Add(-24 * time.Hour)
		case "1h":
			from = time.Now().Add(-2 * time.Hour) // Because response time metrics are stored by hour, we have to cheat a little
		default:
			return httpx.SendString(c, 400, "Durations supported: 30d, 7d, 24h, 1h")
		}
		key, err := url.QueryUnescape(c.Param("key"))
		if err != nil {
			return httpx.SendString(c, 400, "invalid key encoding")
		}
		averageResponseTime, err := store.Get().GetAverageResponseTimeByKey(key, from, time.Now())
		if err != nil {
			if errors.Is(err, common.ErrEndpointNotFound) {
				return httpx.SendString(c, 404, err.Error())
			} else if errors.Is(err, common.ErrInvalidTimeRange) {
				return httpx.SendString(c, 400, err.Error())
			}
			return httpx.SendString(c, 500, err.Error())
		}
		httpx.SetHeader(c, "Content-Type", "image/svg+xml")
		setBadgeCacheControl(c)
		httpx.SetHeader(c, "Expires", "0")
		return httpx.Send(c, 200, generateResponseTimeBadgeSVG(duration, averageResponseTime, key, cfg))
	}
}

// HealthBadge handles the automatic generation of badge based on the group name and endpoint name passed.
func HealthBadge(c *echo.Context) error {
	key, err := url.QueryUnescape(c.Param("key"))
	if err != nil {
		return httpx.SendString(c, 400, "invalid key encoding")
	}
	pagingConfig := paging.NewEndpointStatusParams()
	status, err := store.Get().GetEndpointStatusByKey(key, pagingConfig.WithResults(1, 1))
	if err != nil {
		if errors.Is(err, common.ErrEndpointNotFound) {
			return httpx.SendString(c, 404, err.Error())
		} else if errors.Is(err, common.ErrInvalidTimeRange) {
			return httpx.SendString(c, 400, err.Error())
		}
		return httpx.SendString(c, 500, err.Error())
	}
	healthStatus := HealthStatusUnknown
	if len(status.Results) > 0 {
		if status.Results[0].Success {
			healthStatus = HealthStatusUp
		} else {
			healthStatus = HealthStatusDown
		}
	}
	httpx.SetHeader(c, "Content-Type", "image/svg+xml")
	setBadgeCacheControl(c)
	httpx.SetHeader(c, "Expires", "0")
	return httpx.Send(c, 200, generateHealthBadgeSVG(healthStatus))
}

func HealthBadgeShields(c *echo.Context) error {
	key, err := url.QueryUnescape(c.Param("key"))
	if err != nil {
		return httpx.SendString(c, 400, "invalid key encoding")
	}
	pagingConfig := paging.NewEndpointStatusParams()
	status, err := store.Get().GetEndpointStatusByKey(key, pagingConfig.WithResults(1, 1))
	if err != nil {
		if errors.Is(err, common.ErrEndpointNotFound) {
			return httpx.SendString(c, 404, err.Error())
		} else if errors.Is(err, common.ErrInvalidTimeRange) {
			return httpx.SendString(c, 400, err.Error())
		}
		return httpx.SendString(c, 500, err.Error())
	}
	healthStatus := HealthStatusUnknown
	if len(status.Results) > 0 {
		if status.Results[0].Success {
			healthStatus = HealthStatusUp
		} else {
			healthStatus = HealthStatusDown
		}
	}
	httpx.SetHeader(c, "Content-Type", "application/json")
	setBadgeCacheControl(c)
	httpx.SetHeader(c, "Expires", "0")
	jsonData, err := generateHealthBadgeShields(healthStatus)
	if err != nil {
		return httpx.SendString(c, 500, err.Error())
	}
	return httpx.Send(c, 200, jsonData)
}

func generateUptimeBadgeSVG(duration string, uptime float64) []byte {
	var labelWidth, valueWidth, valueWidthAdjustment int
	switch duration {
	case "30d":
		labelWidth = 70
	case "7d":
		labelWidth = 65
	case "24h":
		labelWidth = 70
	case "1h":
		labelWidth = 65
	default:
	}
	color := getBadgeColorFromUptime(uptime)
	sanitizedValue := strings.TrimRight(strings.TrimRight(fmt.Sprintf("%.2f", uptime*100), "0"), ".") + "%"
	if strings.Contains(sanitizedValue, ".") {
		valueWidthAdjustment = -10
	}
	valueWidth = (len(sanitizedValue) * 11) + valueWidthAdjustment
	width := labelWidth + valueWidth
	labelX := labelWidth / 2
	valueX := labelWidth + (valueWidth / 2)
	svg := []byte(fmt.Sprintf(`<svg xmlns="http://www.w3.org/2000/svg" width="%d" height="20">
  <linearGradient id="b" x2="0" y2="100%%">
    <stop offset="0" stop-color="#bbb" stop-opacity=".1"/>
    <stop offset="1" stop-opacity=".1"/>
  </linearGradient>
  <mask id="a">
    <rect width="%d" height="20" fill="#fff"/>
  </mask>
  <g mask="url(#a)">
    <path fill="#555" d="M0 0h%dv20H0z"/>
    <path fill="%s" d="M%d 0h%dv20H%dz"/>
    <path fill="url(#b)" d="M0 0h%dv20H0z"/>
  </g>
  <g fill="#fff" text-anchor="middle" font-family="DejaVu Sans,Verdana,Geneva,sans-serif" font-size="11">
    <text x="%d" y="15" fill="#010101" fill-opacity=".3">
      uptime %s
    </text>
    <text x="%d" y="14">
      uptime %s
    </text>
    <text x="%d" y="15" fill="#010101" fill-opacity=".3">
      %s
    </text>
    <text x="%d" y="14">
      %s
    </text>
  </g>
</svg>`, width, width, labelWidth, color, labelWidth, valueWidth, labelWidth, width, labelX, duration, labelX, duration, valueX, sanitizedValue, valueX, sanitizedValue))
	return svg
}

func getBadgeColorFromUptime(uptime float64) string {
	if uptime >= 0.975 {
		return badgeColorHexAwesome
	} else if uptime >= 0.95 {
		return badgeColorHexGreat
	} else if uptime >= 0.9 {
		return badgeColorHexGood
	} else if uptime >= 0.8 {
		return badgeColorHexPassable
	} else if uptime >= 0.65 {
		return badgeColorHexBad
	}
	return badgeColorHexVeryBad
}

func generateResponseTimeBadgeSVG(duration string, averageResponseTime int, key string, cfg *config.Config) []byte {
	var labelWidth, valueWidth int
	switch duration {
	case "30d":
		labelWidth = 110
	case "7d":
		labelWidth = 105
	case "24h":
		labelWidth = 110
	case "1h":
		labelWidth = 105
	default:
	}
	color := getBadgeColorFromResponseTime(averageResponseTime, key, cfg)
	sanitizedValue := strconv.Itoa(averageResponseTime) + "ms"
	valueWidth = len(sanitizedValue) * 11
	width := labelWidth + valueWidth
	labelX := labelWidth / 2
	valueX := labelWidth + (valueWidth / 2)
	svg := []byte(fmt.Sprintf(`<svg xmlns="http://www.w3.org/2000/svg" width="%d" height="20">
  <linearGradient id="b" x2="0" y2="100%%">
    <stop offset="0" stop-color="#bbb" stop-opacity=".1"/>
    <stop offset="1" stop-opacity=".1"/>
  </linearGradient>
  <mask id="a">
    <rect width="%d" height="20" fill="#fff"/>
  </mask>
  <g mask="url(#a)">
    <path fill="#555" d="M0 0h%dv20H0z"/>
    <path fill="%s" d="M%d 0h%dv20H%dz"/>
    <path fill="url(#b)" d="M0 0h%dv20H0z"/>
  </g>
  <g fill="#fff" text-anchor="middle" font-family="DejaVu Sans,Verdana,Geneva,sans-serif" font-size="11">
    <text x="%d" y="15" fill="#010101" fill-opacity=".3">
      response time %s
    </text>
    <text x="%d" y="14">
      response time %s
    </text>
    <text x="%d" y="15" fill="#010101" fill-opacity=".3">
      %s
    </text>
    <text x="%d" y="14">
      %s
    </text>
  </g>
</svg>`, width, width, labelWidth, color, labelWidth, valueWidth, labelWidth, width, labelX, duration, labelX, duration, valueX, sanitizedValue, valueX, sanitizedValue))
	return svg
}

func getBadgeColorFromResponseTime(responseTime int, key string, cfg *config.Config) string {
	thresholds := ui.GetDefaultConfig().Badge.ResponseTime.Thresholds
	if endpoint := cfg.GetEndpointByKey(key); endpoint != nil {
		thresholds = endpoint.UIConfig.Badge.ResponseTime.Thresholds
	} else if endpoint := managedendpoint.EndpointByKey(key); endpoint != nil {
		thresholds = endpoint.UIConfig.Badge.ResponseTime.Thresholds
	}
	// the threshold config requires 5 values, so we can be sure it's set here
	for i := range 5 {
		if responseTime <= thresholds[i] {
			return badgeColors[i]
		}
	}
	return badgeColorHexVeryBad
}

func generateHealthBadgeSVG(healthStatus string) []byte {
	var labelWidth, valueWidth int
	switch healthStatus {
	case HealthStatusUp:
		valueWidth = 28
	case HealthStatusDown:
		valueWidth = 44
	case HealthStatusUnknown:
		valueWidth = 10
	default:
	}
	color := getBadgeColorFromHealth(healthStatus)
	labelWidth = 48

	width := labelWidth + valueWidth
	labelX := labelWidth / 2
	valueX := labelWidth + (valueWidth / 2)
	svg := []byte(fmt.Sprintf(`<svg xmlns="http://www.w3.org/2000/svg" width="%d" height="20">
  <linearGradient id="b" x2="0" y2="100%%">
    <stop offset="0" stop-color="#bbb" stop-opacity=".1"/>
    <stop offset="1" stop-opacity=".1"/>
  </linearGradient>
  <mask id="a">
    <rect width="%d" height="20" fill="#fff"/>
  </mask>
  <g mask="url(#a)">
    <path fill="#555" d="M0 0h%dv20H0z"/>
    <path fill="%s" d="M%d 0h%dv20H%dz"/>
    <path fill="url(#b)" d="M0 0h%dv20H0z"/>
  </g>
  <g fill="#fff" text-anchor="middle" font-family="DejaVu Sans,Verdana,Geneva,sans-serif" font-size="11">
    <text x="%d" y="15" fill="#010101" fill-opacity=".3">
      health
    </text>
    <text x="%d" y="14">
      health
    </text>
    <text x="%d" y="15" fill="#010101" fill-opacity=".3">
      %s
    </text>
    <text x="%d" y="14">
      %s
    </text>
  </g>
</svg>`, width, width, labelWidth, color, labelWidth, valueWidth, labelWidth, width, labelX, labelX, valueX, healthStatus, valueX, healthStatus))

	return svg
}

func generateHealthBadgeShields(healthStatus string) ([]byte, error) {
	color := getBadgeShieldsColorFromHealth(healthStatus)
	data := map[string]interface{}{
		"schemaVersion": 1,
		"label":         "gatus",
		"message":       healthStatus,
		"color":         color,
	}
	return json.Marshal(data)
}

func getBadgeColorFromHealth(healthStatus string) string {
	if healthStatus == HealthStatusUp {
		return badgeColorHexAwesome
	} else if healthStatus == HealthStatusDown {
		return badgeColorHexVeryBad
	}
	return badgeColorHexPassable
}

func getBadgeShieldsColorFromHealth(healthStatus string) string {
	if healthStatus == HealthStatusUp {
		return "brightgreen"
	} else if healthStatus == HealthStatusDown {
		return "red"
	}
	return "yellow"
}

// setBadgeCacheControl sets the Cache-Control of a badge before its body is written. The badge of a status page with a
// login of its own is private: with net/http a header set after the first byte never reaches the client, so the handler
// of the page says it beforehand instead of changing the header afterwards, see statusPageBadgeHandler.
func setBadgeCacheControl(c *echo.Context) {
	if private, _ := c.Get(localsPrivateBadge).(bool); private {
		httpx.SetHeader(c, "Cache-Control", "private, no-store")
		httpx.Vary(c, echo.HeaderAuthorization)
		return
	}
	httpx.SetHeader(c, "Cache-Control", "no-cache, no-store, must-revalidate")
}
