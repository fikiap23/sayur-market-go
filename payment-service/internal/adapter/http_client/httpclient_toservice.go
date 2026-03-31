package httpclient

import (
	"bytes"
	"io"
	"net/http"
	"payment-service/config"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/labstack/gommon/log"
)

type HttpClientToService interface {
	Connect()
	CallURL(method, url string, header map[string]string, rawData []byte) (*http.Response, error)
}

type Options struct {
	timeout int
	http    *http.Client
	logger  echo.Logger
}

type loggingTransport struct {
	logger echo.Logger
}

func NewHttpClient(cfg *config.Config) HttpClientToService {
	opt := new(Options)
	opt.timeout = cfg.App.ServerTimeOut
	return opt
}

func (o *Options) Connect() {
	e := echo.New()
	e.Logger.SetLevel(log.INFO)

	httpClient := &http.Client{
		Timeout:   time.Duration(o.timeout) * time.Second,
		Transport: &loggingTransport{e.Logger},
	}

	o.http = httpClient
	o.logger = e.Logger
}

func (o *Options) CallURL(method, url string, header map[string]string, rawData []byte) (*http.Response, error) {
	o.Connect()
	req, err := http.NewRequest(method, url, bytes.NewBuffer(rawData))
	if err != nil {
		o.logger.Errorj(log.JSON{
			"message": "[CallURL-1] Failed To Prepare Request Client HTTP",
			"error":   err.Error(),
		})
		return nil, err
	}

	if len(header) > 0 {
		for key, value := range header {
			req.Header.Set(key, value)
		}
	}

	resp, err := o.http.Do(req)
	if err != nil {
		o.logger.Errorj(log.JSON{
			"message": "[CallURL-2] Failed To DO Request Client HTTP",
			"error":   err.Error(),
		})
		return nil, err
	}

	return resp, nil
}

func (lt *loggingTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	// Logging sebelum request
	lt.logger.Infof("Making request to: %s %s", req.Method, req.URL)
	lt.logger.Infof("Request Headers: %+v", req.Header)

	// Mengganti request body karena sudah dibaca dalam fungsi logging
	reqBody, err := io.ReadAll(req.Body)
	if err != nil {
		return nil, err
	}
	req.Body = io.NopCloser(bytes.NewBuffer(reqBody))
	lt.logger.Infof("Request Body: %s", reqBody)

	resp, err := http.DefaultTransport.RoundTrip(req)
	if err != nil {
		lt.logger.Infof("Request failed: %v", err)
		return nil, err
	}

	// Logging setelah menerima respons
	lt.logger.Infof("Received response with status: %s", resp.Status)
	lt.logger.Infof("Response Headers: %+v", resp.Header)

	// Menampilkan Response Body (jika ada)
	respBody, err := io.ReadAll(resp.Body)
	if err == nil {
		lt.logger.Infof("Response Body: %s", respBody)
	}
	resp.Body = io.NopCloser(bytes.NewBuffer(respBody))

	return resp, nil
}
