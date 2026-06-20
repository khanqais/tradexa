package workers

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"testing"

	"github.com/khanqais/tradexa/models"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func workerSetupDB(t testing.TB) *gorm.DB {
	t.Helper()

	b := make([]byte, 8)
	if _, err := rand.Read(b); err != nil {
		t.Fatalf("rand: %v", err)
	}
	dsn := fmt.Sprintf("file:workerdb_%s?mode=memory&cache=shared&_busy_timeout=5000", hex.EncodeToString(b))

	db, err := gorm.Open(
		sqlite.Open(dsn),
		&gorm.Config{Logger: logger.Default.LogMode(logger.Silent)},
	)
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	if err := db.AutoMigrate(
		&models.User{},
		&models.Listing{},
		&models.ListingImage{},
		&models.Bid{},
		&models.ProxyBid{},
		&models.Order{},
	); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	return db
}

func workerSeedSeller(db *gorm.DB, idx int) uint {
	u := models.User{
		Name:     fmt.Sprintf("Seller%d", idx),
		Email:    fmt.Sprintf("wseller%d@tradexa.io", idx),
		Password: "x",
		Role:     models.RoleSeller,
	}
	db.Create(&u)
	return u.ID
}

func workerSeedBuyer(db *gorm.DB, idx int) uint {
	u := models.User{
		Name:     fmt.Sprintf("Buyer%d", idx),
		Email:    fmt.Sprintf("wbuyer%d@tradexa.io", idx),
		Password: "x",
		Role:     models.RoleBuyer,
	}
	db.Create(&u)
	return u.ID
}

func workerSeedAuction(db *gorm.DB, sellerID uint, price float64) models.Listing {
	l := models.Listing{
		Title:       "Test Item",
		Description: "Worker test auction item",
		Price:       price,
		Type:        models.ListingTypeAuction,
		Category:    "test",
		SellerID:    sellerID,
		Status:      "active",
	}
	db.Create(&l)
	return l
}

func TestProcessAuctionClosure_Sold(t *testing.T) {
	db := workerSetupDB(t)
	sellerID := workerSeedSeller(db, 1)
	buyerID := workerSeedBuyer(db, 1)

	listing := workerSeedAuction(db, sellerID, 100)
	listing.ReservePrice = 150
	db.Create(&models.Bid{ListingID: listing.ID, BidderID: buyerID, Amount: 200})

	processAuctionClosure(db, listing)

	var updated models.Listing
	db.First(&updated, listing.ID)

	if !updated.IsSold {
		t.Error("listing should be is_sold=true after a successful auction close")
	}
	if updated.Status != "sold" {
		t.Errorf("expected status='sold', got %q", updated.Status)
	}

	var order models.Order
	if err := db.Where("listing_id = ?", listing.ID).First(&order).Error; err != nil {
		t.Fatalf("expected an order to be created: %v", err)
	}
	if order.Amount != 200 {
		t.Errorf("order amount: want 200, got %.2f", order.Amount)
	}
	if order.WinnerID != buyerID {
		t.Errorf("order winner: want %d, got %d", buyerID, order.WinnerID)
	}
	if order.SellerID != sellerID {
		t.Errorf("order seller: want %d, got %d", sellerID, order.SellerID)
	}
}

func TestProcessAuctionClosure_ReserveNotMet(t *testing.T) {
	db := workerSetupDB(t)
	sellerID := workerSeedSeller(db, 2)
	buyerID := workerSeedBuyer(db, 2)

	listing := workerSeedAuction(db, sellerID, 100)
	listing.ReservePrice = 500
	db.Create(&models.Bid{ListingID: listing.ID, BidderID: buyerID, Amount: 200})

	processAuctionClosure(db, listing)

	var updated models.Listing
	db.First(&updated, listing.ID)

	if updated.IsSold {
		t.Error("listing must NOT be sold when bid < reserve")
	}
	if updated.Status != "reserve_not_met" {
		t.Errorf("expected status='reserve_not_met', got %q", updated.Status)
	}

	var count int64
	db.Model(&models.Order{}).Where("listing_id = ?", listing.ID).Count(&count)
	if count != 0 {
		t.Errorf("expected 0 orders when reserve not met, got %d", count)
	}
}

func TestProcessAuctionClosure_NoBids(t *testing.T) {
	db := workerSetupDB(t)
	sellerID := workerSeedSeller(db, 3)
	listing := workerSeedAuction(db, sellerID, 100)
	listing.ReservePrice = 100

	processAuctionClosure(db, listing)

	var updated models.Listing
	db.First(&updated, listing.ID)
	if updated.IsSold {
		t.Error("no bids → listing must remain unsold")
	}
}

func TestProcessAuctionClosure_ZeroReserve(t *testing.T) {
	db := workerSetupDB(t)
	sellerID := workerSeedSeller(db, 4)
	buyerID := workerSeedBuyer(db, 4)

	listing := workerSeedAuction(db, sellerID, 50)
	listing.ReservePrice = 0
	db.Create(&models.Bid{ListingID: listing.ID, BidderID: buyerID, Amount: 51})

	processAuctionClosure(db, listing)

	var updated models.Listing
	db.First(&updated, listing.ID)
	if !updated.IsSold {
		t.Error("auction with reserve=0 and any bid should close as sold")
	}
}

func TestProcessAuctionClosure_Idempotent(t *testing.T) {
	db := workerSetupDB(t)
	sellerID := workerSeedSeller(db, 5)
	buyerID := workerSeedBuyer(db, 5)

	listing := workerSeedAuction(db, sellerID, 100)
	db.Create(&models.Bid{ListingID: listing.ID, BidderID: buyerID, Amount: 200})

	processAuctionClosure(db, listing)

	db.First(&listing, listing.ID)

	processAuctionClosure(db, listing)

	var count int64
	db.Model(&models.Order{}).Where("listing_id = ?", listing.ID).Count(&count)
	if count != 1 {
		t.Errorf("expected exactly 1 order after idempotent double-close, got %d", count)
	}
}

func BenchmarkAuctionClosure_Sold(b *testing.B) {
	db := workerSetupDB(b)
	sellerID := workerSeedSeller(db, 0)
	buyerID := workerSeedBuyer(db, 1)

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		b.StopTimer()
		listing := workerSeedAuction(db, sellerID, 100)
		db.Create(&models.Bid{ListingID: listing.ID, BidderID: buyerID, Amount: 200})
		b.StartTimer()

		processAuctionClosure(db, listing)
	}
}

func BenchmarkAuctionClosure_ReserveNotMet(b *testing.B) {
	db := workerSetupDB(b)
	sellerID := workerSeedSeller(db, 0)
	buyerID := workerSeedBuyer(db, 1)

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		b.StopTimer()
		listing := workerSeedAuction(db, sellerID, 100)
		listing.ReservePrice = 9999
		db.Create(&models.Bid{ListingID: listing.ID, BidderID: buyerID, Amount: 200})
		b.StartTimer()

		processAuctionClosure(db, listing)
	}
}
