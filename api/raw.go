package api

import (
	"errors"
	"fmt"
	"net/url"
	"time"

	"gatus/v5/internal/httpx"
	"gatus/v5/storage/store"
	"gatus/v5/storage/store/common"

	"github.com/labstack/echo/v5"
)

func UptimeRaw(c *echo.Context) error {
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

	httpx.SetHeader(c, "Content-Type", "text/plain")
	httpx.SetHeader(c, "Cache-Control", "no-cache, no-store, must-revalidate")
	httpx.SetHeader(c, "Expires", "0")
	return httpx.Send(c, 200, []byte(fmt.Sprintf("%f", uptime)))
}

func ResponseTimeRaw(c *echo.Context) error {
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
	responseTime, err := store.Get().GetAverageResponseTimeByKey(key, from, time.Now())
	if err != nil {
		if errors.Is(err, common.ErrEndpointNotFound) {
			return httpx.SendString(c, 404, err.Error())
		} else if errors.Is(err, common.ErrInvalidTimeRange) {
			return httpx.SendString(c, 400, err.Error())
		}
		return httpx.SendString(c, 500, err.Error())
	}

	httpx.SetHeader(c, "Content-Type", "text/plain")
	httpx.SetHeader(c, "Cache-Control", "no-cache, no-store, must-revalidate")
	httpx.SetHeader(c, "Expires", "0")
	return httpx.Send(c, 200, []byte(fmt.Sprintf("%d", responseTime)))
}
