package main

import (
	"context"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
	"go.uber.org/zap"

	_ "github.com/yourorg/weather-api/docs"
	"github.com/yourorg/weather-api/internal/config"
	"github.com/yourorg/weather-api/internal/handler"
	"github.com/yourorg/weather-api/internal/logger"
	"github.com/yourorg/weather-api/internal/middleware"
	"github.com/yourorg/weather-api/internal/monitor"
	"github.com/yourorg/weather-api/internal/weather"
)

func main() {

	cfg, err := config.Load()
	if err != nil {
		panic("config error: " + err.Error())
	}

	if err := logger.Init(cfg.LogLevel, cfg.LogFormat); err != nil {
		panic("logger error: " + err.Error())
	}
	defer logger.Sync()
	log := logger.Get()

	if cfg.AppEnv == "production" {
		gin.SetMode(gin.ReleaseMode)
	}

	weatherSvc := weather.NewService(
		cfg.OpenMeteoBaseURL,
		cfg.OpenMeteoGeocodingURL,
		cfg.HTTPTimeoutSeconds,
		log,
	)

	weatherHandler := handler.NewWeatherHandler(weatherSvc, log)

	r := gin.New()
	r.Use(
		middleware.RequestID(),
		middleware.Recovery(),
		middleware.Logger(),
		monitor.PrometheusMiddleware(),
	)

	monitor.RegisterRoutes(r, cfg.AppName, cfg.AppEnv)

	if cfg.AppEnv != "production" {
		r.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))
		log.Info("swagger UI available at /swagger/index.html")
	}

	v1 := r.Group("/api/v1", middleware.StaticToken(cfg.StaticAPIToken))
	{
		v1.GET("/weather", weatherHandler.GetCurrent)
		v1.GET("/forecast", weatherHandler.GetForecast)
	}

	srv := &http.Server{
		Addr:         ":" + cfg.AppPort,
		Handler:      r,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	go func() {
		log.Info("server starting", zap.String("port", cfg.AppPort), zap.String("env", cfg.AppEnv))
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatal("server error", zap.Error(err))
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Info("shutting down...")
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		log.Error("forced shutdown", zap.Error(err))
	}
	log.Info("server stopped")
}
