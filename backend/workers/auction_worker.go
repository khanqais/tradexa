package workers

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/hibiken/asynq"
	"github.com/khanqais/tradexa/config"
	"github.com/khanqais/tradexa/handlers"
	"github.com/khanqais/tradexa/models"
	"github.com/khanqais/tradexa/tasks"
	"github.com/khanqais/tradexa/utils"
	ws "github.com/khanqais/tradexa/websocket"
	"gorm.io/gorm"
)

// HandleAuctionCloseTask processes the "auction:close" job
func HandleAuctionCloseTask(ctx context.Context, t *asynq.Task) error {
	var payload tasks.AuctionClosePayload
	if err := json.Unmarshal(t.Payload(), &payload); err != nil {
		return err // Asynq will retry if we return an error
	}

	var listing models.Listing
	if err := config.DB.First(&listing, payload.ListingID).Error; err != nil {
		log.Printf("[AuctionWorker] Listing %d not found, dropping task.", payload.ListingID)
		return nil // Return nil so Asynq stops retrying a deleted listing
	}

	// Safety guard — skip if already processed
	if listing.IsSold || (listing.Status != "" && listing.Status != "active") {
		log.Printf("[AuctionWorker] Listing %d already closed, dropping task.", payload.ListingID)
		return nil
	}

	// -------------------------------------------------------------
	// ANTI-SNIPE MAGIC: The Lazy Re-evaluation Pattern
	// -------------------------------------------------------------
	if listing.AuctionEndsAt != nil && time.Now().Before(*listing.AuctionEndsAt) {
		log.Printf("[AuctionWorker] Auction %d was extended! Rescheduling for %v", listing.ID, listing.AuctionEndsAt)

		newTask, _ := tasks.NewAuctionCloseTask(listing.ID)

		// Enqueue the delayed task
		_, err := config.AsynqClient.Enqueue(newTask, asynq.ProcessAt(*listing.AuctionEndsAt))
		if err != nil {
			log.Printf("[AuctionWorker] Failed to requeue extended auction: %v", err)
			return err
		}

		// Return nil to mark THIS specific job as successfully complete
		return nil
	}

	// If we passed the check, the auction is genuinely over!
	processAuctionClosure(config.DB, listing)

	return nil
}

func processAuctionClosure(db *gorm.DB, listing models.Listing) {
	log.Printf("[AuctionWorker] Processing expired auction: listing_id=%d title=%q", listing.ID, listing.Title)

	var highestBid models.Bid
	hasBids := true
	if err := db.Where("listing_id = ?", listing.ID).Order("amount desc").First(&highestBid).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			hasBids = false
		} else {
			log.Printf("[AuctionWorker] Error fetching highest bid for listing %d: %v", listing.ID, err)
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
		log.Printf("[AuctionWorker] Error marking listing %d as sold: %v", listing.ID, err)
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
		log.Printf("[AuctionWorker] Error creating order for listing %d: %v", listing.ID, err)
		return
	}

	if err := tx.Commit().Error; err != nil {
		log.Printf("[AuctionWorker] Error committing auction sold for listing %d: %v", listing.ID, err)
		return
	}

	log.Printf("[AuctionWorker] Auction SOLD: listing_id=%d winner_id=%d amount=%.2f order_id=%d",
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
	handlers.StreamHub.Broadcast(listing.ID, ssePayload)

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
			log.Printf("[AuctionWorker] Failed to email winner %s: %v", winner.Email, err)
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
			log.Printf("[AuctionWorker] Failed to email seller %s: %v", seller.Email, err)
		}
	}()
}

func handleReserveNotMet(db *gorm.DB, listing models.Listing, hasBids bool) {
	if err := db.Model(&listing).Update("status", "reserve_not_met").Error; err != nil {
		log.Printf("[AuctionWorker] Error updating listing %d to reserve_not_met: %v", listing.ID, err)
		return
	}

	log.Printf("[AuctionWorker] Auction RESERVE NOT MET: listing_id=%d", listing.ID)

	var seller models.User
	db.First(&seller, listing.SellerID)

	ssePayload, _ := json.Marshal(map[string]interface{}{
		"type":       "auction_closed",
		"listing_id": listing.ID,
		"status":     "reserve_not_met",
	})
	handlers.StreamHub.Broadcast(listing.ID, ssePayload)

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
			log.Printf("[AuctionWorker] Failed to email seller %s: %v", seller.Email, err)
		}
	}()
}
