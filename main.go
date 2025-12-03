package main

import (
	"card-collectors-uk/database"
	"context"
	"fmt"
	"log"
)

func main() {
	ctx := context.Background()

	if err := database.InitialiseConnection(ctx); err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}

	defer database.CloseConnection()

	fmt.Println("Database connection established")
}
