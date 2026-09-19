package api

import (
	"fmt"
	"net/http"

	"gatus/v5/internal/config"
	"gatus/v5/internal/config/suite"
	"gatus/v5/internal/httpx"
	"gatus/v5/internal/storage/store"
	"gatus/v5/internal/storage/store/common/paging"

	"github.com/labstack/echo/v5"
)

// SuiteStatuses handles requests to retrieve all suite statuses
func SuiteStatuses(cfg *config.Config) echo.HandlerFunc {
	return func(c *echo.Context) error {
		page, pageSize := extractPageAndPageSizeFromRequest(c, 100)
		params := paging.NewSuiteStatusParams().WithPagination(page, pageSize)
		suiteStatuses, err := store.Get().GetAllSuiteStatuses(params)
		if err != nil {
			return httpx.JSON(c, http.StatusInternalServerError, map[string]any{
				"error": fmt.Sprintf("Failed to retrieve suite statuses: %v", err),
			})
		}
		// If no statuses exist yet, create empty ones from config
		if len(suiteStatuses) == 0 {
			for _, s := range cfg.Suites {
				if s.IsEnabled() {
					suiteStatuses = append(suiteStatuses, suite.NewStatus(s))
				}
			}
		}
		return httpx.JSON(c, http.StatusOK, suiteStatuses)
	}
}

// SuiteStatus handles requests to retrieve a single suite's status
func SuiteStatus(cfg *config.Config) echo.HandlerFunc {
	return func(c *echo.Context) error {
		page, pageSize := extractPageAndPageSizeFromRequest(c, 100)
		key := c.Param("key")
		params := paging.NewSuiteStatusParams().WithPagination(page, pageSize)
		status, err := store.Get().GetSuiteStatusByKey(key, params)
		if err != nil || status == nil {
			// Try to find the suite in config
			for _, s := range cfg.Suites {
				if s.Key() == key {
					status = suite.NewStatus(s)
					break
				}
			}
			if status == nil {
				return httpx.JSON(c, 404, map[string]any{
					"error": fmt.Sprintf("Suite with key '%s' not found", key),
				})
			}
		}
		return httpx.JSON(c, http.StatusOK, status)
	}
}
