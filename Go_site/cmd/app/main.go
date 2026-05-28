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

	// репозитории
	userRepo := repository.NewUserRepository(pool)
	machineRepo := repository.NewMachineRepository(pool)
	accountRepo := repository.NewAccountRepository(pool)

	// сервисы
	userSvc := service.NewUserService(userRepo, cfg.HelmChartDir)
	machineSvc := service.NewMachineService(machineRepo, userRepo, cfg.HelmChartDir)

	sessions := service.NewSessionStore()
	authSvc := service.NewAuthService(accountRepo, sessions, userSvc)

	// хендлеры
	userHandler := handlers.NewUserHandler(userSvc)
	machineHandler := handlers.NewMachineHandler(machineSvc)
	authHandler := handlers.NewAuthHandler(authSvc)

	// router
	r := gin.Default()
	r.LoadHTMLGlob("web/templates/*.tmpl")

	routes.Register(r, userHandler, machineHandler, authHandler, sessions)

	log.Printf("starting HTTP server on %s", cfg.HttpAddr)
	if err := r.Run(cfg.HttpAddr); err != nil {
		log.Fatalf("server error: %v", err)
	}
}
