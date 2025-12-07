package database

import (
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
