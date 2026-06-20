# Tradexa

> A full-stack peer-to-peer marketplace supporting **fixed-price listings** and **live proxy-bidding auctions** — built for speed, concurrency-safety, and real-time interactivity.

🔗 **Live Demo:** [https://tradexa-1-zcv6.onrender.com](https://tradexa-1-zcv6.onrender.com)

---

## Table of Contents

- [Features](#features)
- [Architecture](#architecture)
- [Tech Stack](#tech-stack)
- [File Structure](#file-structure)
- [Performance](#performance)
- [Proxy Bidding — Usage Guide](#proxy-bidding--usage-guide)
- [API Reference](#api-reference)
- [Environment Variables](#environment-variables)
- [How to Run Locally](#how-to-run-locally)
- [Running Tests](#running-tests)

---

## Features

### Marketplace
- **Dual listing types** — fixed-price buy-now & timed auction listings
- **Category & search filtering** with paginated results
- **Multi-image uploads** via Cloudinary CDN
- **Seller dashboard** — create, edit, delete listings

### Auctions
- **Proxy (automatic) bidding** — set a max bid; the system bids on your behalf
- **Anti-snipe protection** — auctions auto-extend when late bids arrive
- **Reserve price** — sellers set a minimum acceptable sale price
- **Real-time bid feed** via Server-Sent Events (SSE)
- **Masked bidder names** — privacy-preserving display (e.g. `Q***s-3f2a`)
- **Automatic auction closure** — background worker detects expiry and creates orders

### Payments & Orders
- **Cashfree payment gateway** integration (test + production)
- **Escrow-style order lifecycle** — `pending_payment → paid_in_escrow → shipped → delivered`
- **Simulated delivery worker** — background task auto-advances delivery status
- **Webhook verification** for secure payment callbacks

### Real-time Communication
- **WebSocket notifications** — instant alerts for bids won/lost, auction results, reserve not met
- **In-app messaging** — buyer↔seller conversation threads over WebSocket
- **SSE bid stream** — zero-polling live price updates for any listing

### Auth & Security
- **Email/password auth** with OTP email verification
- **Google OAuth 2.0** sign-in
- **Forgot-password** flow with time-limited OTP
- **JWT-based sessions** with HTTP-only cookie strategy
- **Rate limiting** via Redis (per-IP; configurable per route)
- **Row-level locking** (`SELECT ... FOR UPDATE`) on the bid transaction to prevent race conditions

---

## Architecture

```
┌──────────────────────────────────────────────────────────────┐
│                        Browser (React)                        │
│   REST API (axios)  │  WebSocket (WS)  │  SSE (EventSource)  │
└────────────┬─────────────────┬──────────────────┬────────────┘
             │                 │                  │
             ▼                 ▼                  ▼
┌──────────────────────────────────────────────────────────────┐
│                 Go / Gin HTTP Server (:8080)                  │
│                                                               │
│  ┌────────────┐  ┌──────────────┐  ┌──────────────────────┐  │
│  │  Auth      │  │  Listings    │  │  BidHandler (atomic  │  │
│  │  Handler   │  │  Handler     │  │  tx + row lock)      │  │
│  └────────────┘  └──────────────┘  └──────────────────────┘  │
│  ┌────────────┐  ┌──────────────┐  ┌──────────────────────┐  │
│  │  Payment   │  │  Conversation│  │  SSE Stream Hub      │  │
│  │  Handler   │  │  Handler     │  │  (per-listing fan-   │  │
│  └────────────┘  └──────────────┘  │   out broadcaster)   │  │
│                                    └──────────────────────┘  │
│  ┌─────────────────────────────────────────────────────────┐  │
│  │              WebSocket Manager (gorilla/ws)              │  │
│  │   Per-user notification channels + conversation rooms   │  │
│  └─────────────────────────────────────────────────────────┘  │
└────────┬───────────────────────────────────────┬──────────────┘
         │                                       │
         ▼                                       ▼
┌─────────────────┐                   ┌──────────────────────┐
│   PostgreSQL    │                   │   Redis (Upstash)    │
│   (Supabase)    │                   │                      │
│                 │                   │  • Asynq task queue  │
│  • Users        │                   │  • Rate-limit        │
│  • Listings     │                   │    counters          │
│  • Bids         │                   └──────────────────────┘
│  • ProxyBids    │
│  • Orders       │                   ┌──────────────────────┐
│  • Conversations│                   │  Asynq Background    │
│  • Messages     │                   │  Workers             │
│  • OTPs         │                   │                      │
└─────────────────┘                   │  • AuctionWorker     │
                                      │    (close + notify)  │
         ┌────────────────────────┐   │  • DeliveryWorker    │
         │  Cloudinary CDN        │   │    (simulate ship)   │
         │  Image upload & serve  │   └──────────────────────┘
         └────────────────────────┘
```

### Key Design Decisions

| Concern | Solution |
|---|---|
| Concurrent bid conflicts | `SELECT … FOR UPDATE` row-level lock inside a DB transaction |
| Proxy war resolution | Single `ProxyBid` row per listing; challenger vs. defender logic in-process |
| Auction expiry | Asynq task scheduled at `auction_ends_at`; lazy re-evaluation handles extensions |
| Anti-snipe | Worker checks if `Now() < AuctionEndsAt` and re-queues if auction was extended |
| Live price feed | SSE fan-out hub; no polling, no WebSocket overhead for read-only viewers |
| Bidder privacy | SHA-256–derived mask (`Q***s-3f2a`) — same user always gets same tag |

---

## Tech Stack

### Backend
| Layer | Technology |
|---|---|
| Language | Go 1.25 |
| Web Framework | Gin |
| ORM | GORM + PostgreSQL driver |
| Database | PostgreSQL (Supabase) |
| Cache / Queue | Redis (Upstash) via `go-redis` |
| Background Jobs | Asynq (Redis-backed task queue) |
| WebSocket | Gorilla WebSocket |
| Auth | JWT (`golang-jwt/jwt v5`) + Google OAuth 2.0 |
| Image Storage | Cloudinary |
| Payments | Cashfree |
| Email | SMTP (Gmail) |
| Hot-reload (dev) | Air |

### Frontend
| Layer | Technology |
|---|---|
| Framework | React 19 + Vite 7 |
| Routing | React Router DOM v7 |
| HTTP Client | Axios |
| Animations | Framer Motion |
| Icons | Lucide React |
| Auth | `@react-oauth/google` |
| Styling | Vanilla CSS (no framework) |

---

## File Structure

```
Tradexa/
├── README.md
│
├── backend/
│   ├── main.go                    # App entry point; starts Gin + Asynq worker
│   ├── go.mod / go.sum
│   ├── .env                       # Environment variables (see below)
│   ├── .air.toml                  # Hot-reload config
│   │
│   ├── config/
│   │   ├── db.go                  # PostgreSQL connection (GORM)
│   │   ├── redis.go               # Redis client init
│   │   ├── asynq.go               # Asynq client + server init
│   │   ├── cloudinary.go          # Cloudinary SDK init
│   │   └── migrations.go          # Schema migrations
│   │
│   ├── models/
│   │   ├── user.go
│   │   ├── listing.go             # ListingType (fixed | auction)
│   │   ├── bid.go                 # Public bid ledger
│   │   ├── proxy_bid.go           # One reigning proxy per listing
│   │   ├── order.go               # Order lifecycle statuses
│   │   ├── conversation.go
│   │   ├── message.go
│   │   └── otp.go
│   │
│   ├── handlers/
│   │   ├── auth.go                # Register, Login, Google OAuth, OTP, forgot-password
│   │   ├── listing.go             # CRUD + BidHandler (proxy war logic)
│   │   ├── stream.go              # SSE endpoint (GET /stream/:id)
│   │   ├── hub.go                 # SSE broadcaster hub
│   │   ├── payment.go             # Cashfree order create, verify, webhook
│   │   ├── order.go               # Mark order shipped
│   │   ├── conversation.go        # Conversation CRUD
│   │   ├── conversation_ws.go     # WebSocket conversation handler
│   │   ├── notifications.go       # WebSocket notification handler
│   │   ├── websocket.go           # WS upgrade helpers
│   │   ├── chat.go                # Chat utilities
│   │   └── upload.go              # Cloudinary image upload
│   │
│   ├── middleware/
│   │   ├── auth.go                # JWT AuthRequired + OptionalAuth
│   │   ├── rate_limit.go          # Redis sliding-window rate limiter
│   │   └── validation.go          # ValidateParamInt helper
│   │
│   ├── routes/
│   │   └── route.go               # All route registrations
│   │
│   ├── workers/
│   │   ├── auction_worker.go      # Auction close + order create + email + WS notify
│   │   └── delivery_worker.go     # Simulated delivery progression
│   │
│   ├── tasks/
│   │   └── ...                    # Asynq task type definitions & constructors
│   │
│   ├── websocket/
│   │   └── ...                    # WebSocket manager (per-user channel registry)
│   │
│   └── utils/
│       └── ...                    # Email sender, misc helpers
│
└── Frontend/
    ├── index.html
    ├── vite.config.js
    ├── package.json
    └── src/
        ├── main.jsx               # React entry point
        ├── App.jsx                # Router + route definitions
        ├── index.css              # Global design tokens & utilities
        │
        ├── api/                   # Axios API client modules
        ├── context/               # React context providers (auth, etc.)
        ├── hooks/                 # Custom React hooks
        ├── utils/                 # Frontend utility functions
        │
        ├── components/
        │   ├── Navbar.jsx / .css
        │   ├── ListingCard.jsx / .css
        │   └── Spinner.jsx / .css
        │
        └── pages/
            ├── HomePage.jsx / .css
            ├── AuthPage.jsx / .css          # Login + Register
            ├── ListingDetailPage.jsx / .css  # Bid UI + SSE feed
            ├── AuctionsPage.jsx
            ├── BuyProductsPage.jsx
            ├── CreateListingPage.jsx / .css
            ├── MyListingsPage.jsx / .css
            ├── ConversationsPage.jsx / .css
            ├── ConversationDetailPage.jsx / .css
            └── PaymentStatusPage.jsx
```

---

## Performance

Tests run on **Windows / 12th Gen Intel Core i5-12450H** against an **in-memory SQLite** test DB.

### Unit & Integration Tests (handlers package)
```
go test ./handlers/... -run "TestBidHandler" -v

TestBidHandler_InvalidJSON              PASS  (0.00s)
TestBidHandler_ListingNotFound          PASS  (0.00s)
TestBidHandler_CannotBidOnOwnListing    PASS  (0.01s)
TestBidHandler_FirstBid_CreatesProxy    PASS  (0.00s)
TestBidHandler_BelowMinimumIncrement    PASS  (0.00s)
TestBidHandler_ProxyWar_NewBidderWins   PASS  (0.01s)
TestBidHandler_ProxyWar_ExistingProxy   PASS  (0.01s)
TestBidHandler_SelfProxyUpgrade         PASS  (0.00s)
TestBidHandler_SelfProxyDowngrade       PASS  (0.01s)
TestBidHandler_ExpiredAuction           PASS  (0.00s)
TestBidHandler_ClosedAuction            PASS  (0.01s)
```
All 11 bid-logic scenarios **pass**.

### Race-Detector Tests (`-race`)
```
go test ./handlers/... -race -v -timeout 60s

TestConcurrentBids_RaceDetection    PASS — 11/20 concurrent bids succeeded (correct serialization)
TestConcurrentBids_MultipleAuctions PASS — isolated per-listing locking verified
TestHighThroughput_BidStorm         PASS — 100/100 sequential bids accepted
Total elapsed: 6.933s
```
**Zero data races** detected.

### High-Throughput Benchmark

```
go test -v -bench=BenchmarkBidHandler_FirstBid -benchmem github.com/khanqais/tradexa/handlers

BenchmarkBidHandler_FirstBid-12    3056 iterations    438,037 ns/op    43 KB/op    657 allocs/op
```

| Metric | Value |
|---|---|
| Throughput (sequential) | **~1,678 bids/sec** |
| Throughput (race mode) | **~590 bids/sec** |
| Latency per bid | **438 µs** |
| Memory per bid | **43 KB / 657 allocs** |
| Concurrent safety | ✅ Row-level DB lock |
| Data races | ✅ Zero (verified with `-race`) |

---

## Proxy Bidding — Usage Guide

Tradexa implements **eBay-style automatic (proxy) bidding**. Here's how it works:

### How Proxy Bidding Works

When you submit a bid, you enter your **maximum bid** — the highest amount you're willing to pay. The system then bids on your behalf automatically, only as much as necessary to stay ahead of other bidders.

**Rules:**
- Minimum bid increment: **$5**
- One active proxy per listing at a time
- You cannot bid on your own listing

### Scenario Walkthroughs

#### 1. First Bid on a Listing
```
Listing price: $100
You submit max bid: $200

→ Your proxy is registered at $200
→ Current public price stays at $100 (no one to outbid)
→ You are the current winner at $100
```

#### 2. Challenger Enters (and Loses)
```
Current price: $100, Your proxy max: $200
Challenger bids max: $150

→ Proxy war resolves immediately:
   - Challenger's max ($150) < Your max ($200)
   - Your proxy counter-bids at $155 ($150 + $5 increment)
   - You remain the winner at $155
   - Challenger is notified they were outbid
```

#### 3. Challenger Wins (Higher Max)
```
Current price: $100, Existing proxy max: $150
You bid max: $300

→ Proxy war resolves:
   - Your max ($300) > Old proxy max ($150)
   - Old proxy fires its last shot at $150
   - You counter at $155 ($150 + $5 increment)
   - You become the new proxy holder at max $300
   - You win at $155 (not $300)
```

#### 4. Self Proxy Upgrade
```
You are current proxy at max: $200
You submit a new max bid: $350

→ Your proxy max is updated to $350
→ Public price stays at its current level
→ You remain the winner
```

#### 5. Self Proxy Downgrade (Rejected)
```
You are current proxy at max: $200
You submit max bid: $180

→ Rejected: "new max bid must be higher than your current max bid"
```

#### 6. Expired Auction
```
Auction ended at: 2026-06-01 10:00 UTC
You attempt to bid at: 2026-06-01 10:05 UTC

→ Rejected: "this auction has expired"
```

### Proxy Bidding API

**Endpoint:** `POST /api/bid`  
**Auth:** Required (JWT)

```json
{
  "listing_id": 42,
  "amount": 350
}
```

**Success Response:**
```json
{
  "Message": "bid placed successfully",
  "current_price": 155.00,
  "winning_bidder": 7
}
```

**Error Responses:**

| Status | Error |
|---|---|
| 400 | `"this item is not up for auction"` |
| 400 | `"you cannot bid on your own listing"` |
| 400 | `"this auction has expired"` |
| 400 | `"this auction is already closed"` |
| 400 | `"Your max bid must be at least X.XX"` |
| 400 | `"new max bid must be higher than your current max bid"` |
| 404 | `"listing not found"` |

---

## API Reference

All routes are prefixed with `/api`.

### Public Routes

| Method | Path | Description |
|---|---|---|
| GET | `/health` | Health check |
| POST | `/login` | Email/password login (rate limited: 10/15 min per IP) |
| POST | `/register` | Create new account |
| POST | `/auth/send-otp` | Send email verification OTP (5/hr per IP) |
| POST | `/auth/forgot-password/send-otp` | Forgot password OTP (3/hr per IP) |
| POST | `/auth/forgot-password/reset` | Reset password with OTP |
| POST | `/auth/google` | Google OAuth sign-in |
| GET | `/listings` | List all listings (filter: `search`, `category`, `type`, `sold`, `page`, `limit`) |
| GET | `/listings/:id` | Get listing details + highest bid + user's proxy max |
| GET | `/stream/:id` | SSE stream for live bid updates on a listing |
| POST | `/payment/webhook` | Cashfree payment webhook (public, verified by signature) |

### Protected Routes (JWT required)

| Method | Path | Description |
|---|---|---|
| POST | `/logout` | Invalidate session |
| GET | `/me` | Get current user profile |
| POST | `/me/avatar` | Upload profile avatar |
| POST | `/forget` | Change password |
| **POST** | **`/bid`** | **Place a proxy bid** |
| POST | `/listings` | Create a new listing |
| PUT | `/listings/:id` | Update your listing |
| DELETE | `/listings/:id` | Delete your listing |
| POST | `/upload` | Upload image to Cloudinary |
| POST | `/conversations` | Get or create a conversation with a user |
| GET | `/conversations` | List all conversations for current user |
| GET | `/conversations/:id/messages` | Get messages in a conversation |
| POST | `/payment/create-order` | Create Cashfree payment order |
| POST | `/payment/verify` | Verify Cashfree payment |
| POST | `/orders/:id/ship` | Mark an order as shipped |
| GET | `/ws/notifications` | WebSocket — personal notification stream |
| GET | `/ws/conversation/:id` | WebSocket — real-time conversation channel |

---

## Environment Variables

Create a `.env` file inside the `backend/` directory:

```env
# ── Database ──────────────────────────────────────────────────
DATABASE_URL=postgresql://user:password@host:port/dbname

# ── Auth ──────────────────────────────────────────────────────
JWT_SECRET=your_super_secret_jwt_key

# ── Google OAuth ──────────────────────────────────────────────
GOOGLE_CLIENT_ID=your_google_client_id.apps.googleusercontent.com
GOOGLE_CLIENT_SECRET=your_google_client_secret

# ── Redis (Upstash or self-hosted) ───────────────────────────
REDIS_URL=rediss://default:your_token@your-redis-host:6379

# ── Cloudinary ────────────────────────────────────────────────
CLOUDINARY_CLOUD_NAME=your_cloud_name
CLOUDINARY_API_KEY=your_api_key
CLOUDINARY_API_SECRET=your_api_secret

# ── SMTP Email ────────────────────────────────────────────────
SMTP_HOST=smtp.gmail.com
SMTP_PORT=587
SMTP_EMAIL=your_email@gmail.com
SMTP_PASSWORD=your_app_password

# ── Cashfree Payments ─────────────────────────────────────────
CASHFREE_APP_ID=your_cashfree_app_id
CASHFREE_SECRET_KEY=your_cashfree_secret_key

# ── Server ────────────────────────────────────────────────────
PORT=8080
FRONTEND_URL=http://localhost:3000
```

> **Note:** For Gmail SMTP, use an **App Password** (not your account password). Enable 2FA in your Google account first, then generate an App Password at [myaccount.google.com/apppasswords](https://myaccount.google.com/apppasswords).

---

## How to Run Locally

### Prerequisites

- **Go** 1.21+ — [https://go.dev/dl](https://go.dev/dl)
- **Node.js** 18+ — [https://nodejs.org](https://nodejs.org)
- **PostgreSQL** database (or a free [Supabase](https://supabase.com) project)
- **Redis** instance (or a free [Upstash](https://upstash.com) Redis)

### 1. Clone the Repository

```bash
git clone https://github.com/khanqais/tradexa.git
cd tradexa
```

### 2. Backend Setup

```bash
cd backend

# Copy environment file and fill in your values
cp .env.example .env   # (edit .env with your credentials)

# Download dependencies
go mod download

# Run the server (with hot-reload using Air)
air

# Or without Air:
go run main.go
```

The API will start at `http://localhost:8080`.  
GORM will **automatically migrate** all tables on first run.

### 3. Frontend Setup

```bash
cd Frontend

# Install dependencies
npm install

# Start development server
npm run dev
```

The frontend will start at `http://localhost:3000`.

### 4. Optional: Install Air (hot-reload)

```bash
go install github.com/air-verse/air@latest
```

Then run `air` inside the `backend/` directory.

---

## Running Tests

```bash
cd backend

# Run all bid handler tests
go test ./handlers/... -run "TestBidHandler" -v

# Run concurrency tests
go test ./handlers/... -run "TestConcurrent" -v

# Run with race detector 
go test ./handlers/... -race -v -timeout 60s

# Run high-throughput bid storm test
go test ./handlers/... -run "TestHighThroughput" -v

# Run benchmark
go test -bench=BenchmarkBidHandler_FirstBid -benchmem ./handlers/...
```


---

## License

MIT © 2026 Qais Khan
