package usecase

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"pos-engine/internal/config"
	"pos-engine/internal/domain"
	"pos-engine/internal/pkg/crypto"
	"pos-engine/internal/pkg/uid"
	"pos-engine/internal/repository"
)

type IssueTicketRequest struct {
	TenantID   string `json:"tenant_id"`
	Category   string `json:"category"`
	Pax        int    `json:"pax"`
	ValidHours int    `json:"valid_hours"`
	OrderID    string `json:"order_id,omitempty"`
}

type RedeemTicketRequest struct {
	QRString string `json:"qr_string"`
	TenantID string `json:"tenant_id"`
	OutletID string `json:"outlet_id"`
	GateZone string `json:"gate_zone"`
	DeviceID string `json:"device_id,omitempty"`
}

type TicketUsecase struct {
	cfg         *config.Config
	txMgr       *repository.TxManager
	ticketRepo  domain.TicketRepository
	outboxRepo  domain.OutboxRepository
	broadcaster EventBroadcaster
}

func NewTicketUsecase(
	cfg *config.Config,
	txMgr *repository.TxManager,
	ticketRepo domain.TicketRepository,
	outboxRepo domain.OutboxRepository,
	broadcaster EventBroadcaster,
) *TicketUsecase {
	return &TicketUsecase{
		cfg:         cfg,
		txMgr:       txMgr,
		ticketRepo:  ticketRepo,
		outboxRepo:  outboxRepo,
		broadcaster: broadcaster,
	}
}

// IssueTicket signs a ticket payload using the Cloud Ed25519 private key.
// Encodes as a compact QR string format: <base64url(payload_json)>.<signature>
func (uc *TicketUsecase) IssueTicket(ctx context.Context, req IssueTicketRequest) (*domain.SignedTicketQR, string, error) {
	if uc.cfg.Ed25519PrivKey == "" {
		return nil, "", errors.New("cannot issue tickets: ED25519_PRIVATE_KEY is not configured on this server")
	}

	validHours := req.ValidHours
	if validHours <= 0 {
		validHours = 24
	}
	pax := req.Pax
	if pax <= 0 {
		pax = 1
	}

	now := time.Now().UTC()
	validUntil := now.Add(time.Duration(validHours) * time.Hour).Unix()

	payload := domain.TicketPayload{
		TicketID:   uid.NewV7(),
		TenantID:   req.TenantID,
		Category:   req.Category,
		Pax:        pax,
		ValidUntil: validUntil,
		IssuedAt:   now.Unix(),
		OrderID:    req.OrderID,
	}

	// Sign the canonical deterministic compact string
	signData := payload.CompactSignData()
	sig, err := crypto.Sign(uc.cfg.Ed25519PrivKey, signData)
	if err != nil {
		return nil, "", fmt.Errorf("failed to sign ticket payload: %w", err)
	}

	signedTicket := &domain.SignedTicketQR{
		Payload:   payload,
		Signature: sig,
	}

	// Persist to issued_tickets table if repository is available
	if uc.ticketRepo != nil {
		_ = uc.ticketRepo.SaveIssuedTicket(ctx, &payload, sig)
	}

	// Construct compact QR string representation
	jsonPayload, _ := json.Marshal(payload)
	payloadB64 := base64.RawURLEncoding.EncodeToString(jsonPayload)
	qrString := fmt.Sprintf("%s.%s", payloadB64, sig)

	return signedTicket, qrString, nil
}

// RedeemTicketOffline executes 100% offline verification and double-scan prevention at the Edge turnstile.
func (uc *TicketUsecase) RedeemTicketOffline(ctx context.Context, req RedeemTicketRequest) (*domain.TicketRedemption, error) {
	if uc.cfg.Ed25519PubKey == "" {
		return nil, errors.New("cannot verify tickets: ED25519_PUBLIC_KEY is not configured on this node")
	}

	// 1. Parse QR string
	parts := strings.Split(req.QRString, ".")
	if len(parts) != 2 {
		return nil, domain.ErrInvalidTicketCode
	}

	payloadRaw, err := base64.RawURLEncoding.DecodeString(parts[0])
	if err != nil {
		// Fallback to standard base64
		payloadRaw, err = base64.StdEncoding.DecodeString(parts[0])
		if err != nil {
			return nil, fmt.Errorf("%w: base64 decode failed", domain.ErrInvalidTicketCode)
		}
	}

	var payload domain.TicketPayload
	if err := json.Unmarshal(payloadRaw, &payload); err != nil {
		return nil, fmt.Errorf("%w: json unmarshal failed", domain.ErrInvalidTicketCode)
	}

	signature := parts[1]

	// 2. Mathematical Cryptographic Signature Verification (Zero Network / 100% Offline)
	signData := payload.CompactSignData()
	if err := crypto.Verify(uc.cfg.Ed25519PubKey, signData, signature); err != nil {
		return nil, fmt.Errorf("%w: cryptographic signature check failed", domain.ErrInvalidTicketCode)
	}

	// 3. Mathematical Expiration Check
	nowUnix := time.Now().UTC().Unix()
	if nowUnix > payload.ValidUntil {
		return nil, domain.ErrTicketExpired
	}

	redemption := &domain.TicketRedemption{
		ID:          uid.NewV7(),
		TicketID:    payload.TicketID,
		TenantID:    req.TenantID,
		OutletID:    req.OutletID,
		GateZone:    req.GateZone,
		PaxRedeemed: payload.Pax,
		RedeemedAt:  time.Now().UTC(),
		DeviceID:    req.DeviceID,
	}

	// 4. Atomic Double-Scan Prevention & Outbox Queuing
	err = uc.txMgr.WithTransaction(ctx, func(txCtx context.Context) error {
		// A. Check if already redeemed
		alreadyRedeemed, err := uc.ticketRepo.IsTicketRedeemed(txCtx, req.TenantID, payload.TicketID)
		if err != nil {
			return err
		}
		if alreadyRedeemed {
			return domain.ErrTicketAlreadyUsed
		}

		// B. Record redemption
		if err := uc.ticketRepo.RecordRedemption(txCtx, redemption); err != nil {
			return fmt.Errorf("failed to record ticket redemption: %w", err)
		}

		// C. In EDGE Mode, queue TICKET_REDEEMED event in transactional outbox
		if uc.cfg.AppMode == config.ModeEdge && uc.outboxRepo != nil {
			redBytes, _ := json.Marshal(redemption)
			outboxItem := &domain.SyncOutbox{
				ID:            uid.NewV7(),
				TenantID:      req.TenantID,
				OutletID:      req.OutletID,
				EventType:     domain.EventTicketRedeemed,
				AggregateType: "ticket",
				AggregateID:   payload.TicketID,
				Payload:       redBytes,
				Status:        domain.OutboxStatusPending,
			}
			if err := uc.outboxRepo.Enqueue(txCtx, outboxItem); err != nil {
				return fmt.Errorf("failed to enqueue redemption outbox: %w", err)
			}
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	// 5. Post-Commit Real-Time Gate Turnstile Trigger
	if uc.broadcaster != nil {
		uc.broadcaster.BroadcastEvent("gate", "GATE_OPEN", map[string]any{
			"ticket_id":  redemption.TicketID,
			"gate_zone":  redemption.GateZone,
			"pax":        redemption.PaxRedeemed,
			"status":     "ACCESS_GRANTED",
			"timestamp":  redemption.RedeemedAt,
		})
	}

	return redemption, nil
}
