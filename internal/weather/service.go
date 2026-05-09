package weather

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"

	"github.com/yourorg/weather-api/internal/monitor"
	"go.uber.org/zap"
)

// --- DTOs ---

type CurrentWeather struct {
	City        string  `json:"city"`
	Country     string  `json:"country"`
	Latitude    float64 `json:"latitude"`
	Longitude   float64 `json:"longitude"`
	Temperature float64 `json:"temperature_c"`
	Windspeed   float64 `json:"windspeed_kmh"`
	Weathercode int     `json:"weathercode"`
	Description string  `json:"description"`
	Time        string  `json:"time"`
}

type ForecastDay struct {
	Date          string  `json:"date"`
	TempMax       float64 `json:"temp_max_c"`
	TempMin       float64 `json:"temp_min_c"`
	Precipitation float64 `json:"precipitation_mm"`
	Weathercode   int     `json:"weathercode"`
	Description   string  `json:"description"`
}

type Forecast struct {
	City      string        `json:"city"`
	Country   string        `json:"country"`
	Latitude  float64       `json:"latitude"`
	Longitude float64       `json:"longitude"`
	Days      []ForecastDay `json:"days"`
}

type geoResult struct {
	Results []geoLocation `json:"results"`
}

type geoLocation struct {
	Name      string  `json:"name"`
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
	Country   string  `json:"country"`
}

type currentResponse struct {
	Current struct {
		Time        string  `json:"time"`
		Temperature float64 `json:"temperature_2m"`
		Windspeed   float64 `json:"windspeed_10m"`
		Weathercode int     `json:"weathercode"`
	} `json:"current"`
}

type forecastResponse struct {
	Daily struct {
		Time          []string  `json:"time"`
		TempMax       []float64 `json:"temperature_2m_max"`
		TempMin       []float64 `json:"temperature_2m_min"`
		Precipitation []float64 `json:"precipitation_sum"`
		Weathercode   []int     `json:"weathercode"`
	} `json:"daily"`
}

// --- Service ---

type Service struct {
	baseURL      string
	geocodingURL string
	client       *http.Client
	log          *zap.Logger
}

func NewService(baseURL, geocodingURL string, timeoutSec int, log *zap.Logger) *Service {
	return &Service{
		baseURL:      baseURL,
		geocodingURL: geocodingURL,
		client:       &http.Client{Timeout: time.Duration(timeoutSec) * time.Second},
		log:          log,
	}
}

func (s *Service) GetCurrent(city string) (*CurrentWeather, error) {
	loc, err := s.geocode(city)
	if err != nil {
		return nil, err
	}

	start := time.Now()
	endpoint := fmt.Sprintf(
		"%s/forecast?latitude=%.4f&longitude=%.4f&current=temperature_2m,windspeed_10m,weathercode&timezone=auto",
		s.baseURL, loc.Latitude, loc.Longitude,
	)

	var raw currentResponse
	if err := s.get(endpoint, &raw); err != nil {
		monitor.ExternalAPIErrors.WithLabelValues("open-meteo").Inc()
		return nil, fmt.Errorf("open-meteo current: %w", err)
	}
	monitor.ExternalAPIDuration.WithLabelValues("open-meteo", "current").Observe(time.Since(start).Seconds())

	return &CurrentWeather{
		City:        loc.Name,
		Country:     loc.Country,
		Latitude:    loc.Latitude,
		Longitude:   loc.Longitude,
		Temperature: raw.Current.Temperature,
		Windspeed:   raw.Current.Windspeed,
		Weathercode: raw.Current.Weathercode,
		Description: describeCode(raw.Current.Weathercode),
		Time:        raw.Current.Time,
	}, nil
}

func (s *Service) GetForecast(city string, days int) (*Forecast, error) {
	if days < 1 {
		days = 1
	}
	if days > 16 {
		days = 16
	}

	loc, err := s.geocode(city)
	if err != nil {
		return nil, err
	}

	start := time.Now()
	endpoint := fmt.Sprintf(
		"%s/forecast?latitude=%.4f&longitude=%.4f&daily=temperature_2m_max,temperature_2m_min,precipitation_sum,weathercode&forecast_days=%d&timezone=auto",
		s.baseURL, loc.Latitude, loc.Longitude, days,
	)

	var raw forecastResponse
	if err := s.get(endpoint, &raw); err != nil {
		monitor.ExternalAPIErrors.WithLabelValues("open-meteo").Inc()
		return nil, fmt.Errorf("open-meteo forecast: %w", err)
	}
	monitor.ExternalAPIDuration.WithLabelValues("open-meteo", "forecast").Observe(time.Since(start).Seconds())

	forecastDays := make([]ForecastDay, len(raw.Daily.Time))
	for i, t := range raw.Daily.Time {
		code := 0
		if i < len(raw.Daily.Weathercode) {
			code = raw.Daily.Weathercode[i]
		}
		forecastDays[i] = ForecastDay{
			Date:          t,
			TempMax:       safeFloat(raw.Daily.TempMax, i),
			TempMin:       safeFloat(raw.Daily.TempMin, i),
			Precipitation: safeFloat(raw.Daily.Precipitation, i),
			Weathercode:   code,
			Description:   describeCode(code),
		}
	}

	return &Forecast{
		City:      loc.Name,
		Country:   loc.Country,
		Latitude:  loc.Latitude,
		Longitude: loc.Longitude,
		Days:      forecastDays,
	}, nil
}

func (s *Service) geocode(city string) (*geoLocation, error) {
	cleanCity := removeAccents(city)

	start := time.Now()
	endpoint := fmt.Sprintf("%s/search?name=%s&count=1&format=json",
		s.geocodingURL, url.QueryEscape(cleanCity))

	var result geoResult
	if err := s.get(endpoint, &result); err != nil {
		monitor.ExternalAPIErrors.WithLabelValues("open-meteo-geo").Inc()
		return nil, fmt.Errorf("geocoding: %w", err)
	}
	monitor.ExternalAPIDuration.WithLabelValues("open-meteo-geo", "geocode").Observe(time.Since(start).Seconds())

	if len(result.Results) == 0 {
		return nil, fmt.Errorf("city not found: %s", city)
	}
	return &result.Results[0], nil
}

func (s *Service) get(url string, dest any) error {
	s.log.Info("making HTTP request", zap.String("url", url))

	resp, err := s.client.Get(url)
	if err != nil {
		s.log.Error("HTTP request failed", zap.String("url", url), zap.Error(err))
		return err
	}
	defer resp.Body.Close()

	s.log.Info("HTTP response received", zap.Int("status_code", resp.StatusCode), zap.String("url", url))

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		s.log.Error("upstream error", zap.Int("status_code", resp.StatusCode), zap.String("body", string(body)), zap.String("url", url))
		return fmt.Errorf("upstream returned %d: %s", resp.StatusCode, string(body))
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		s.log.Error("failed to read response body", zap.Error(err))
		return err
	}

	s.log.Info("response body", zap.String("body", string(body)))

	return json.Unmarshal(body, dest)
}

func safeFloat(slice []float64, i int) float64 {
	if i < len(slice) {
		return slice[i]
	}
	return 0
}

func describeCode(code int) string {
	descriptions := map[int]string{
		0: "Céu limpo", 1: "Principalmente limpo", 2: "Parcialmente nublado", 3: "Nublado",
		45: "Neblina", 48: "Neblina congelada",
		51: "Garoa leve", 53: "Garoa moderada", 55: "Garoa densa",
		61: "Chuva leve", 63: "Chuva moderada", 65: "Chuva pesada",
		71: "Neve leve", 73: "Neve moderada", 75: "Neve pesada",
		77: "Grãos de neve",
		80: "Chuva leve", 81: "Chuva moderada", 82: "Chuva violenta",
		85: "Neve leve em pancadas", 86: "Neve pesada em pancadas",
		95: "Tempestade", 96: "Tempestade com granizo", 99: "Tempestade com granizo pesado",
	}
	if d, ok := descriptions[code]; ok {
		return d
	}
	return "Desconhecido"
}

func removeAccents(s string) string {
	replacements := map[rune]string{
		'á': "a", 'à': "a", 'â': "a", 'ã': "a", 'ä': "a", 'å': "a",
		'é': "e", 'è': "e", 'ê': "e", 'ë': "e",
		'í': "i", 'ì': "i", 'î': "i", 'ï': "i",
		'ó': "o", 'ò': "o", 'ô': "o", 'õ': "o", 'ö': "o",
		'ú': "u", 'ù': "u", 'û': "u", 'ü': "u",
		'ý': "y", 'ÿ': "y",
		'Á': "A", 'À': "A", 'Â': "A", 'Ã': "A", 'Ä': "A", 'Å': "A",
		'É': "E", 'È': "E", 'Ê': "E", 'Ë': "E",
		'Í': "I", 'Ì': "I", 'Î': "I", 'Ï': "I",
		'Ó': "O", 'Ò': "O", 'Ô': "O", 'Õ': "O", 'Ö': "O",
		'Ú': "U", 'Ù': "U", 'Û': "U", 'Ü': "U",
		'Ý': "Y", 'Ÿ': "Y",
		'ç': "c", 'Ç': "C",
		'ñ': "n", 'Ñ': "N",
	}

	result := ""
	for _, r := range s {
		if replacement, ok := replacements[r]; ok {
			result += replacement
		} else {
			result += string(r)
		}
	}
	return result
}
