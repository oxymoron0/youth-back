package config

import (
	"log"
	"os"
	"strconv"
)

type Config struct {
	DSN               string
	Port              string
	CORSOrigins       string
	SyncEnabled       bool
	SyncIntervalMins  int
	AdminKey          string
	NaverClientID     string
	NaverClientSecret string
}

func Load() Config {
	c := Config{
		DSN:               os.Getenv("KR_METRO_DSN"),
		Port:              os.Getenv("PORT"),
		CORSOrigins:       os.Getenv("CORS_ORIGINS"),
		SyncEnabled:       os.Getenv("HOUSING_SYNC_ENABLED") != "false",
		SyncIntervalMins:  30,
		AdminKey:          os.Getenv("ADMIN_KEY"),
		NaverClientID:     os.Getenv("NAVER_MAPS_CLIENT_ID"),
		NaverClientSecret: os.Getenv("NAVER_MAPS_CLIENT_SECRET"),
	}
	if c.Port == "" {
		c.Port = "8080"
	}
	if c.CORSOrigins == "" {
		c.CORSOrigins = "http://localhost:5173"
	}
	if v, err := strconv.Atoi(os.Getenv("HOUSING_SYNC_INTERVAL_MINS")); err == nil && v > 0 {
		c.SyncIntervalMins = v
	}
	if c.DSN == "" {
		log.Fatal("KR_METRO_DSN environment variable must be set")
	}
	return c
}
