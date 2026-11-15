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
	txMgr       domain.TransactionManager
	ticketRepo  domain.TicketRepository
	outboxRepo  domain.OutboxRepository
	broadcaster EventBroadcaster
}

func NewTicketUsecase(
	cfg *config.Config,
	txMgr domain.TransactionManager,
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

	signData := payload.CompactSignData()
	sig, err := crypto.Sign(uc.cfg.Ed25519PrivKey, signData)
	if err != nil {
		return nil, "", fmt.Errorf("failed to sign ticket payload: %w", err)
	}

	signedTicket := &domain.SignedTicketQR{
		Payload:   payload,
		Signature: sig,
	}

	if uc.ticketRepo != nil {
		_ = uc.ticketRepo.SaveIssuedTicket(ctx, &payload, sig)
	}

	jsonPayload, err := json.Marshal(payload)
	if err != nil {
		return nil, "", fmt.Errorf("failed to marshal ticket payload: %w", err)
	}

	payloadB64 := base64.RawURLEncoding.EncodeToString(jsonPayload)
	qrString := fmt.Sprintf("%s.%s", payloadB64, sig)

	return signedTicket, qrString, nil
}

func (uc *TicketUsecase) RedeemTicketOffline(ctx context.Context, req RedeemTicketRequest) (*domain.TicketRedemption, error) {
	if uc.cfg.Ed25519PubKey == "" {
		return nil, errors.New("cannot verify tickets: ED25519_PUBLIC_KEY is not configured on this node")
	}

	payload, err := uc.parseAndVerifyTicket(req.QRString)
	if err != nil {
		return nil, err
	}

	if req.TenantID != "" && payload.TenantID != "" && payload.TenantID != req.TenantID {
		return nil, fmt.Errorf("%w: ticket belongs to different tenant", domain.ErrInvalidTicketCode)
	}

	if err := uc.verifyExpiration(payload); err != nil {
		return nil, err
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

	err = uc.txMgr.WithTransaction(ctx, func(txCtx context.Context) error {
		alreadyRedeemed, err := uc.ticketRepo.IsTicketRedeemed(txCtx, req.TenantID, payload.TicketID)
		if err != nil {
			return err
		}
		if alreadyRedeemed {
			return domain.ErrTicketAlreadyUsed
		}

		if err := uc.ticketRepo.RecordRedemption(txCtx, redemption); err != nil {
			return fmt.Errorf("failed to record ticket redemption: %w", err)
		}

		return uc.enqueueRedemptionOutbox(txCtx, req, payload.TicketID, redemption)
	})

	if err != nil {
		return nil, err
	}

	uc.dispatchGateAccess(redemption)
	return redemption, nil
}

func (uc *TicketUsecase) parseAndVerifyTicket(qrString string) (*domain.TicketPayload, error) {
	parts := strings.Split(qrString, ".")
	if len(parts) != 2 {
		return nil, domain.ErrInvalidTicketCode
	}

	payloadRaw, err := base64.RawURLEncoding.DecodeString(parts[0])
	if err != nil {
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
	signData := payload.CompactSignData()
	if err := crypto.Verify(uc.cfg.Ed25519PubKey, signData, signature); err != nil {
		return nil, fmt.Errorf("%w: cryptographic signature check failed", domain.ErrInvalidTicketCode)
	}

	return &payload, nil
}

func (uc *TicketUsecase) verifyExpiration(payload *domain.TicketPayload) error {
	nowUnix := time.Now().UTC().Unix()
	if nowUnix > payload.ValidUntil {
		return domain.ErrTicketExpired
	}
	return nil
}

func (uc *TicketUsecase) enqueueRedemptionOutbox(txCtx context.Context, req RedeemTicketRequest, ticketID string, redemption *domain.TicketRedemption) error {
	if uc.cfg.AppMode != config.ModeEdge || uc.outboxRepo == nil {
		return nil
	}

	redBytes, err := json.Marshal(redemption)
	if err != nil {
		return fmt.Errorf("failed to marshal redemption: %w", err)
	}

	outboxItem := &domain.SyncOutbox{
		ID:            uid.NewV7(),
		TenantID:      req.TenantID,
		OutletID:      req.OutletID,
		EventType:     domain.EventTicketRedeemed,
		AggregateType: "ticket",
		AggregateID:   ticketID,
		Payload:       redBytes,
		Status:        domain.OutboxStatusPending,
	}
	if err := uc.outboxRepo.Enqueue(txCtx, outboxItem); err != nil {
		return fmt.Errorf("failed to enqueue redemption outbox: %w", err)
	}

	return nil
}

func (uc *TicketUsecase) dispatchGateAccess(redemption *domain.TicketRedemption) {
	if uc.broadcaster == nil {
		return
	}

	uc.broadcaster.BroadcastEvent("gate", "GATE_OPEN", map[string]any{
		"ticket_id": redemption.TicketID,
		"gate_zone": redemption.GateZone,
		"pax":       redemption.PaxRedeemed,
		"status":    "ACCESS_GRANTED",
		"timestamp": redemption.RedeemedAt,
	})
}
