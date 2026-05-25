package main

import (
	"context"
	"log"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/grooptroop/KyNa/Go_site/internal/config"
	"github.com/grooptroop/KyNa/Go_site/internal/http/handlers"
	"github.com/grooptroop/KyNa/Go_site/internal/http/routes"
	"github.com/grooptroop/KyNa/Go_site/internal/repository"
	"github.com/grooptroop/KyNa/Go_site/internal/service"
	"github.com/grooptroop/KyNa/Go_site/migrations"
)

func main() {
	cfg := config.Load()
	ctx := context.Background()

	pool, err := pgxpool.New(ctx, cfg.PostgresDSN)
	if err != nil {
		log.Fatalf("failed to connect postgres: %v", err)
	}
	defer pool.Close()

	if err := pool.Ping(ctx); err != nil {
		log.Fatalf("failed to ping postgres: %v", err)
	}

	if err := migrations.RunMigrations(ctx, pool); err != nil {
		log.Fatalf("failed to run migrations: %v", err)
	}

	userRepo := repository.NewUserRepository(pool)
	userSvc := service.NewUserService(userRepo, cfg.HelmChartDir)
	userHandler := handlers.NewUserHandler(userSvc)

	r := gin.Default()
	r.LoadHTMLGlob("web/templates/*.tmpl")

	routes.Register(r, userHandler)

	log.Printf("starting HTTP server on %s", cfg.HttpAddr)
	if err := r.Run(cfg.HttpAddr); err != nil {
		log.Fatalf("server error: %v", err)
	}
}
