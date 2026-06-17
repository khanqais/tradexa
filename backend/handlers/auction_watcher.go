package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strconv"
	"time"

	"github.com/khanqais/tradexa/config"
	"github.com/khanqais/tradexa/models"
	"github.com/khanqais/tradexa/utils"
	ws "github.com/khanqais/tradexa/websocket"
	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
)

func StartAuctionWatcher(db *gorm.DB) {
	log.Println("[AuctionWatcher] Started — watching for expired auctions")

	idleInterval := 60 * time.Second
	activeInterval := 10 * time.Second
	interval := idleInterval

	for {
		time.Sleep(interval)

		found := processExpiredAuctions(db)

		// Check if any auctions are still active to decide polling speed
		var activeCount int64
		db.Model(&models.Listing{}).
			Where("type = ? AND is_sold = ? AND (status = ? OR status = '') AND deleted_at IS NULL",
				"auction", false, "active").
			Count(&activeCount)

		if activeCount > 0 {
			interval = activeInterval
		} else {
			interval = idleInterval
		}

		if found > 0 {
			log.Printf("[AuctionWatcher] Processed %d expired auction(s)", found)
		}
	}
}

func processExpiredAuctions(db *gorm.DB) int {
	ctx := context.Background()
	nowStr := strconv.FormatInt(time.Now().Unix(), 10)

	expiredMembers, redisErr := config.RDB.ZRangeArgs(ctx, redis.ZRangeArgs{
		Key:     "auctions:pending",
		Start:   "-inf",
		Stop:    nowStr,
		ByScore: true,
	}).Result()

	if redisErr == nil && len(expiredMembers) > 0 {
		args := make([]interface{}, len(expiredMembers))
		for i, m := range expiredMembers {
			args[i] = m
		}
		config.RDB.ZRem(ctx, "auctions:pending", args...)

		processed := 0
		for _, memberStr := range expiredMembers {
			n, err := strconv.ParseUint(memberStr, 10, 64)
			if err != nil {
				log.Printf("[AuctionWatcher] Invalid member in Redis sorted set: %q — skipping", memberStr)
				continue
			}
			listingID := uint(n)

			var listing models.Listing
			if err := db.First(&listing, listingID).Error; err != nil {
				log.Printf("[AuctionWatcher] Listing %d not found in DB: %v", listingID, err)
				continue
			}
			if listing.IsSold || (listing.Status != "" && listing.Status != "active") {
				continue
			}
			processAuctionClosure(db, listing)
			processed++
		}
		return processed
	}

	var expiredListings []models.Listing
	err := db.Where(
		"type = ? AND is_sold = ? AND (status = ? OR status = '') AND auction_ends_at <= ?",
		"auction", false, "active", time.Now(),
	).Find(&expiredListings).Error
	if err != nil {
		log.Printf("[AuctionWatcher] Error querying expired auctions (DB fallback): %v", err)
		return 0
	}
	for _, listing := range expiredListings {
		processAuctionClosure(db, listing)
	}
	return len(expiredListings)
}

func processAuctionClosure(db *gorm.DB, listing models.Listing) {
	log.Printf("[AuctionWatcher] Processing expired auction: listing_id=%d title=%q", listing.ID, listing.Title)

	var highestBid models.Bid
	hasBids := true
	if err := db.Where("listing_id = ?", listing.ID).Order("amount desc").First(&highestBid).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			hasBids = false
		} else {
			log.Printf("[AuctionWatcher] Error fetching highest bid for listing %d: %v", listing.ID, err)
			return
		}
	}

	reserveMet := hasBids && (listing.ReservePrice <= 0 || highestBid.Amount >= listing.ReservePrice)

	if reserveMet {
		handleAuctionSold(db, listing, highestBid)
	} else {
		handleReserveNotMet(db, listing, hasBids)
	}
}

func handleAuctionSold(db *gorm.DB, listing models.Listing, highestBid models.Bid) {
	tx := db.Begin()

	if err := tx.Model(&listing).Updates(map[string]interface{}{
		"is_sold": true,
		"status":  "sold",
	}).Error; err != nil {
		tx.Rollback()
		log.Printf("[AuctionWatcher] Error marking listing %d as sold: %v", listing.ID, err)
		return
	}

	order := models.Order{
		ListingID: listing.ID,
		WinnerID:  highestBid.BidderID,
		SellerID:  listing.SellerID,
		Amount:    highestBid.Amount,
		Status:    models.OrderStatusPendingPayment,
	}
	if err := tx.Create(&order).Error; err != nil {
		tx.Rollback()
		log.Printf("[AuctionWatcher] Error creating order for listing %d: %v", listing.ID, err)
		return
	}

	if err := tx.Commit().Error; err != nil {
		log.Printf("[AuctionWatcher] Error committing auction sold for listing %d: %v", listing.ID, err)
		return
	}

	log.Printf("[AuctionWatcher] Auction SOLD: listing_id=%d winner_id=%d amount=%.2f order_id=%d",
		listing.ID, highestBid.BidderID, highestBid.Amount, order.ID)

	var winner, seller models.User
	db.First(&winner, highestBid.BidderID)
	db.First(&seller, listing.SellerID)

	ssePayload, _ := json.Marshal(map[string]interface{}{
		"type":       "auction_closed",
		"listing_id": listing.ID,
		"status":     "sold",
		"winner_id":  highestBid.BidderID,
		"amount":     highestBid.Amount,
		"order_id":   order.ID,
	})
	StreamHub.Broadcast(listing.ID, ssePayload)

	winnerNotif, _ := json.Marshal(map[string]interface{}{
		"type":       "auction_won",
		"listing_id": listing.ID,
		"title":      listing.Title,
		"amount":     highestBid.Amount,
		"order_id":   order.ID,
	})
	ws.Manager.NotifyUser(highestBid.BidderID, winnerNotif)

	sellerNotif, _ := json.Marshal(map[string]interface{}{
		"type":       "auction_sold",
		"listing_id": listing.ID,
		"title":      listing.Title,
		"amount":     highestBid.Amount,
		"buyer_name": winner.Name,
	})
	ws.Manager.NotifyUser(listing.SellerID, sellerNotif)

	go func() {
		subject := fmt.Sprintf("🏆 You won the auction for \"%s\"!", listing.Title)
		body := fmt.Sprintf(`
		<div style="font-family: Arial, sans-serif; padding: 20px; max-width: 600px; margin: 0 auto; border: 1px solid #e0e0e0; border-radius: 5px;">
			<h2 style="color: #333;">Congratulations, %s!</h2>
			<p>You won the auction for <strong>%s</strong>.</p>
			<div style="font-size: 24px; font-weight: bold; background-color: #f0fdf4; padding: 15px; border-radius: 4px; text-align: center; margin: 20px 0; color: #16a34a;">
				Winning Bid: $%.0f
			</div>
			<p>Please complete your payment within <strong>48 hours</strong> to secure your item.</p>
			<hr style="border: 0; border-top: 1px solid #eee; margin: 20px 0;">
			<p style="color: #999; font-size: 12px; text-align: center;">Tradexa &copy; 2026. All rights reserved.</p>
		</div>
		`, winner.Name, listing.Title, highestBid.Amount)

		if err := utils.SendEmail(winner.Email, subject, body); err != nil {
			log.Printf("[AuctionWatcher] Failed to email winner %s: %v", winner.Email, err)
		}
	}()

	go func() {
		subject := fmt.Sprintf("✅ Your item \"%s\" has been sold!", listing.Title)
		body := fmt.Sprintf(`
		<div style="font-family: Arial, sans-serif; padding: 20px; max-width: 600px; margin: 0 auto; border: 1px solid #e0e0e0; border-radius: 5px;">
			<h2 style="color: #333;">Great news, %s!</h2>
			<p>Your item <strong>%s</strong> has been sold at auction.</p>
			<div style="font-size: 24px; font-weight: bold; background-color: #f0fdf4; padding: 15px; border-radius: 4px; text-align: center; margin: 20px 0; color: #16a34a;">
				Final Price: $%.0f
			</div>
			<p>The buyer, <strong>%s</strong>, has been notified to complete payment within 48 hours.</p>
			<hr style="border: 0; border-top: 1px solid #eee; margin: 20px 0;">
			<p style="color: #999; font-size: 12px; text-align: center;">Tradexa &copy; 2026. All rights reserved.</p>
		</div>
		`, seller.Name, listing.Title, highestBid.Amount, winner.Name)

		if err := utils.SendEmail(seller.Email, subject, body); err != nil {
			log.Printf("[AuctionWatcher] Failed to email seller %s: %v", seller.Email, err)
		}
	}()
}

func handleReserveNotMet(db *gorm.DB, listing models.Listing, hasBids bool) {
	if err := db.Model(&listing).Update("status", "reserve_not_met").Error; err != nil {
		log.Printf("[AuctionWatcher] Error updating listing %d to reserve_not_met: %v", listing.ID, err)
		return
	}

	log.Printf("[AuctionWatcher] Auction RESERVE NOT MET: listing_id=%d", listing.ID)

	var seller models.User
	db.First(&seller, listing.SellerID)

	ssePayload, _ := json.Marshal(map[string]interface{}{
		"type":       "auction_closed",
		"listing_id": listing.ID,
		"status":     "reserve_not_met",
	})
	StreamHub.Broadcast(listing.ID, ssePayload)

	sellerNotif, _ := json.Marshal(map[string]interface{}{
		"type":       "auction_reserve_not_met",
		"listing_id": listing.ID,
		"title":      listing.Title,
	})
	ws.Manager.NotifyUser(listing.SellerID, sellerNotif)

	// WS notify all unique bidders
	if hasBids {
		var bidderIDs []uint
		db.Model(&models.Bid{}).Where("listing_id = ?", listing.ID).
			Distinct("bidder_id").Pluck("bidder_id", &bidderIDs)

		bidderNotif, _ := json.Marshal(map[string]interface{}{
			"type":       "auction_reserve_not_met",
			"listing_id": listing.ID,
			"title":      listing.Title,
		})
		for _, bidderID := range bidderIDs {
			ws.Manager.NotifyUser(bidderID, bidderNotif)
		}
	}

	go func() {
		subject := fmt.Sprintf("❌ Reserve price not met for \"%s\"", listing.Title)
		body := fmt.Sprintf(`
		<div style="font-family: Arial, sans-serif; padding: 20px; max-width: 600px; margin: 0 auto; border: 1px solid #e0e0e0; border-radius: 5px;">
			<h2 style="color: #333;">Auction Ended</h2>
			<p>Unfortunately, the reserve price was not met for your item <strong>%s</strong>.</p>
			<p>The item remains unsold. You can relist it or adjust the reserve price.</p>
			<hr style="border: 0; border-top: 1px solid #eee; margin: 20px 0;">
			<p style="color: #999; font-size: 12px; text-align: center;">Tradexa &copy; 2026. All rights reserved.</p>
		</div>
		`, listing.Title)

		if err := utils.SendEmail(seller.Email, subject, body); err != nil {
			log.Printf("[AuctionWatcher] Failed to email seller %s: %v", seller.Email, err)
		}
	}()
}
