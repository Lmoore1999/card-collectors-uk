package main

import (
	"card-collectors-uk/config"
	"card-collectors-uk/database"
	"card-collectors-uk/retrievers"
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"time"
)

func main() {
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()

	config.InitialiseConfig()
	cfg := config.GetConfig()

	if err := database.InitialiseConnection(ctx); err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}

	defer database.CloseConnection()
	fmt.Println("Database connection established")

	players, err := database.GetAllPlayers(ctx)
	if err != nil {
		log.Fatalf("Failed to get all players: %v", err)
	}

	sets, err := database.GetAllSets(ctx)
	if err != nil {
		log.Fatalf("Failed to get all sets: %v", err)
	}

	fmt.Printf("Players found: %d\nSets found: %d\n", len(players), len(sets))

	initialisedRetrievers, err := retrievers.InitialiseRetrievers(ctx, sets, players)
	if err != nil {
		log.Fatalf("Failed to initialise retreivers: %v", err)
	}

	go startNewListingsPoller(ctx, cfg, initialisedRetrievers)
	select {}
}

func startNewListingsPoller(ctx context.Context, cfg *config.Config, retrievers []retrievers.Retriever) {
	// Run immediately
	for _, retriever := range retrievers {
		listings, err := retriever.GetListings(ctx)
		if err != nil {
			log.Printf("Failed to get listings: %v", err)
			continue
		}
		fmt.Printf("Retrieved %d listings\n", len(listings))
	}

	// Start ticker for scheduled polling
	ticker := time.NewTicker(cfg.NewListingsPollingInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			log.Println("Shutting down new listings poller")
			return
		case <-ticker.C:
			for _, retriever := range retrievers {
				listings, err := retriever.GetListings(ctx)
				if err != nil {
					log.Printf("Failed to get listings: %v", err)
					continue
				}
				fmt.Printf("Retrieved %d listings\n", len(listings))
			}
		}
	}
}
