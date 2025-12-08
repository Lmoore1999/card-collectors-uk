package retrievers

import (
	"card-collectors-uk/database"
	"card-collectors-uk/retrievers/ebay"
	"context"
	"fmt"
)

type Retriever interface {
	Name() string
	GetAndStoreListings(ctx context.Context)
}

func InitialiseRetrievers(ctx context.Context, sets []database.SetRow, players []database.PlayerRow) ([]Retriever, error) {
	var retrievers []Retriever

	ebayRetriever, err := ebay.InitialiseRetriever(ctx, sets, players)
	if err != nil {
		return nil, err
	}

	retrievers = append(retrievers, ebayRetriever)

	fmt.Println("Retrievers Initialised")
	return retrievers, nil
}
