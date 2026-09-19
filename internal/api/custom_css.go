package api

import (
	"gatus/v5/internal/httpx"

	"github.com/labstack/echo/v5"
)

type CustomCSSHandler struct {
	customCSS string
}

func (handler CustomCSSHandler) GetCustomCSS(c *echo.Context) error {
	httpx.SetHeader(c, "Content-Type", "text/css")
	return httpx.SendString(c, 200, handler.customCSS)
}
