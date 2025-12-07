package retrievers

import (
	"card-collectors-uk/analysis"
	"card-collectors-uk/card"
	"card-collectors-uk/database"
	"card-collectors-uk/retrievers/ebay"
	"context"
	"fmt"
)

type Retriever interface {
	Name() string
	GetListings(ctx context.Context) (map[card.CanonicalCard][]analysis.Listing, error)
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
