package dto

// ProductDTO contains full product fields returned by read queries.
// Timestamps and optional fields use *string (RFC3339) to mirror how they
// typically come from Spanner/SQL. Use helpers to parse them into time.Time.
type ProductDTO struct {
	ProductID     string
	Name          string
	Description   *string
	Category      string
	BasePriceNum  int64
	BasePriceDen  int64
	DiscountPct   *string
	DiscountStart *string
	DiscountEnd   *string
	Status        string
	CreatedAt     *string
	UpdatedAt     *string
	ArchivedAt    *string

	// EffectivePrice computed by read query (decimal string).
	EffectivePrice string
}

// ProductSummaryDTO is a compact DTO for list queries.
type ProductSummaryDTO struct {
	ProductID string
	Name      string
	Category  string
	// EffectivePrice is a decimal string representation (best-effort) of the current effective price.
	EffectivePrice string

	// BasePriceNum/BasePriceDen are included so transport can return Money in API responses.
	BasePriceNum int64
	BasePriceDen int64
	Status       string
}
