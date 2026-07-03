# Billing Stripe Simulator — Project Analysis

> **Status:** Pre-hackathon audit | **Date:** 2025-07-03
> **Target:** Win hackathon by fixing critical gaps in 4-6 hours

---

## ✅ Strengths (Already Strong)

| Area | Evidence | Score |
|------|----------|-------|
| **Architecture** | Clean layered: `app → service → repository → transport` with interfaces | 9/10 |
| **Time Travel (Killer Feature #1)** | `internal/clock.VirtualClock` with Redis persistence, used in `BillingService` | 9/10 |
| **Kafka Infrastructure** | Producer, Consumer, Processor with exponential backoff, DLQ | 8/10 |
| **Database** | Migrations, indexes, transactions, connection pooling | 8/10 |
| **Observability** | OpenTelemetry, structured logging (zap), health checks | 7/10 |
| **DevOps** | Multi-stage Dockerfile, docker-compose with healthchecks | 7/10 |
| **Code Quality** | golangci-lint strict config, Makefile with all targets | 8/10 |

---

## 🔴 Critical Gaps (Block Demo / Lose Points)

### 1. WebSocket Dashboard — **COMPLETELY MISSING** (Criteria 2, 5: -20 pts)
- `web/` directory is empty (only `.gitkeep`)
- `internal/transport/ws/hub.go:66` — `ServeWS()` upgrades connection but **never registers client**
- No HTTP route to serve dashboard HTML/JS
- **No real-time event stream to frontend** — killer feature "Live logging" doesn't exist

### 2. Time Travel API — **Implemented but Unreachable** (Criteria 2: -15 pts)
- `internal/service/time.TimeService.AdvanceTime()` exists and works with Virtual Clock
- **No HTTP handlers** — routes defined in `routes.go` (`/v1/time/advance`, `/v1/time/current`) return 404
- Cannot demo "one-click 30/90/365 days forward" from UI or curl

### 3. Webhook Delivery Engine — **Schema Only, No Logic** (Criteria 2: -15 pts)
- Table `webhook_logs` exists (migration `00000004`)
- **No delivery service**, no retry logic, no exponential backoff for webhooks
- Kafka DLQ exists but for **event processing**, not webhook delivery
- Cannot demo "guaranteed delivery with retry"

### 4. Docker Compose Broken (Criteria 6: Deploy fails)
```yaml
# Line 112: TYPO
networks:
  inernal:  # ← should be "internal"
```
- Network won't create, containers can't communicate

### 5. Dockerfile Won't Run Healthchecks (Criteria 6)
```dockerfile
FROM scratch
# No ca-certificates, no curl/wget, no tzdata
# Healthcheck `wget -qO- http://localhost:8080/healthz` will fail
```

### 6. README Empty (Criteria 7: -10 pts)
- Only `# Billing Stripe Simulator` — no quickstart, no demo, no architecture

---

## 🟡 Code Quality Issues (Criteria 1)

| File | Issue | Severity |
|------|-------|----------|
| `internal/transport/ws/hub.go:66` | `ServeWS` doesn't register client → WS dead | **Critical** |
| `internal/transport/http/handlers.go` | Missing `AdvanceTime`, `GetCurrentTime` handlers | **Critical** |
| `internal/service/notification.go` | `EventSender` interface in `service`, impl in `transport/kafka` — DIP violation | Medium |
| `internal/transport/http/handlers.go` | No idempotency keys, no `PriceID` validation | Medium |
| `pkg/kafka/processor.go:115` | Commits offset after DLQ publish error → **message loss risk** | High |
| `deployments/Dockerfile` | `FROM scratch` — no certs, no timezone, no healthcheck binary | High |

---

## 🎯 Hackathon Action Plan (Priority Order)

### Phase 1: Unblock Demo (P0 — ~2 hrs)
- [ ] Fix `hub.ServeWS()` — register client, add read/write pumps
- [ ] Add HTTP handlers: `AdvanceTime`, `GetCurrentTime` → wire to `TimeService`
- [ ] Fix docker-compose network typo (`inernal` → `internal`)
- [ ] Fix Dockerfile: use `gcr.io/distroless/static-debian12:nonroot` or add ca-certificates to scratch

### Phase 2: Webhook Engine (P1 — ~2-3 hrs)
- [ ] Create `internal/service/webhook.go` with:
  - HTTP client with exponential backoff (1s, 2s, 4s, 8s, 16s, 32s, max 5min)
  - Circuit breaker after 3 failures
  - Persist attempts to `webhook_logs` table
  - DLQ to Kafka `billing.dlq.webhooks` after max retries
- [ ] Trigger webhook delivery from `NotificationService` on events

### Phase 3: Real-time Dashboard (P1 — ~1.5 hrs)
- [ ] `web/index.html` + `app.js` (vanilla JS, no build step)
- [ ] Connect to `ws://localhost:8080/ws`, render event feed
- [ ] Time Travel controls: buttons +30d, +90d, +1y → POST `/v1/time/advance`
- [ ] Webhook log table: status badges, retry count, "Replay" button

### Phase 4: Polish & Docs (P2 — ~1 hr)
- [ ] Complete `README.md`: architecture diagram, quickstart, demo script, screenshots
- [ ] Add ADR docs: `docs/adr/001-virtual-clock.md`, `002-kafka-webhooks.md`, `003-websocket-dashboard.md`
- [ ] Run `make compose-up`, record demo GIF/asciinema

---

## 💡 Demo Script for Judges (30 seconds)

```bash
# 1. Start stack (one command)
make compose-up

# 2. Create customer + subscription
curl -X POST localhost:8080/v1/customers -d '{"email":"dev@test.com"}'
curl -X POST localhost:8080/v1/subscriptions -d '{"customer_id":"...", "price_id":"price_monthly"}'

# 3. OPEN DASHBOARD → see live events: customer.created, subscription.created, invoice.paid

# 4. TIME TRAVEL: click "+30 days" button
#    → Instantly see: subscription.renewed, invoice.created, invoice.paid, webhook.delivered

# 5. Kill your backend → webhook fails → dashboard shows "Retrying in 2s... 4s... 8s..."
#    → Restart backend → "Delivered" badge turns green
```

---

## 📁 Files to Create/Modify (Priority)

```
C:\Users\esandalov\Desktop\221\kodik\
├── internal/transport/ws/hub.go           ← FIX: register client
├── internal/transport/http/handlers.go    ← ADD: AdvanceTime, GetCurrentTime
├── internal/service/webhook.go            ← NEW: delivery engine
├── internal/service/notification.go       ← MODIFY: trigger webhook delivery
├── web/index.html                         ← NEW: dashboard HTML
├── web/app.js                             ← NEW: WS client + UI logic
├── web/style.css                          ← NEW: Stripe-like dark theme
├── docker-compose.yml                     ← FIX: network typo
├── deployments/Dockerfile                 ← FIX: distroless base
├── README.md                              ← WRITE: full documentation
└── docs/adr/*.md                          ← NEW: architecture decisions
```

---

## 🏆 Winning Criteria Mapping

| Criterion | Weight | Current | Target After Fix |
|-----------|--------|---------|------------------|
| 1. Code Quality & Architecture | 20 | 16/20 | 19/20 |
| 2. Product Implementation | 20 | 5/20 | 19/20 |
| 3. Idea & Commercial Value | 20 | 18/20 | 20/20 |
| 4. Presentation | 10 | 2/10 | 9/10 |
| 5. UX/Design | 10 | 0/10 | 8/10 |
| 6. Security & Correctness | 10 | 5/10 | 9/10 |
| 7. Documentation | 10 | 1/10 | 9/10 |
| **TOTAL** | **100** | **47/100** | **93/100** |

---

*Generated by Qwen Code analysis. Ready to implement fixes.*