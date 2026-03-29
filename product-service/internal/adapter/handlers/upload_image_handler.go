package handlers

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"product-service/config"
	"product-service/internal/adapter"
	"product-service/internal/adapter/handlers/response"
	"product-service/internal/adapter/storage"
	"time"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"github.com/labstack/gommon/log"
)

type UploadImageInterface interface {
	UploadImage(c echo.Context) error
}

type uploadImage struct {
	storageHandler storage.SupabaseInterface
}

// UploadImage implements UploadImageInterface.
func (u *uploadImage) UploadImage(c echo.Context) error {
	var resp = response.DefaultResponse{}

	file, err := c.FormFile("image")
	if err != nil {
		log.Errorf("[UploadImage-1] UploadImage: %v", err)
		resp.Message = err.Error()
		resp.Data = nil
		return c.JSON(http.StatusUnprocessableEntity, resp)
	}

	src, err := file.Open()
	if err != nil {
		log.Errorf("[UploadImage-2] UploadImage: %v", err)
		resp.Message = err.Error()
		resp.Data = nil
		return c.JSON(http.StatusBadRequest, resp)
	}

	defer src.Close()

	fileBuffer := new(bytes.Buffer)
	_, err = io.Copy(fileBuffer, src)
	if err != nil {
		log.Errorf("[UploadImage-3] UploadImage: %v", err)
		resp.Message = err.Error()
		resp.Data = nil
		return c.JSON(http.StatusInternalServerError, resp)
	}

	newFileName := fmt.Sprintf("%s_%d%s", uuid.New().String(), time.Now().Unix(), getExtension(file.Filename))

	uploadPath := fmt.Sprintf("public/uploads/%s", newFileName)
	url, err := u.storageHandler.UploadFile(uploadPath, fileBuffer)
	if err != nil {
		log.Errorf("[UploadImage-4] UploadImage: %v", err)
		resp.Message = err.Error()
		resp.Data = nil
		return c.JSON(http.StatusInternalServerError, resp)
	}

	resp.Message = "Success"
	resp.Data = map[string]string{"image_url": url}

	return c.JSON(http.StatusOK, resp)
}

func getExtension(fileName string) string {
	ext := "." + fileName[len(fileName)-3:] // Ambil 3 karakter terakhir untuk ekstensi
	if len(fileName) > 4 && fileName[len(fileName)-4] == '.' {
		ext = "." + fileName[len(fileName)-4:]
	}
	return ext
}

func NewUploadImage(e *echo.Echo, cfg *config.Config, storageHandler storage.SupabaseInterface) UploadImageInterface {
	res := &uploadImage{
		storageHandler: storageHandler,
	}

	mid := adapter.NewMiddlewareAdapter(cfg)
	e.POST("/admin/image-upload", res.UploadImage, mid.CheckToken())

	return res
}
