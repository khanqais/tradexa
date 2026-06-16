package config

import (
	"log"

	"gorm.io/gorm"
)

// RunMigrations creates database indexes that GORM AutoMigrate may skip.
// All statements use IF NOT EXISTS so they are safe to run repeatedly.
func RunMigrations(db *gorm.DB) {
	indexes := []struct {
		name string
		sql  string
	}{
		// ── listings ──────────────────────────────────────────────
		{
			"idx_listings_is_sold_created",
			`CREATE INDEX IF NOT EXISTS idx_listings_is_sold_created
			 ON listings (is_sold, created_at DESC, deleted_at)`,
		},
		{
			"idx_listings_auction_watcher",
			`CREATE INDEX IF NOT EXISTS idx_listings_auction_watcher
			 ON listings (type, is_sold, status, auction_ends_at)
			 WHERE deleted_at IS NULL`,
		},
		{
			"idx_listings_category",
			`CREATE INDEX IF NOT EXISTS idx_listings_category
			 ON listings (category)`,
		},
		{
			"idx_listings_seller_id",
			`CREATE INDEX IF NOT EXISTS idx_listings_seller_id
			 ON listings (seller_id)`,
		},

		// ── listing_images ────────────────────────────────────────
		{
			"idx_listing_images_listing_id",
			`CREATE INDEX IF NOT EXISTS idx_listing_images_listing_id
			 ON listing_images (listing_id)`,
		},

		// ── bids ──────────────────────────────────────────────────
		{
			"idx_bids_listing_amount",
			`CREATE INDEX IF NOT EXISTS idx_bids_listing_amount
			 ON bids (listing_id, amount DESC)`,
		},
		{
			"idx_bids_bidder_id",
			`CREATE INDEX IF NOT EXISTS idx_bids_bidder_id
			 ON bids (bidder_id)`,
		},
	}

	for _, idx := range indexes {
		if err := db.Exec(idx.sql).Error; err != nil {
			log.Printf("[Migrations] ⚠ Failed to create index %s: %v", idx.name, err)
		} else {
			log.Printf("[Migrations] ✓ Index %s ready", idx.name)
		}
	}
}
