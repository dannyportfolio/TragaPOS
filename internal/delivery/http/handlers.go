package http

import (
	"context"
	"errors"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"

	"pos-engine/internal/config"
	"pos-engine/internal/domain"
	"pos-engine/internal/usecase"
)

type HealthChecker interface {
	Ping(ctx context.Context) error
}

type Handler struct {
	cfg           *config.Config
	healthChecker HealthChecker
	catalogUC     *usecase.CatalogUsecase
	orderUC       *usecase.OrderUsecase
	checkoutUC    *usecase.CheckoutUsecase
	ticketUC      *usecase.TicketUsecase
	shiftUC       *usecase.ShiftUsecase
	syncUC        *usecase.SyncUsecase
	syncWorker    *usecase.SyncWorker
}

func NewHandler(
	cfg *config.Config,
	healthChecker HealthChecker,
	catalogUC *usecase.CatalogUsecase,
	orderUC *usecase.OrderUsecase,
	checkoutUC *usecase.CheckoutUsecase,
	ticketUC *usecase.TicketUsecase,
	shiftUC *usecase.ShiftUsecase,
	syncUC *usecase.SyncUsecase,
	syncWorker *usecase.SyncWorker,
) *Handler {
	return &Handler{
		cfg:           cfg,
		healthChecker: healthChecker,
		catalogUC:     catalogUC,
		orderUC:       orderUC,
		checkoutUC:    checkoutUC,
		ticketUC:      ticketUC,
		shiftUC:       shiftUC,
		syncUC:        syncUC,
		syncWorker:    syncWorker,
	}
}

func (h *Handler) Healthz(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status":    "HEALTHY",
		"mode":      h.cfg.AppMode,
		"timestamp": time.Now().UTC(),
	})
}

func (h *Handler) Readyz(c *gin.Context) {
	if h.healthChecker != nil {
		if err := h.healthChecker.Ping(c.Request.Context()); err != nil {
			c.JSON(http.StatusServiceUnavailable, gin.H{
				"status": "UNREADY",
				"error":  err.Error(),
			})
			return
		}
	}
	c.JSON(http.StatusOK, gin.H{
		"status": "READY",
		"mode":   h.cfg.AppMode,
	})
}

func (h *Handler) UpsertItem(c *gin.Context) {
	var item domain.Item
	if err := c.ShouldBindJSON(&item); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := h.catalogUC.UpsertItem(c.Request.Context(), &item); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, item)
}

func (h *Handler) CreateItem(c *gin.Context) {
	var item domain.Item
	if err := c.ShouldBindJSON(&item); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := h.catalogUC.CreateItem(c.Request.Context(), &item); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, item)
}

func (h *Handler) GetItemByID(c *gin.Context) {
	h.GetItem(c)
}

func (h *Handler) GetItemBySKU(c *gin.Context) {
	h.GetItem(c)
}

func (h *Handler) GetItem(c *gin.Context) {
	key := c.Param("id")
	if key == "" {
		key = c.Param("sku")
	}
	tenantID := c.DefaultQuery("tenant_id", h.cfg.TenantID)

	// First try lookup by ID
	item, err := h.catalogUC.GetItemByID(c.Request.Context(), key)
	if err == nil {
		c.JSON(http.StatusOK, item)
		return
	}

	// Next try lookup by SKU
	item, err = h.catalogUC.GetItemBySKU(c.Request.Context(), tenantID, key)
	if err != nil {
		if errors.Is(err, domain.ErrItemNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "item not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, item)
}

func (h *Handler) UpdateItem(c *gin.Context) {
	id := c.Param("id")
	var item domain.Item
	if err := c.ShouldBindJSON(&item); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := h.catalogUC.UpdateItem(c.Request.Context(), id, &item); err != nil {
		if errors.Is(err, domain.ErrItemNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "item not found"})
			return
		}
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"status": "UPDATED", "id": id})
}

func (h *Handler) DeleteItem(c *gin.Context) {
	id := c.Param("id")
	tenantID := c.DefaultQuery("tenant_id", h.cfg.TenantID)

	if err := h.catalogUC.DeleteItem(c.Request.Context(), tenantID, id); err != nil {
		if errors.Is(err, domain.ErrItemNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "item not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"status": "DELETED", "id": id})
}

type AdjustStockRequest struct {
	TenantID string  `json:"tenant_id"`
	OutletID string  `json:"outlet_id"`
	SKU      string  `json:"sku"`
	DeltaQty float64 `json:"delta_qty"`
	Reason   string  `json:"reason"`
	RefID    string  `json:"ref_id"`
}

func (h *Handler) AdjustStock(c *gin.Context) {
	var req AdjustStockRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if req.TenantID == "" {
		req.TenantID = h.cfg.TenantID
	}
	if req.OutletID == "" {
		req.OutletID = h.cfg.OutletID
	}

	bal, err := h.catalogUC.AdjustStock(c.Request.Context(), req.TenantID, req.OutletID, req.SKU, req.DeltaQty, req.Reason, req.RefID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, bal)
}


func (h *Handler) GetItemByBarcode(c *gin.Context) {
	barcode := c.Param("barcode")
	tenantID := c.DefaultQuery("tenant_id", h.cfg.TenantID)

	item, err := h.catalogUC.GetItemByBarcode(c.Request.Context(), tenantID, barcode)
	if err != nil {
		if errors.Is(err, domain.ErrItemNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "item not found with barcode"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, item)
}

func (h *Handler) ListItems(c *gin.Context) {
	tenantID := c.DefaultQuery("tenant_id", h.cfg.TenantID)
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "50"))
	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))

	items, err := h.catalogUC.ListItems(c.Request.Context(), tenantID, limit, offset)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"items":  items,
		"count":  len(items),
		"offset": offset,
		"limit":  limit,
	})
}

func (h *Handler) GetStock(c *gin.Context) {
	sku := c.Param("sku")
	tenantID := c.DefaultQuery("tenant_id", h.cfg.TenantID)
	outletID := c.DefaultQuery("outlet_id", h.cfg.OutletID)

	stock, err := h.catalogUC.GetStock(c.Request.Context(), tenantID, outletID, sku)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, stock)
}

func (h *Handler) Checkout(c *gin.Context) {
	var req usecase.CheckoutRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if req.TenantID == "" {
		req.TenantID = h.cfg.TenantID
	}
	if req.OutletID == "" {
		req.OutletID = h.cfg.OutletID
	}

	order, err := h.checkoutUC.Checkout(c.Request.Context(), req)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, order)
}

func (h *Handler) GetOrder(c *gin.Context) {
	id := c.Param("id")
	order, err := h.orderUC.GetOrderByID(c.Request.Context(), id)
	if err != nil {
		if errors.Is(err, domain.ErrOrderNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "order not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, order)
}

type UpdateOrderStatusRequest struct {
	Status domain.OrderStatus `json:"status" binding:"required"`
}

func (h *Handler) UpdateOrderStatus(c *gin.Context) {
	id := c.Param("id")
	var req UpdateOrderStatusRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	order, err := h.orderUC.UpdateOrderStatus(c.Request.Context(), id, req.Status)
	if err != nil {
		if errors.Is(err, domain.ErrOrderNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "order not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, order)
}

func (h *Handler) ListOrders(c *gin.Context) {
	tenantID := c.DefaultQuery("tenant_id", h.cfg.TenantID)
	outletID := c.DefaultQuery("outlet_id", "")
	status := c.DefaultQuery("status", "")
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "50"))
	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))

	orders, err := h.orderUC.ListOrders(c.Request.Context(), tenantID, outletID, status, limit, offset)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"orders": orders,
		"count":  len(orders),
		"offset": offset,
		"limit":  limit,
	})
}

type VoidOrderRequest struct {
	TenantID string `json:"tenant_id"`
	Reason   string `json:"reason"`
}

func (h *Handler) VoidOrder(c *gin.Context) {
	orderID := c.Param("id")
	var req VoidOrderRequest
	_ = c.ShouldBindJSON(&req)
	if req.TenantID == "" {
		req.TenantID = h.cfg.TenantID
	}
	if req.Reason == "" {
		req.Reason = "CUSTOMER_REQUEST"
	}

	if err := h.orderUC.VoidOrder(c.Request.Context(), req.TenantID, orderID, req.Reason); err != nil {
		if errors.Is(err, domain.ErrOrderNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "order not found"})
			return
		}
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status":   "VOIDED",
		"order_id": orderID,
		"reason":   req.Reason,
	})
}

func (h *Handler) ListKdsOrders(c *gin.Context) {
	tenantID := c.DefaultQuery("tenant_id", h.cfg.TenantID)
	outletID := c.DefaultQuery("outlet_id", h.cfg.OutletID)

	orders, err := h.orderUC.ListKdsOrders(c.Request.Context(), tenantID, outletID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"orders": orders,
		"count":  len(orders),
	})
}

func (h *Handler) CreateFolio(c *gin.Context) {
	var folio domain.GuestFolio
	if err := c.ShouldBindJSON(&folio); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := h.orderUC.CreateFolio(c.Request.Context(), &folio); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, folio)
}

func (h *Handler) GetFolioByRoom(c *gin.Context) {
	room := c.Param("room")
	tenantID := c.DefaultQuery("tenant_id", h.cfg.TenantID)

	folio, err := h.orderUC.GetFolioByRoom(c.Request.Context(), tenantID, room)
	if err != nil {
		if errors.Is(err, domain.ErrFolioNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "no active folio found for this room"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, folio)
}

func (h *Handler) ListFolios(c *gin.Context) {
	tenantID := c.DefaultQuery("tenant_id", h.cfg.TenantID)
	openOnly := c.DefaultQuery("open_only", "false") == "true"
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "50"))
	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))

	folios, err := h.orderUC.ListFolios(c.Request.Context(), tenantID, openOnly, limit, offset)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"folios": folios,
		"count":  len(folios),
	})
}

func (h *Handler) SettleFolio(c *gin.Context) {
	folioID := c.Param("id")
	tenantID := c.DefaultQuery("tenant_id", h.cfg.TenantID)

	if err := h.orderUC.CloseFolio(c.Request.Context(), tenantID, folioID); err != nil {
		if errors.Is(err, domain.ErrFolioNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "folio not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"status": "SETTLED", "folio_id": folioID})
}

func (h *Handler) IssueTicket(c *gin.Context) {
	var req usecase.IssueTicketRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if req.TenantID == "" {
		req.TenantID = h.cfg.TenantID
	}

	signedTicket, qrString, err := h.ticketUC.IssueTicket(c.Request.Context(), req)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"ticket":    signedTicket,
		"qr_string": qrString,
	})
}

func (h *Handler) RedeemTicket(c *gin.Context) {
	var req usecase.RedeemTicketRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if req.TenantID == "" {
		req.TenantID = h.cfg.TenantID
	}
	if req.OutletID == "" {
		req.OutletID = h.cfg.OutletID
	}

	redemption, err := h.ticketUC.RedeemTicketOffline(c.Request.Context(), req)
	if err != nil {
		status := http.StatusBadRequest
		if errors.Is(err, domain.ErrTicketAlreadyUsed) {
			status = http.StatusConflict
		} else if errors.Is(err, domain.ErrTicketExpired) {
			status = http.StatusGone
		}
		c.JSON(status, gin.H{
			"error":  err.Error(),
			"status": "ACCESS_DENIED",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"redemption": redemption,
		"status":     "ACCESS_GRANTED",
	})
}

func (h *Handler) PushSync(c *gin.Context) {
	if h.syncUC == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "sync usecase is not configured"})
		return
	}

	var batch domain.SyncPushBatch
	if err := c.ShouldBindJSON(&batch); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	resp, err := h.syncUC.IngestUpstreamBatch(c.Request.Context(), batch)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, resp)
}

func (h *Handler) PullSync(c *gin.Context) {
	tenantID := c.DefaultQuery("tenant_id", h.cfg.TenantID)
	sinceVer, _ := strconv.ParseInt(c.DefaultQuery("last_synced_version", "0"), 10, 64)
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "100"))

	items, err := h.catalogUC.ListUpdatedSince(c.Request.Context(), tenantID, sinceVer, limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	var latestVer int64 = sinceVer
	for _, it := range items {
		if it.Version > latestVer {
			latestVer = it.Version
		}
	}

	c.JSON(http.StatusOK, domain.SyncPullResponse{
		Items:         items,
		LatestVersion: latestVer,
		HasMore:       len(items) == limit,
	})
}

func (h *Handler) TriggerSync(c *gin.Context) {
	if h.syncWorker == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "sync worker is not enabled on this node"})
		return
	}

	go h.syncWorker.PushOutboxBatch(context.Background())
	go h.syncWorker.PullCatalogDelta(context.Background())

	c.JSON(http.StatusAccepted, gin.H{"status": "SYNC_TRIGGERED"})
}

func (h *Handler) OpenShift(c *gin.Context) {
	if h.shiftUC == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "shift management is not enabled"})
		return
	}

	var req usecase.OpenShiftRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if req.TenantID == "" {
		req.TenantID = h.cfg.TenantID
	}
	if req.OutletID == "" {
		req.OutletID = h.cfg.OutletID
	}

	shift, err := h.shiftUC.OpenShift(c.Request.Context(), req)
	if err != nil {
		if errors.Is(err, domain.ErrActiveShiftExists) {
			c.JSON(http.StatusConflict, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, shift)
}

func (h *Handler) GetCurrentShift(c *gin.Context) {
	if h.shiftUC == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "shift management is not enabled"})
		return
	}

	tenantID := c.DefaultQuery("tenant_id", h.cfg.TenantID)
	outletID := c.DefaultQuery("outlet_id", h.cfg.OutletID)
	cashierID := c.Query("cashier_id")
	if cashierID == "" {
		cashierID = "default_cashier"
	}

	summary, err := h.shiftUC.GetCurrentShift(c.Request.Context(), tenantID, outletID, cashierID)
	if err != nil {
		if errors.Is(err, domain.ErrNoActiveShift) {
			c.JSON(http.StatusNotFound, gin.H{"error": "no active shift found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, summary)
}

func (h *Handler) RecordCashMovement(c *gin.Context) {
	if h.shiftUC == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "shift management is not enabled"})
		return
	}

	var req usecase.CashMovementRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if req.TenantID == "" {
		req.TenantID = h.cfg.TenantID
	}
	if req.OutletID == "" {
		req.OutletID = h.cfg.OutletID
	}
	if req.CashierID == "" {
		req.CashierID = "default_cashier"
	}

	movement, err := h.shiftUC.RecordCashMovement(c.Request.Context(), req)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, movement)
}

func (h *Handler) CloseShift(c *gin.Context) {
	if h.shiftUC == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "shift management is not enabled"})
		return
	}

	var req usecase.CloseShiftRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if req.TenantID == "" {
		req.TenantID = h.cfg.TenantID
	}
	if req.OutletID == "" {
		req.OutletID = h.cfg.OutletID
	}
	if req.CashierID == "" {
		req.CashierID = "default_cashier"
	}

	summary, err := h.shiftUC.CloseShift(c.Request.Context(), req)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, summary)
}

func (h *Handler) HoldOrder(c *gin.Context) {
	var req usecase.CheckoutRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if req.TenantID == "" {
		req.TenantID = h.cfg.TenantID
	}
	if req.OutletID == "" {
		req.OutletID = h.cfg.OutletID
	}

	order, err := h.checkoutUC.HoldOrder(c.Request.Context(), req)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, order)
}

func (h *Handler) ListHeldOrders(c *gin.Context) {
	tenantID := c.DefaultQuery("tenant_id", h.cfg.TenantID)
	outletID := c.DefaultQuery("outlet_id", h.cfg.OutletID)

	orders, err := h.orderUC.ListHeldOrders(c.Request.Context(), tenantID, outletID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"orders": orders,
		"count":  len(orders),
	})
}

func (h *Handler) GetOrderReceipt(c *gin.Context) {
	id := c.Param("id")
	receiptText, err := h.orderUC.GenerateReceipt(c.Request.Context(), id)
	if err != nil {
		if errors.Is(err, domain.ErrOrderNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "order not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	format := c.DefaultQuery("format", "text")
	if format == "json" {
		c.JSON(http.StatusOK, gin.H{
			"order_id": id,
			"receipt":  receiptText,
		})
		return
	}

	c.Header("Content-Type", "text/plain; charset=utf-8")
	c.String(http.StatusOK, receiptText)
}

func (h *Handler) ListShifts(c *gin.Context) {
	if h.shiftUC == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "shift management is not enabled"})
		return
	}

	tenantID := c.DefaultQuery("tenant_id", h.cfg.TenantID)
	outletID := c.DefaultQuery("outlet_id", h.cfg.OutletID)
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "50"))
	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))

	shifts, err := h.shiftUC.ListShifts(c.Request.Context(), tenantID, outletID, limit, offset)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"shifts": shifts,
		"count":  len(shifts),
	})
}


