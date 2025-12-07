package analysis

import "time"

type Listing struct {
	MarketplaceListingID string
	Marketplace          string
	Title                string
	Description          string
	URL                  string
	Price                float64
	CurrencyCode         string
	ShippingCost         float64
	ListingDate          time.Time
	EndDate              time.Time
}

func NewListing(
	marketplaceListingID string,
	marketplace string,
	title string,
	description string,
	url string,
	price float64,
	currency string,
	shippingCost float64,
	listingDate, endDate time.Time,
) Listing {
	return Listing{
		MarketplaceListingID: marketplaceListingID,
		Marketplace:          marketplace,
		Title:                title,
		Description:          description,
		URL:                  url,
		Price:                price,
		CurrencyCode:         currency,
		ShippingCost:         shippingCost,
		ListingDate:          listingDate,
		EndDate:              endDate,
	}
}
