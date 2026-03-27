package handler

import (
	"encoding/json"
	"net/http"
	"strconv"
	"user-service/config"
	"user-service/internal/adapter"
	"user-service/internal/adapter/handler/request"
	"user-service/internal/adapter/handler/response"
	"user-service/internal/core/domain/entity"
	"user-service/internal/core/service"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/labstack/gommon/log"
)

type RoleHandlerInterface interface {
	GetAll(c echo.Context) error
	GetByID(c echo.Context) error
	Create(c echo.Context) error
	Delete(c echo.Context) error
	Update(c echo.Context) error
}

type roleHandler struct {
	roleService service.RoleServiceInterface
}

// Create implements RoleHandlerInterface.
func (r *roleHandler) Create(c echo.Context) error {
	var (
		req         = request.RoleRequest{}
		resp        = response.DefaultResponse{}
		ctx         = c.Request().Context()
		jwtUserData = entity.JwtUserData{}
	)

	user := c.Get("user").(string)
	if user == "" {
		log.Errorf("[RoleHandler-1] Create: %s", "data token not found")
		resp.Message = "data token not found"
		resp.Data = nil
		return c.JSON(http.StatusNotFound, resp)
	}

	err := json.Unmarshal([]byte(user), &jwtUserData)
	if err != nil {
		log.Errorf("[RoleHandler-2] Create: %v", err)
		resp.Message = err.Error()
		resp.Data = nil
		return c.JSON(http.StatusBadRequest, resp)
	}

	if jwtUserData.RoleName != "Super Admin" {
		log.Errorf("[RoleHandler-3] Create: %s", "only Super Admin can access API role")
		resp.Message = "only Super Admin can access API role"
		resp.Data = nil
		return c.JSON(http.StatusForbidden, resp)
	}

	if err := c.Bind(&req); err != nil {
		log.Errorf("[RoleHandler-4] Create: %v", err)
		resp.Message = err.Error()
		resp.Data = nil
		return c.JSON(http.StatusUnprocessableEntity, resp)
	}

	if err := c.Validate(req); err != nil {
		log.Errorf("[RoleHandler-5] Create: %v", err)
		resp.Message = err.Error()
		resp.Data = nil
		return c.JSON(http.StatusUnprocessableEntity, resp)
	}

	roleEntity := entity.RoleEntity{
		Name: req.Name,
	}

	err = r.roleService.Create(ctx, roleEntity)
	if err != nil {
		log.Errorf("[RoleHandler-3] Create: %v", err)
		resp.Message = err.Error()
		resp.Data = nil
		return c.JSON(http.StatusInternalServerError, resp)
	}

	resp.Message = "Success"
	resp.Data = nil
	return c.JSON(http.StatusCreated, resp)
}

// Delete implements RoleHandlerInterface.
func (r *roleHandler) Delete(c echo.Context) error {
	var (
		resp        = response.DefaultResponse{}
		ctx         = c.Request().Context()
		jwtUserData = entity.JwtUserData{}
	)

	user := c.Get("user").(string)
	if user == "" {
		log.Errorf("[RoleHandler-1] Delete: %s", "data token not found")
		resp.Message = "data token not found"
		resp.Data = nil
		return c.JSON(http.StatusNotFound, resp)
	}

	err := json.Unmarshal([]byte(user), &jwtUserData)
	if err != nil {
		log.Errorf("[RoleHandler-2] Delete: %v", err)
		resp.Message = err.Error()
		resp.Data = nil
		return c.JSON(http.StatusBadRequest, resp)
	}

	if jwtUserData.RoleName != "Super Admin" {
		log.Errorf("[RoleHandler-3] Delete: %s", "only Super Admin can access API role")
		resp.Message = "only Super Admin can access API role"
		resp.Data = nil
		return c.JSON(http.StatusForbidden, resp)
	}

	roleIDString := c.Param("id")
	if roleIDString == "" {
		log.Infof("[RoleHandler-4] Delete: %s", "missing or invalid role ID")
		resp.Message = "missing or invalid role ID"
		resp.Data = nil
		return c.JSON(http.StatusBadRequest, resp)
	}

	roleID, err := strconv.Atoi(roleIDString)
	if err != nil {
		log.Errorf("[RoleHandler-5] Delete: %v", err)
		resp.Message = err.Error()
		resp.Data = nil
		return c.JSON(http.StatusBadRequest, resp)
	}

	err = r.roleService.Delete(ctx, int64(roleID))
	if err != nil {
		log.Errorf("[RoleHandler-6] Delete: %v", err)
		if err.Error() == "404" {
			resp.Message = "Role not found"
			resp.Data = nil
			return c.JSON(http.StatusNotFound, resp)
		}
		resp.Message = err.Error()
		resp.Data = nil
		return c.JSON(http.StatusInternalServerError, resp)
	}

	resp.Message = "Role deleted successfully"
	resp.Data = nil
	return c.JSON(http.StatusOK, resp)
}

// GetAll implements RoleHandlerInterface.
func (r *roleHandler) GetAll(c echo.Context) error {
	var (
		respRole = []response.RoleResponse{}
		resp     = response.DefaultResponse{}
		ctx      = c.Request().Context()
	)

	user := c.Get("user").(string)
	if user == "" {
		log.Errorf("[RoleHandler-1] GetAll: %s", "data token not found")
		resp.Message = "data token not found"
		resp.Data = nil
		return c.JSON(http.StatusNotFound, resp)
	}

	search := c.QueryParam("search")

	roles, err := r.roleService.GetAll(ctx, search)
	if err != nil {
		log.Errorf("[RoleHandler-4] GetAll: %v", err)
		resp.Message = err.Error()
		resp.Data = nil
		return c.JSON(http.StatusInternalServerError, resp)
	}

	for _, role := range roles {
		respRole = append(respRole, response.RoleResponse{
			ID:   role.ID,
			Name: role.Name,
		})
	}

	resp.Message = "success"
	resp.Data = respRole
	return c.JSON(http.StatusOK, resp)
}

// GetByID implements RoleHandlerInterface.
func (r *roleHandler) GetByID(c echo.Context) error {
	var (
		respRole    = response.RoleResponse{}
		resp        = response.DefaultResponse{}
		ctx         = c.Request().Context()
		jwtUserData = entity.JwtUserData{}
	)

	user := c.Get("user").(string)
	if user == "" {
		log.Errorf("[RoleHandler-1] GetByID: %s", "data token not found")
		resp.Message = "data token not found"
		resp.Data = nil
		return c.JSON(http.StatusNotFound, resp)
	}

	err := json.Unmarshal([]byte(user), &jwtUserData)
	if err != nil {
		log.Errorf("[RoleHandler-2] GetByID: %v", err)
		resp.Message = err.Error()
		resp.Data = nil
		return c.JSON(http.StatusBadRequest, resp)
	}

	if jwtUserData.RoleName != "Super Admin" {
		log.Errorf("[RoleHandler-3] GetByID: %s", "only Super Admin can access API role")
		resp.Message = "only Super Admin can access API role"
		resp.Data = nil
		return c.JSON(http.StatusForbidden, resp)
	}

	roleIDString := c.Param("id")
	if roleIDString == "" {
		log.Infof("[RoleHandler-4] GetByID: %s", "missing or invalid role ID")
		resp.Message = "missing or invalid role ID"
		resp.Data = nil
		return c.JSON(http.StatusBadRequest, resp)
	}

	roleID, err := strconv.Atoi(roleIDString)
	if err != nil {
		log.Errorf("[RoleHandler-5] GetByID: %v", err)
		resp.Message = err.Error()
		resp.Data = nil
		return c.JSON(http.StatusBadRequest, resp)
	}

	role, err := r.roleService.GetByID(ctx, int64(roleID))
	if err != nil {
		log.Errorf("[RoleHandler-6] GetByID: %v", err)
		if err.Error() == "404" {
			resp.Message = "Role not found"
			resp.Data = nil
			return c.JSON(http.StatusNotFound, resp)
		}
		resp.Message = err.Error()
		resp.Data = nil
		return c.JSON(http.StatusInternalServerError, resp)
	}

	respRole.ID = role.ID
	respRole.Name = role.Name
	resp.Message = "success"
	resp.Data = respRole
	return c.JSON(http.StatusOK, resp)
}

// Update implements RoleHandlerInterface.
func (r *roleHandler) Update(c echo.Context) error {
	var (
		req         = request.RoleRequest{}
		resp        = response.DefaultResponse{}
		ctx         = c.Request().Context()
		jwtUserData = entity.JwtUserData{}
	)

	user := c.Get("user").(string)
	if user == "" {
		log.Errorf("[RoleHandler-1] Update: %s", "data token not found")
		resp.Message = "data token not found"
		resp.Data = nil
		return c.JSON(http.StatusNotFound, resp)
	}

	err := json.Unmarshal([]byte(user), &jwtUserData)
	if err != nil {
		log.Errorf("[RoleHandler-2] Update: %v", err)
		resp.Message = err.Error()
		resp.Data = nil
		return c.JSON(http.StatusBadRequest, resp)
	}

	if jwtUserData.RoleName != "Super Admin" {
		log.Errorf("[RoleHandler-3] Update: %s", "only Super Admin can access API role")
		resp.Message = "only Super Admin can access API role"
		resp.Data = nil
		return c.JSON(http.StatusForbidden, resp)
	}

	roleIDString := c.Param("id")
	if roleIDString == "" {
		log.Infof("[RoleHandler-4] Update: %s", "missing or invalid role ID")
		resp.Message = "missing or invalid role ID"
		resp.Data = nil
		return c.JSON(http.StatusBadRequest, resp)
	}

	roleID, err := strconv.Atoi(roleIDString)
	if err != nil {
		log.Errorf("[RoleHandler-5] Update: %v", err)
		resp.Message = err.Error()
		resp.Data = nil
		return c.JSON(http.StatusBadRequest, resp)
	}

	if err := c.Bind(&req); err != nil {
		log.Errorf("[RoleHandler-6] Update: %v", err)
		resp.Message = err.Error()
		resp.Data = nil
		return c.JSON(http.StatusUnprocessableEntity, resp)
	}

	if err := c.Validate(req); err != nil {
		log.Errorf("[RoleHandler-7] Update: %v", err)
		resp.Message = err.Error()
		resp.Data = nil
		return c.JSON(http.StatusUnprocessableEntity, resp)
	}

	reqEntity := entity.RoleEntity{
		ID:   int64(roleID),
		Name: req.Name,
	}

	err = r.roleService.Update(ctx, reqEntity)
	if err != nil {
		log.Errorf("[RoleHandler-8] Update: %v", err)
		if err.Error() == "404" {
			resp.Message = "Role not found"
			resp.Data = nil
			return c.JSON(http.StatusNotFound, resp)
		}
		resp.Message = err.Error()
		resp.Data = nil
		return c.JSON(http.StatusInternalServerError, resp)
	}

	resp.Message = "Role updated successfully"
	resp.Data = nil

	return c.JSON(http.StatusOK, resp)
}

func NewRoleHandler(e *echo.Echo, roleService service.RoleServiceInterface, cfg *config.Config, jwtService service.JwtServiceInterface) RoleHandlerInterface {
	role := &roleHandler{roleService: roleService}

	e.Use(middleware.Recover())
	mid := adapter.NewMiddlewareAdapter(cfg, jwtService)
	adminGroup := e.Group("/admin", mid.CheckToken())
	adminGroup.GET("/roles", role.GetAll)
	adminGroup.POST("/roles", role.Create)
	adminGroup.PUT("/roles/{id}", role.Update)
	adminGroup.DELETE("/roles/{id}", role.Delete)
	adminGroup.GET("/roles/{id}", role.GetByID)

	return role
}
