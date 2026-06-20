package handlers_test

import (
	"bytes"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/khanqais/tradexa/config"
	"github.com/khanqais/tradexa/handlers"
	"github.com/khanqais/tradexa/models"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func setupTestDB(t testing.TB) *gorm.DB {
	t.Helper()

	b := make([]byte, 8)
	if _, err := rand.Read(b); err != nil {
		t.Fatalf("rand: %v", err)
	}
	dsn := fmt.Sprintf("file:testdb_%s?mode=memory&cache=shared&_busy_timeout=5000", hex.EncodeToString(b))

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

func replaceGlobalDB(db *gorm.DB) func() {
	old := config.DB
	config.DB = db
	return func() { config.DB = old }
}

func seedSeller(db *gorm.DB, idx int) uint {
	u := models.User{
		Name:     fmt.Sprintf("Seller%d", idx),
		Email:    fmt.Sprintf("seller%d@tradexa.io", idx),
		Password: "hashed",
		Role:     models.RoleSeller,
	}
	db.Create(&u)
	return u.ID
}

func seedBuyer(db *gorm.DB, idx int) uint {
	u := models.User{
		Name:     fmt.Sprintf("Buyer%d", idx),
		Email:    fmt.Sprintf("buyer%d@tradexa.io", idx),
		Password: "hashed",
		Role:     models.RoleBuyer,
	}
	db.Create(&u)
	return u.ID
}

func seedAuction(db *gorm.DB, sellerID uint, startPrice float64) models.Listing {
	ends := time.Now().Add(24 * time.Hour)
	l := models.Listing{
		Title:         "Vintage Watch",
		Description:   "An authentic vintage timepiece",
		Price:         startPrice,
		Type:          models.ListingTypeAuction,
		Category:      "collectibles",
		SellerID:      sellerID,
		AuctionEndsAt: &ends,
		Status:        "active",
	}
	db.Create(&l)
	return l
}

func bidRequest(userID uint, body interface{}) (*gin.Context, *httptest.ResponseRecorder) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	b, _ := json.Marshal(body)
	c.Request, _ = http.NewRequest(http.MethodPost, "/bid", bytes.NewBuffer(b))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Set("user_id", float64(userID))
	return c, w
}

func TestBidHandler_InvalidJSON(t *testing.T) {
	db := setupTestDB(t)
	defer replaceGlobalDB(db)()

	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request, _ = http.NewRequest(http.MethodPost, "/bid", bytes.NewBufferString("{invalid"))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Set("user_id", float64(1))

	handlers.BidHandler(c)
	if w.Code != http.StatusBadRequest {
		t.Errorf("malformed JSON: want 400, got %d", w.Code)
	}
}

func TestBidHandler_ListingNotFound(t *testing.T) {
	db := setupTestDB(t)
	defer replaceGlobalDB(db)()

	c, w := bidRequest(1, handlers.NewBid{Amount: 100, ListingID: 99999})
	handlers.BidHandler(c)
	if w.Code != http.StatusNotFound {
		t.Errorf("non-existent listing: want 404, got %d", w.Code)
	}
}

func TestBidHandler_CannotBidOnOwnListing(t *testing.T) {
	db := setupTestDB(t)
	defer replaceGlobalDB(db)()

	sellerID := seedSeller(db, 1)
	listing := seedAuction(db, sellerID, 100)

	c, w := bidRequest(sellerID, handlers.NewBid{Amount: 150, ListingID: listing.ID})
	handlers.BidHandler(c)
	if w.Code != http.StatusBadRequest {
		t.Errorf("self-bid: want 400, got %d", w.Code)
	}
}

func TestBidHandler_FirstBid_CreatesProxy(t *testing.T) {
	db := setupTestDB(t)
	defer replaceGlobalDB(db)()

	sellerID := seedSeller(db, 1)
	buyerID := seedBuyer(db, 1)
	listing := seedAuction(db, sellerID, 100)

	c, w := bidRequest(buyerID, handlers.NewBid{Amount: 120, ListingID: listing.ID})
	handlers.BidHandler(c)

	if w.Code != http.StatusOK {
		t.Fatalf("first bid: want 200, got %d — %s", w.Code, w.Body.String())
	}

	var proxy models.ProxyBid
	if err := db.Where("listing_id = ?", listing.ID).First(&proxy).Error; err != nil {
		t.Errorf("proxy bid not persisted: %v", err)
	}
	if proxy.MaxAmount != 120 {
		t.Errorf("proxy.MaxAmount: want 120, got %.2f", proxy.MaxAmount)
	}
	if proxy.BidderID != buyerID {
		t.Errorf("proxy.BidderID: want %d, got %d", buyerID, proxy.BidderID)
	}
}

func TestBidHandler_BelowMinimumIncrement_Rejected(t *testing.T) {
	db := setupTestDB(t)
	defer replaceGlobalDB(db)()

	sellerID := seedSeller(db, 1)
	buyer1 := seedBuyer(db, 1)
	buyer2 := seedBuyer(db, 2)
	listing := seedAuction(db, sellerID, 100)

	c1, _ := bidRequest(buyer1, handlers.NewBid{Amount: 120, ListingID: listing.ID})
	handlers.BidHandler(c1)

	c2, w2 := bidRequest(buyer2, handlers.NewBid{Amount: 102, ListingID: listing.ID})
	handlers.BidHandler(c2)
	if w2.Code != http.StatusBadRequest {
		t.Errorf("below-increment bid: want 400, got %d", w2.Code)
	}
}

func TestBidHandler_ProxyWar_NewBidderWins(t *testing.T) {
	db := setupTestDB(t)
	defer replaceGlobalDB(db)()

	sellerID := seedSeller(db, 1)
	buyer1 := seedBuyer(db, 1) // proxy at 200
	buyer2 := seedBuyer(db, 2) // challenger at 300 → wins
	listing := seedAuction(db, sellerID, 100)

	c1, _ := bidRequest(buyer1, handlers.NewBid{Amount: 200, ListingID: listing.ID})
	handlers.BidHandler(c1)

	c2, w2 := bidRequest(buyer2, handlers.NewBid{Amount: 300, ListingID: listing.ID})
	handlers.BidHandler(c2)

	if w2.Code != http.StatusOK {
		t.Fatalf("proxy war winner: want 200, got %d — %s", w2.Code, w2.Body.String())
	}

	var resp map[string]interface{}
	json.Unmarshal(w2.Body.Bytes(), &resp)
	gotWinner := uint(resp["winning_bidder"].(float64))
	if gotWinner != buyer2 {
		t.Errorf("winning_bidder: want buyer2 (%d), got %d", buyer2, gotWinner)
	}
}

func TestBidHandler_ProxyWar_ExistingProxyDefends(t *testing.T) {
	db := setupTestDB(t)
	defer replaceGlobalDB(db)()

	sellerID := seedSeller(db, 1)
	buyer1 := seedBuyer(db, 1) // proxy at 300
	buyer2 := seedBuyer(db, 2) // challenger at 200 → loses
	listing := seedAuction(db, sellerID, 100)

	c1, _ := bidRequest(buyer1, handlers.NewBid{Amount: 300, ListingID: listing.ID})
	handlers.BidHandler(c1)

	c2, w2 := bidRequest(buyer2, handlers.NewBid{Amount: 200, ListingID: listing.ID})
	handlers.BidHandler(c2)

	if w2.Code != http.StatusOK {
		t.Fatalf("defending proxy: want 200, got %d — %s", w2.Code, w2.Body.String())
	}

	var resp map[string]interface{}
	json.Unmarshal(w2.Body.Bytes(), &resp)
	gotWinner := uint(resp["winning_bidder"].(float64))
	if gotWinner != buyer1 {
		t.Errorf("winning_bidder: want buyer1 (%d) to defend, got %d", buyer1, gotWinner)
	}
}

func TestBidHandler_SelfProxyUpgrade(t *testing.T) {
	db := setupTestDB(t)
	defer replaceGlobalDB(db)()

	sellerID := seedSeller(db, 1)
	buyerID := seedBuyer(db, 1)
	listing := seedAuction(db, sellerID, 100)

	c1, _ := bidRequest(buyerID, handlers.NewBid{Amount: 120, ListingID: listing.ID})
	handlers.BidHandler(c1)

	c2, w2 := bidRequest(buyerID, handlers.NewBid{Amount: 200, ListingID: listing.ID})
	handlers.BidHandler(c2)
	if w2.Code != http.StatusOK {
		t.Errorf("self proxy upgrade: want 200, got %d — %s", w2.Code, w2.Body.String())
	}

	var proxy models.ProxyBid
	db.Where("listing_id = ?", listing.ID).First(&proxy)
	if proxy.MaxAmount != 200 {
		t.Errorf("upgraded proxy max: want 200, got %.2f", proxy.MaxAmount)
	}
}

func TestBidHandler_SelfProxyDowngrade_Rejected(t *testing.T) {
	db := setupTestDB(t)
	defer replaceGlobalDB(db)()

	sellerID := seedSeller(db, 1)
	buyerID := seedBuyer(db, 1)
	listing := seedAuction(db, sellerID, 100)

	c1, _ := bidRequest(buyerID, handlers.NewBid{Amount: 200, ListingID: listing.ID})
	handlers.BidHandler(c1)

	c2, w2 := bidRequest(buyerID, handlers.NewBid{Amount: 150, ListingID: listing.ID})
	handlers.BidHandler(c2)
	if w2.Code != http.StatusBadRequest {
		t.Errorf("proxy downgrade: want 400, got %d", w2.Code)
	}
}

func TestBidHandler_ExpiredAuction_Rejected(t *testing.T) {
	db := setupTestDB(t)
	defer replaceGlobalDB(db)()

	sellerID := seedSeller(db, 1)
	buyerID := seedBuyer(db, 1)

	past := time.Now().Add(-1 * time.Hour)
	listing := models.Listing{
		Title:         "Expired Item",
		Description:   "This auction already ended",
		Price:         50,
		Type:          models.ListingTypeAuction,
		SellerID:      sellerID,
		AuctionEndsAt: &past,
		Status:        "active",
	}
	db.Create(&listing)

	c, w := bidRequest(buyerID, handlers.NewBid{Amount: 100, ListingID: listing.ID})
	handlers.BidHandler(c)
	if w.Code != http.StatusBadRequest {
		t.Errorf("expired auction: want 400, got %d", w.Code)
	}
}

func TestBidHandler_ClosedAuction_Rejected(t *testing.T) {
	db := setupTestDB(t)
	defer replaceGlobalDB(db)()

	sellerID := seedSeller(db, 1)
	buyerID := seedBuyer(db, 1)
	listing := seedAuction(db, sellerID, 100)

	db.Model(&listing).Updates(map[string]interface{}{"is_sold": true, "status": "sold"})

	c, w := bidRequest(buyerID, handlers.NewBid{Amount: 200, ListingID: listing.ID})
	handlers.BidHandler(c)
	if w.Code != http.StatusBadRequest {
		t.Errorf("closed auction: want 400, got %d", w.Code)
	}
}

func TestConcurrentBids_RaceDetection(t *testing.T) {
	db := setupTestDB(t)
	defer replaceGlobalDB(db)()

	sellerID := seedSeller(db, 0)
	listing := seedAuction(db, sellerID, 50)

	const numBidders = 20
	var wg sync.WaitGroup
	var successCount int64

	for i := 1; i <= numBidders; i++ {
		wg.Add(1)
		buyerID := seedBuyer(db, i)
		amt := uint(100 + i*10)

		go func(bID uint, amount uint) {
			defer wg.Done()
			c, w := bidRequest(bID, handlers.NewBid{Amount: amount, ListingID: listing.ID})
			handlers.BidHandler(c)
			if w.Code == http.StatusOK {
				atomic.AddInt64(&successCount, 1)
			}
		}(buyerID, amt)
	}
	wg.Wait()

	t.Logf("Concurrent bids: %d/%d succeeded", successCount, numBidders)
	if successCount == 0 {
		t.Error("at least one bid must succeed under concurrent load")
	}

	var proxyCount int64
	db.Model(&models.ProxyBid{}).Where("listing_id = ?", listing.ID).Count(&proxyCount)
	if proxyCount != 1 {
		t.Errorf("expected 1 proxy after concurrent bids, got %d", proxyCount)
	}
}

func TestConcurrentBids_MultipleAuctions(t *testing.T) {
	db := setupTestDB(t)
	defer replaceGlobalDB(db)()

	sellerID := seedSeller(db, 0)
	const numAuctions = 8
	const biddersEach = 4

	type auctionResult struct {
		listingID  uint
		proxyCount int64
	}

	type auctionSetup struct {
		listing models.Listing
		bidders []uint
	}
	setups := make([]auctionSetup, numAuctions)
	for a := 0; a < numAuctions; a++ {
		setups[a].listing = seedAuction(db, sellerID, 50)
		setups[a].bidders = make([]uint, biddersEach)
		for b := 0; b < biddersEach; b++ {
			setups[a].bidders[b] = seedBuyer(db, a*100+b+8000)
		}
	}

	results := make([]auctionResult, numAuctions)
	var wg sync.WaitGroup

	for a := 0; a < numAuctions; a++ {
		wg.Add(1)
		s := setups[a]
		ai := a
		go func(setup auctionSetup, idx int) {
			defer wg.Done()
			for b, bID := range setup.bidders {
				amt := uint(100 + b*20)
				c, _ := bidRequest(bID, handlers.NewBid{Amount: amt, ListingID: setup.listing.ID})
				handlers.BidHandler(c)
			}
			var pc int64
			db.Model(&models.ProxyBid{}).Where("listing_id = ?", setup.listing.ID).Count(&pc)
			results[idx] = auctionResult{setup.listing.ID, pc}
		}(s, ai)
	}
	wg.Wait()

	for _, r := range results {
		if r.proxyCount > 1 {
			t.Errorf("listing %d: got %d proxy bids, expected at most 1", r.listingID, r.proxyCount)
		}
		t.Logf("Listing %d: proxies=%d", r.listingID, r.proxyCount)
	}
}

func BenchmarkBidHandler_FirstBid(b *testing.B) {
	db := setupTestDB(b)
	defer replaceGlobalDB(db)()
	sellerID := seedSeller(db, 0)

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		b.StopTimer()
		listing := seedAuction(db, sellerID, 50)
		buyerID := seedBuyer(db, i+1000)
		c, _ := bidRequest(buyerID, handlers.NewBid{Amount: 100, ListingID: listing.ID})
		b.StartTimer()

		handlers.BidHandler(c)
	}
}

func BenchmarkBidHandler_ProxyWar(b *testing.B) {
	db := setupTestDB(b)
	defer replaceGlobalDB(db)()
	sellerID := seedSeller(db, 0)

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		b.StopTimer()
		listing := seedAuction(db, sellerID, 50)
		b1 := seedBuyer(db, i*2+2000)
		b2 := seedBuyer(db, i*2+2001)
		c1, _ := bidRequest(b1, handlers.NewBid{Amount: 200, ListingID: listing.ID})
		handlers.BidHandler(c1)
		c2, _ := bidRequest(b2, handlers.NewBid{Amount: 300, ListingID: listing.ID})
		b.StartTimer()

		handlers.BidHandler(c2)
	}
}

func BenchmarkBidHandler_Parallel(b *testing.B) {
	db := setupTestDB(b)
	defer replaceGlobalDB(db)()
	sellerID := seedSeller(db, 0)

	const pool = 50
	listings := make([]models.Listing, pool)
	buyers := make([]uint, pool)
	for i := range listings {
		listings[i] = seedAuction(db, sellerID, 50)
		buyers[i] = seedBuyer(db, i+5000)
	}

	var idx int64
	b.ResetTimer()
	b.ReportAllocs()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			i := int(atomic.AddInt64(&idx, 1)) % pool
			c, _ := bidRequest(buyers[i], handlers.NewBid{
				Amount:    uint(100 + i),
				ListingID: listings[i].ID,
			})
			handlers.BidHandler(c)
		}
	})
}

func BenchmarkGetListingByID(b *testing.B) {
	gin.SetMode(gin.TestMode)
	db := setupTestDB(b)
	defer replaceGlobalDB(db)()

	sellerID := seedSeller(db, 0)
	buyerID := seedBuyer(db, 1)
	listing := seedAuction(db, sellerID, 100)
	db.Create(&models.Bid{ListingID: listing.ID, BidderID: buyerID, Amount: 150})
	idStr := fmt.Sprintf("%d", listing.ID)

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request, _ = http.NewRequest(http.MethodGet, "/listings/"+idStr, nil)
		c.Params = gin.Params{{Key: "id", Value: idStr}}
		handlers.GetListingByID(c)
	}
}

func BenchmarkGetListings_Paginated(b *testing.B) {
	gin.SetMode(gin.TestMode)
	db := setupTestDB(b)
	defer replaceGlobalDB(db)()

	sellerID := seedSeller(db, 0)
	for i := 0; i < 100; i++ {
		seedAuction(db, sellerID, float64(50+i))
	}

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request, _ = http.NewRequest(http.MethodGet, "/listings?type=auction&page=1&limit=10", nil)
		handlers.GetListings(c)
	}
}

func TestHighThroughput_BidStorm(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping bid-storm test in -short mode")
	}

	db := setupTestDB(t)
	defer replaceGlobalDB(db)()

	sellerID := seedSeller(db, 0)
	listing := seedAuction(db, sellerID, 50)

	const n = 100
	bidders := make([]uint, n)
	for i := range bidders {
		bidders[i] = seedBuyer(db, i+1)
	}

	start := time.Now()
	var successes, failures int

	for i, bID := range bidders {
		amt := uint(100 + i*10)
		c, w := bidRequest(bID, handlers.NewBid{Amount: amt, ListingID: listing.ID})
		handlers.BidHandler(c)
		if w.Code == http.StatusOK {
			successes++
		} else {
			failures++
		}
	}

	elapsed := time.Since(start)
	rps := float64(n) / elapsed.Seconds()

	t.Logf("Bid Storm: %d total | %d success | %d fail | %.0f bids/sec | %v elapsed",
		n, successes, failures, rps, elapsed)

	if rps < 50 {
		t.Errorf("throughput %.0f bids/sec is below minimum 50 bids/sec", rps)
	}
}
