package m_product

import (
	"time"

	"cloud.google.com/go/spanner"
)

// InsertMutation builds a spanner.Insert mutation for a product using a map of values.
// expected keys are the column names declared in fields.go
func InsertMutation(values map[string]interface{}) *spanner.Mutation {
	cols := make([]string, 0, len(values))
	vals := make([]interface{}, 0, len(values))
	for col, v := range values {
		cols = append(cols, col)
		vals = append(vals, v)
	}
	return spanner.Insert(TableName, cols, vals)
}

// UpdateMutation builds a spanner.Update mutation for a product.
// The values map should NOT include the product_id key (we accept productID separately).
// This helper will construct the columns slice with product_id first (primary key) then the update columns.
func UpdateMutation(productID string, values map[string]interface{}) *spanner.Mutation {
	// always include product_id as the first column
	cols := []string{ColProductID}
	vals := []interface{}{productID}

	for col, v := range values {
		cols = append(cols, col)
		vals = append(vals, v)
	}

	return spanner.Update(TableName, cols, vals)
}

// BuildInsertMap prepares the canonical fields for insertion.
// The caller should set created_at and updated_at (time.Time).
func BuildInsertMap(productID, name string, description *string, category string,
	baseNum, baseDen int64, discountPct *string,
	discountStart, discountEnd *time.Time, status string, createdAt, updatedAt time.Time) map[string]interface{} {

	m := map[string]interface{}{
		ColProductID:            productID,
		ColName:                 name,
		ColCategory:             category,
		ColBasePriceNumerator:   baseNum,
		ColBasePriceDenominator: baseDen,
		ColStatus:               status,
		ColCreatedAt:            createdAt,
		ColUpdatedAt:            updatedAt,
		ColArchivedAt:           nil,
	}

	if description != nil {
		m[ColDescription] = *description
	} else {
		m[ColDescription] = nil
	}

	if discountPct != nil {
		m[ColDiscountPercent] = *discountPct
	} else {
		m[ColDiscountPercent] = nil
	}

	if discountStart != nil {
		m[ColDiscountStartDate] = *discountStart
	} else {
		m[ColDiscountStartDate] = nil
	}

	if discountEnd != nil {
		m[ColDiscountEndDate] = *discountEnd
	} else {
		m[ColDiscountEndDate] = nil
	}

	return m
}

// BuildUpdateMap builds a map of columns -> values for UpdateMut,
// caller will pass the map to UpdateMutation(productID, map).
// updatedAt should be supplied by caller.
func BuildUpdateMap(updatedAt time.Time) map[string]interface{} {
	return map[string]interface{}{
		ColUpdatedAt: updatedAt,
	}
}
