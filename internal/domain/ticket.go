package domain

import (
	"context"
	"errors"
	"fmt"
	"time"
)

var (
	ErrTicketExpired     = errors.New("ticket has expired")
	ErrTicketAlreadyUsed = errors.New("ticket has already been redeemed")
	ErrInvalidTicketCode = errors.New("invalid ticket qr payload or format")
	ErrUnauthorizedGate  = errors.New("ticket category is not authorized for this gate zone")
)

// TicketPayload represents the unencoded data structure signed by the Cloud Ed25519 private key.
type TicketPayload struct {
	TicketID   string `json:"tid"` // UUIDv7
	TenantID   string `json:"ten"`
	Category   string `json:"cat"` // e.g., "REGULAR", "VIP", "FAMILY"
	Pax        int    `json:"pax"` // Number of persons
	ValidUntil int64  `json:"exp"` // Unix timestamp
	IssuedAt   int64  `json:"iat"` // Unix timestamp
	OrderID    string `json:"oid,omitempty"`
}

// CompactSignData returns the canonical deterministic byte slice to sign or verify.
func (p *TicketPayload) CompactSignData() []byte {
	return []byte(fmt.Sprintf("%s|%s|%s|%d|%d|%d",
		p.TicketID,
		p.TenantID,
		p.Category,
		p.Pax,
		p.ValidUntil,
		p.IssuedAt,
	))
}

// SignedTicketQR is the complete string encoded inside the visual 2D QR code.
// Format: base64(payload_json) + "." + signature_base64
type SignedTicketQR struct {
	Payload   TicketPayload `json:"payload"`
	Signature string        `json:"signature"`
}

type TicketRedemption struct {
	ID          string    `json:"id"` // UUIDv7
	TicketID    string    `json:"ticket_id"`
	TenantID    string    `json:"tenant_id"`
	OutletID    string    `json:"outlet_id"`
	GateZone    string    `json:"gate_zone"` // e.g., "MAIN_TURNSTILE_01"
	PaxRedeemed int       `json:"pax_redeemed"`
	RedeemedAt  time.Time `json:"redeemed_at"`
	DeviceID    string    `json:"device_id,omitempty"`
}

type TicketRepository interface {
	// Edge Local Verification & Redemption Record
	IsTicketRedeemed(ctx context.Context, tenantID, ticketID string) (bool, error)
	RecordRedemption(ctx context.Context, redemption *TicketRedemption) error
	GetRedemption(ctx context.Context, tenantID, ticketID string) (*TicketRedemption, error)

	// Cloud Master Ticket Registry (Optional audit on Cloud)
	SaveIssuedTicket(ctx context.Context, payload *TicketPayload, signature string) error
	GetIssuedTicket(ctx context.Context, ticketID string) (*SignedTicketQR, error)
}
