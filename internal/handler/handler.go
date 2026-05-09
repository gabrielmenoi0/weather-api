package handler

import (
	"errors"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/yourorg/weather-api/internal/weather"
	"go.uber.org/zap"
)

type WeatherHandler struct {
	svc *weather.Service
	log *zap.Logger
}

func NewWeatherHandler(svc *weather.Service, log *zap.Logger) *WeatherHandler {
	return &WeatherHandler{svc: svc, log: log}
}

func (h *WeatherHandler) GetCurrent(c *gin.Context) {
	city := strings.TrimSpace(c.Query("city"))
	if city == "" {
		c.JSON(http.StatusBadRequest, ErrorResponse{Message: "parametro da consulta 'city' é obrigatório"})
		return
	}

	result, err := h.svc.GetCurrent(city)
	if err != nil {
		h.handleServiceError(c, err)
		return
	}

	c.JSON(http.StatusOK, result)
}

func (h *WeatherHandler) GetForecast(c *gin.Context) {
	city := strings.TrimSpace(c.Query("city"))
	if city == "" {
		c.JSON(http.StatusBadRequest, ErrorResponse{Message: "parametro da consulta 'city' é obrigatório"})
		return
	}

	days := 7
	if d := c.Query("days"); d != "" {
		parsed, err := strconv.Atoi(d)
		if err != nil || parsed < 1 || parsed > 16 {
			c.JSON(http.StatusBadRequest, ErrorResponse{Message: "parametro 'days' deve ser um inteiro entre 1 e 16"})
			return
		}
		days = parsed
	}

	result, err := h.svc.GetForecast(city, days)
	if err != nil {
		h.handleServiceError(c, err)
		return
	}

	c.JSON(http.StatusOK, result)
}

func (h *WeatherHandler) handleServiceError(c *gin.Context, err error) {
	requestID, _ := c.Get("request_id")
	h.log.Error("service error",
		zap.Error(err),
		zap.String("request_id", requestID.(string)),
	)

	msg := err.Error()
	if errors.Is(err, errCityNotFound(msg)) || strings.Contains(msg, "city not found") {
		c.JSON(http.StatusNotFound, ErrorResponse{Message: msg})
		return
	}
	c.JSON(http.StatusInternalServerError, ErrorResponse{Message: "failed to fetch weather data"})
}

type errCityNotFound string

func (e errCityNotFound) Error() string { return string(e) }

type ErrorResponse struct {
	Message string `json:"message" example:"city not found: Atlantis"`
}
