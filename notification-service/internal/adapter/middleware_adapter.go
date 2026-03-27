package adapter

import (
	"encoding/json"
	"net/http"
	"notification-service/config"
	"notification-service/internal/adapter/handlers/response"
	"notification-service/internal/core/domain/entity"

	"strings"

	"github.com/golang-jwt/jwt/v5"
	"github.com/labstack/echo/v4"
	"github.com/labstack/gommon/log"
)

type MiddlewareAdapterInterface interface {
	CheckToken() echo.MiddlewareFunc
}

type middlewareAdapter struct {
	cfg *config.Config
}

// CheckToken implements MiddlewareAdapterInterface.
func (m *middlewareAdapter) CheckToken() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			redisConn := config.NewConfig().NewRedisClient()
			authHeader := c.Request().Header.Get("Authorization")
			if authHeader == "" {
				log.Errorf("[MiddlewareAdapter-1] CheckToken: %s", "missing or invalid token")
				return c.JSON(http.StatusUnauthorized, response.Response("missing or invalid token", nil))
			}

			tokenString := strings.TrimPrefix(authHeader, "Bearer ")

			_, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
				if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
					return nil, jwt.ErrSignatureInvalid
				}

				return []byte(m.cfg.App.JwtSecretKey), nil
			})
			if err != nil {
				log.Errorf("[MiddlewareAdapter-2] CheckToken: %s", err.Error())
				return c.JSON(http.StatusUnauthorized, response.Response(err.Error(), nil))
			}

			getSession, err := redisConn.Get(c.Request().Context(), tokenString).Result()
			if err != nil || len(getSession) == 0 {
				log.Errorf("[MiddlewareAdapter-3] CheckToken: %s", err.Error())
				return c.JSON(http.StatusUnauthorized, response.Response(err.Error(), nil))
			}

			jwtUserData := entity.JwtUserData{}
			err = json.Unmarshal([]byte(getSession), &jwtUserData)
			if err != nil {
				log.Errorf("[MiddlewareAdapter-4] CheckToken: %v", err)
				return c.JSON(http.StatusInternalServerError, response.Response(err.Error(), nil))
			}

			path := c.Request().URL.Path
			segments := strings.Split(strings.Trim(path, "/"), "/")
			if jwtUserData.RoleName == "Customer" && segments[0] == "admin" {
				log.Infof("[MiddlewareAdapter-5] CheckToken: %s", "customer cannot access admin routes")
				return c.JSON(http.StatusForbidden, response.Response("customer cannot access admin routes", nil))
			}

			c.Set("user", getSession)
			return next(c)
		}
	}
}

func NewMiddlewareAdapter(cfg *config.Config) MiddlewareAdapterInterface {
	return &middlewareAdapter{
		cfg: cfg,
	}
}
