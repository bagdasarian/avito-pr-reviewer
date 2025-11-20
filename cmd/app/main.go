package main

import (
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/bagdasarian/avito-pr-reviewer/internal/config"
	"github.com/bagdasarian/avito-pr-reviewer/internal/db"
)

func main() {
	cfg := config.Load()

	database := db.MustLoad(cfg)
	log.Println("Successfully connected to database!")
	defer database.Close()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down...")
}
