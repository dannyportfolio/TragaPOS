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

type TicketPayload struct {
	TicketID   string `json:"tid"`
	TenantID   string `json:"ten"`
	Category   string `json:"cat"`
	Pax        int    `json:"pax"`
	ValidUntil int64  `json:"exp"`
	IssuedAt   int64  `json:"iat"`
	OrderID    string `json:"oid,omitempty"`
}

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

type SignedTicketQR struct {
	Payload   TicketPayload `json:"payload"`
	Signature string        `json:"signature"`
}

type TicketRedemption struct {
	ID          string    `json:"id"`
	TicketID    string    `json:"ticket_id"`
	TenantID    string    `json:"tenant_id"`
	OutletID    string    `json:"outlet_id"`
	GateZone    string    `json:"gate_zone"`
	PaxRedeemed int       `json:"pax_redeemed"`
	RedeemedAt  time.Time `json:"redeemed_at"`
	DeviceID    string    `json:"device_id,omitempty"`
}

type TicketRepository interface {
	IsTicketRedeemed(ctx context.Context, tenantID, ticketID string) (bool, error)
	RecordRedemption(ctx context.Context, redemption *TicketRedemption) error
	GetRedemption(ctx context.Context, tenantID, ticketID string) (*TicketRedemption, error)
	SaveIssuedTicket(ctx context.Context, payload *TicketPayload, signature string) error
	GetIssuedTicket(ctx context.Context, ticketID string) (*SignedTicketQR, error)
}
