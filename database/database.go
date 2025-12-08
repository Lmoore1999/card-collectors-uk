package database

import (
	"card-collectors-uk/analysis"
	"context"
	"fmt"
)

type PlayerRow struct {
	FirstName  string
	SecondName string
}

func GetAllPlayers(ctx context.Context) ([]PlayerRow, error) {
	rows, err := executeFunction(ctx, "Player_GetAll")
	if err != nil {
		return nil, err
	}

	defer rows.Close()

	var players []PlayerRow
	for rows.Next() {
		var p PlayerRow
		if err := rows.Scan(&p.FirstName, &p.SecondName); err != nil {
			return nil, err
		}
		players = append(players, p)
	}

	return players, nil
}

type SetRow struct {
	CompanyName string
	SetName     string
	SubsetName  string
}

func GetAllSets(ctx context.Context) ([]SetRow, error) {
	rows, err := executeFunction(ctx, "Set_GetAll")
	if err != nil {
		return nil, err
	}

	defer rows.Close()

	var sets []SetRow
	for rows.Next() {
		var s SetRow
		if err := rows.Scan(&s.CompanyName, &s.SetName, &s.SubsetName); err != nil {
			return nil, err
		}
		sets = append(sets, s)
	}

	return sets, nil
}

type APICredentials struct {
	Username          string
	EncryptedPassword []byte
}

func GetAPICredentialsByMarketplaceName(ctx context.Context, marketplaceName string) (APICredentials, error) {
	var credentials APICredentials

	query := `SELECT username, password FROM get_api_credentials($1)`
	err := ConnectionPool.QueryRow(ctx, query, marketplaceName).Scan(&credentials.Username, &credentials.EncryptedPassword)
	if err != nil {
		return credentials, fmt.Errorf("failed to get API credentials: %w", err)
	}

	return credentials, nil
}

func InsertListingWithCanonicalCard(ctx context.Context, canonicalCardKey string, listing analysis.Listing) (int, error) {
	var listingID int

	query := `SELECT insert_listing_with_canonical($1,$2,$3,$4,$5,$6,$7,$8,$9,$10);`

	err := ConnectionPool.QueryRow(
		ctx,
		query,
		canonicalCardKey,
		listing.MarketplaceListingID,
		listing.Marketplace,
		listing.Title,
		listing.Description,
		listing.URL,
		listing.Price,
		listing.CurrencyCode,
		listing.ShippingCost,
		listing.ListingDate).Scan(&listingID)
	if err != nil {
		return 0, fmt.Errorf("failed to insert listing: %w", err)
	}

	return listingID, nil
}
