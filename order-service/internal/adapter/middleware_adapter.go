package adapter

import (
	"encoding/json"
	"math"
	"net/http"
	"order-service/config"
	"order-service/internal/adapter/handlers/response"
	"order-service/internal/core/domain/entity"
	"strconv"
	"strings"

	"github.com/golang-jwt/jwt/v5"
	"github.com/labstack/echo/v4"
	"github.com/labstack/gommon/log"
)

type MiddlewareAdapterInterface interface {
	CheckToken() echo.MiddlewareFunc
	DistanceCheck() echo.MiddlewareFunc
}

type middlewareAdapter struct {
	cfg *config.Config
}

func (m *middlewareAdapter) HaversineDistance(lat1, lon1, lat2, lon2 float64) float64 {
	const R = 6371 // radius bumi dalam kilometer

	dLat := (lat2 - lat1) * (math.Pi / 180)
	dLon := (lon2 - lon1) * (math.Pi / 180)

	a := math.Sin(dLat/2)*math.Sin(dLat/2) +
		math.Cos(lat1*(math.Pi/180))*math.Cos(lat2*(math.Pi/180))*
			math.Sin(dLon/2)*math.Sin(dLon/2)

	c := 2 * math.Atan2(math.Sqrt(a), math.Sqrt(1-a))

	return R * c //kilometer
}

// DistanceCheck implements MiddlewareAdapterInterface.
func (m *middlewareAdapter) DistanceCheck() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			latParam := c.QueryParam("lat")
			lonParam := c.QueryParam("lng")
			if latParam == "" || lonParam == "" {
				log.Errorf("[MiddlewareAdapter-1] DistanceCheck: %s", "missing or invalid lat or lng")
				return c.JSON(http.StatusBadRequest, response.ResponseError("missing or invalid lat or lng"))
			}

			lat, err1 := strconv.ParseFloat(latParam, 64)
			lng, err2 := strconv.ParseFloat(lonParam, 64)

			if err1 != nil || err2 != nil {
				log.Errorf("[MiddlewareAdapter-1] DistanceCheck: %s", "missing or invalid lat or lng")
				return c.JSON(http.StatusBadRequest, response.ResponseError("missing or invalid lat or lng"))
			}

			latRef, _ := strconv.ParseFloat(m.cfg.App.LatitudeRef, 64)
			lngRef, _ := strconv.ParseFloat(m.cfg.App.LongitudeRef, 64)
			distance := m.HaversineDistance(latRef, lngRef, lat, lng)
			if distance > float64(m.cfg.App.MaxDistance) {
				log.Errorf("[MiddlewareAdapter-1] DistanceCheck: %s", "distance too far")
				return c.JSON(http.StatusBadRequest, response.ResponseError("distance too far"))
			}

			return next(c)
		}
	}
}

// CheckToken implements MiddlewareAdapterInterface.
func (m *middlewareAdapter) CheckToken() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			redisConn := config.NewConfig().NewRedisClient()
			authHeader := c.Request().Header.Get("Authorization")
			if authHeader == "" {
				log.Errorf("[MiddlewareAdapter-1] CheckToken: %s", "missing or invalid token")
				return c.JSON(http.StatusUnauthorized, response.ResponseError("missing or invalid token"))
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
				return c.JSON(http.StatusUnauthorized, response.ResponseError(err.Error()))
			}

			getSession, err := redisConn.Get(c.Request().Context(), tokenString).Result()
			if err != nil || len(getSession) == 0 {
				log.Errorf("[MiddlewareAdapter-3] CheckToken: %s", err.Error())
				return c.JSON(http.StatusUnauthorized, response.ResponseError(err.Error()))
			}

			jwtUserData := entity.JwtUserData{}
			err = json.Unmarshal([]byte(getSession), &jwtUserData)
			if err != nil {
				log.Errorf("[MiddlewareAdapter-4] CheckToken: %v", err)
				return c.JSON(http.StatusInternalServerError, response.ResponseError(err.Error()))
			}

			path := c.Request().URL.Path
			segments := strings.Split(strings.Trim(path, "/"), "/")
			if jwtUserData.RoleName == "Customer" && segments[0] == "admin" {
				log.Infof("[MiddlewareAdapter-5] CheckToken: %s", "customer cannot access admin routes")
				return c.JSON(http.StatusForbidden, response.ResponseError("customer cannot access admin routes"))
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
