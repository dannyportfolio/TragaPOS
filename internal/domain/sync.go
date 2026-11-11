package domain

import (
	"context"
	"encoding/json"
	"time"
)

type OutboxEventType string

const (
	EventOrderCreated     OutboxEventType = "ORDER_CREATED"
	EventPaymentReceived  OutboxEventType = "PAYMENT_RECEIVED"
	EventStockMoved       OutboxEventType = "STOCK_MOVED"
	EventTicketRedeemed   OutboxEventType = "TICKET_REDEEMED"
	EventDiscrepancyAlert OutboxEventType = "STOCK_DISCREPANCY_ALERT"
)

type OutboxStatus string

const (
	OutboxStatusPending   OutboxStatus = "PENDING"
	OutboxStatusSent      OutboxStatus = "SENT"
	OutboxStatusFailed    OutboxStatus = "FAILED"
)

// SyncOutbox represents the transactional outbox record stored on the Edge node.
type SyncOutbox struct {
	ID             string          `json:"id"` // UUIDv7
	TenantID       string          `json:"tenant_id"`
	OutletID       string          `json:"outlet_id"`
	EventType      OutboxEventType `json:"event_type"`
	AggregateType  string          `json:"aggregate_type"` // e.g. "order", "stock", "ticket"
	AggregateID    string          `json:"aggregate_id"`
	Payload        json.RawMessage `json:"payload"`
	Status         OutboxStatus    `json:"status"`
	RetryCount     int             `json:"retry_count"`
	LastAttemptAt  *time.Time      `json:"last_attempt_at,omitempty"`
	ErrorMessage   string          `json:"error_message,omitempty"`
	CreatedAt      time.Time       `json:"created_at"`
}

// SyncPushBatch is the HTTP payload sent from Edge to Cloud /v1/sync/push
type SyncPushBatch struct {
	OutletID       string       `json:"outlet_id"`
	TenantID       string       `json:"tenant_id"`
	IdempotencyKey string       `json:"idempotency_key"`
	Events         []SyncOutbox `json:"events"`
	SentAt         time.Time    `json:"sent_at"`
}

type SyncPushResponse struct {
	ProcessedCount int               `json:"processed_count"`
	AckEventIDs    []string          `json:"ack_event_ids"`
	Discrepancies  []DiscrepancyNotice `json:"discrepancies,omitempty"`
}

type DiscrepancyNotice struct {
	SKU            string  `json:"sku"`
	RequestedDelta float64 `json:"delta"`
	CurrentStock   float64 `json:"current_stock"`
	Message        string  `json:"message"`
}

// SyncPullRequest is the HTTP query sent from Edge to Cloud /v1/sync/pull
type SyncPullRequest struct {
	TenantID          string `json:"tenant_id"`
	OutletID          string `json:"outlet_id"`
	LastSyncedVersion int64  `json:"last_synced_version"`
	Limit             int    `json:"limit"`
}

// SyncPullResponse delivers updated catalog items to the Edge node.
type SyncPullResponse struct {
	Items        []Item `json:"items"`
	LatestVersion int64 `json:"latest_version"`
	HasMore      bool   `json:"has_more"`
}

type OutboxRepository interface {
	Enqueue(ctx context.Context, item *SyncOutbox) error
	FetchPendingBatch(ctx context.Context, limit int) ([]SyncOutbox, error)
	MarkSent(ctx context.Context, ids []string) error
	MarkFailed(ctx context.Context, id string, errMsg string) error
	PruneSent(ctx context.Context, olderThan time.Duration) (int64, error)
}
