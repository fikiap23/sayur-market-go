package middleware

import (
	"net/http"
	"os"

	"github.com/labstack/echo/v4"
)

func GatewayValidationMiddleware() echo.MiddlewareFunc {
	requireGateway := os.Getenv("REQUEST_API_GATEWAY")
	if requireGateway == "" {
		requireGateway = "true"
	}

	expectedGateway := os.Getenv("GATEWAY_SECRET_KEY")

	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			if requireGateway == "false" {
				return next(c)
			}

			if c.Request().URL.Path == "/health" {
				return next(c)
			}

			gatewayHeader := c.Request().Header.Get("X-API-Gateway")
			if gatewayHeader != "true" {
				return c.JSON(http.StatusForbidden, map[string]interface{}{
					"status":  "error",
					"message": "Access denied. Request must go through API Gateway.",
					"code":    "GATEWAY_REQUIRED",
				})
			}

			if expectedGateway != "" {
				recievedSecret := c.Request().Header.Get("X-Gateway-Secret")
				if recievedSecret != expectedGateway {
					return c.JSON(http.StatusForbidden, map[string]interface{}{
						"status":  "error",
						"message": "Invalid gateway secret key.",
						"code":    "INVALID_GATEWAY_SECRET",
					})
				}
			}

			gatewayVersion := c.Request().Header.Get("X-API-Gateway-Version")
			if gatewayVersion == "" {
				c.Logger().Warn("Missing X-API-Gateway-Version header")
			}

			secretStatus := "not configured"
			if expectedGateway != "" {
				secretStatus = "validated"
			}
			c.Logger().Infof("Request from API Gateway - Version: %s, Request-ID: %s, Secret: %s",
				gatewayVersion,
				c.Request().Header.Get("X-Request-ID"),
				secretStatus)

			return next(c)
		}
	}
}
