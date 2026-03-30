package main

import (
	"context"
	"log"
	"net/http"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"kr-metro-api/config"
	"kr-metro-api/db"
	"kr-metro-api/handler"
	"kr-metro-api/middleware"
	"kr-metro-api/repository"
	"kr-metro-api/sync"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	cfg := config.Load()

	pool, err := db.NewPool(ctx, cfg.DSN)
	if err != nil {
		log.Fatalf("DB connection failed: %v", err)
	}
	defer pool.Close()

	stationRepo := repository.NewStationRepo(pool)
	lineRepo := repository.NewLineRepo(pool)
	transferRepo := repository.NewTransferRepo(pool)
	housingRepo := repository.NewHousingRepo(pool)

	stationH := handler.NewStationHandler(stationRepo)
	lineH := handler.NewLineHandler(lineRepo)
	transferH := handler.NewTransferHandler(transferRepo)
	housingH := handler.NewHousingHandler(housingRepo)

	r := gin.Default()
	r.Use(middleware.CORS(cfg.CORSOrigins))

	v1 := r.Group("/api/v1")
	{
		v1.GET("/stations", stationH.List)
		v1.GET("/stations/:id", stationH.GetByID)
		v1.GET("/stations/search", stationH.Search)
		v1.GET("/stations/nearby", stationH.Nearby)
		v1.GET("/lines", lineH.List)
		v1.GET("/lines/:id/stations", lineH.ListStations)
		v1.GET("/lines/:id/geometry", lineH.GetGeometry)
		v1.GET("/transfers/:station_id", transferH.GetByStation)
		v1.GET("/housings", housingH.List)
		v1.GET("/housings/:home_code", housingH.GetByHomeCode)
		v1.GET("/housings/:home_code/nearby-stations", housingH.NearbyStations)
	}
	r.GET("/healthz", handler.Health(pool))

	if cfg.SyncEnabled {
		housingClient := sync.NewHousingClient()
		housingSync := sync.NewHousingSync(housingClient, housingRepo,
			sync.WithInterval(time.Duration(cfg.SyncIntervalMins)*time.Minute),
		)
		go housingSync.Start(ctx)

		syncH := handler.NewSyncHandler(housingSync, cfg.AdminKey)
		admin := r.Group("/api/v1/admin")
		admin.POST("/sync/housing", syncH.TriggerHousingSync)
		admin.GET("/sync/status", syncH.SyncStatus)
	}

	srv := &http.Server{Addr: ":" + cfg.Port, Handler: r}

	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("listen: %v", err)
		}
	}()
	log.Printf("Server started on :%s", cfg.Port)

	<-ctx.Done()
	log.Println("Shutting down...")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Fatalf("shutdown: %v", err)
	}
	log.Println("Server stopped")
}
