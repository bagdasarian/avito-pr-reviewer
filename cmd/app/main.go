package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/bagdasarian/avito-pr-reviewer/internal/config"
	"github.com/bagdasarian/avito-pr-reviewer/internal/db"
	"github.com/bagdasarian/avito-pr-reviewer/internal/handler"
	"github.com/bagdasarian/avito-pr-reviewer/internal/handler/server"
	"github.com/bagdasarian/avito-pr-reviewer/internal/repository/postgres"
	"github.com/bagdasarian/avito-pr-reviewer/internal/service"
)

func main() {
	cfg := config.Load()

	database := db.MustLoad(cfg)
	log.Println("Successfully connected to database!")
	defer database.Close()

	teamRepo := postgres.NewTeamRepository(database)
	userRepo := postgres.NewUserRepository(database)
	pullRequestRepo := postgres.NewPullRequestRepository(database)
	statsRepo := postgres.NewStatsRepository(database)

	teamService := service.NewTeamService(teamRepo, userRepo)
	userService := service.NewUserService(userRepo, pullRequestRepo)
	pullRequestService := service.NewPullRequestService(pullRequestRepo, userRepo, teamRepo)
	statsService := service.NewStatsService(statsRepo)

	h := handler.NewHandler(teamService, userService, pullRequestService, statsService)
	srv := server.NewServer(h, ":8080")

	go func() {
		if err := srv.Start(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Server failed to start: %v", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		log.Fatalf("Server forced to shutdown: %v", err)
	}
}
