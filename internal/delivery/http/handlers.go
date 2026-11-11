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
	"pos-engine/internal/pkg/uid"
	"pos-engine/internal/repository"
	"pos-engine/internal/usecase"
)

type Handler struct {
	cfg         *config.Config
	txMgr       *repository.TxManager
	catalogRepo domain.CatalogRepository
	orderRepo   domain.OrderRepository
	ticketRepo  domain.TicketRepository
	checkoutUC  *usecase.CheckoutUsecase
	ticketUC    *usecase.TicketUsecase
	syncWorker  *usecase.SyncWorker
}

func NewHandler(
	cfg *config.Config,
	txMgr *repository.TxManager,
	catalogRepo domain.CatalogRepository,
	orderRepo domain.OrderRepository,
	ticketRepo domain.TicketRepository,
	checkoutUC *usecase.CheckoutUsecase,
	ticketUC *usecase.TicketUsecase,
	syncWorker *usecase.SyncWorker,
) *Handler {
	return &Handler{
		cfg:         cfg,
		txMgr:       txMgr,
		catalogRepo: catalogRepo,
		orderRepo:   orderRepo,
		ticketRepo:  ticketRepo,
		checkoutUC:  checkoutUC,
		ticketUC:    ticketUC,
		syncWorker:  syncWorker,
	}
}

// Healthz and Readyz probes for Kubernetes
func (h *Handler) Healthz(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status":    "HEALTHY",
		"mode":      h.cfg.AppMode,
		"timestamp": time.Now().UTC(),
	})
}

func (h *Handler) Readyz(c *gin.Context) {
	if err := h.txMgr.DB().PingContext(c.Request.Context()); err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"status": "UNREADY",
			"error":  err.Error(),
		})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"status": "READY",
		"mode":   h.cfg.AppMode,
	})
}

// --- CATALOG HANDLERS ---

func (h *Handler) UpsertItem(c *gin.Context) {
	var item domain.Item
	if err := c.ShouldBindJSON(&item); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if item.ID == "" {
		item.ID = uid.NewV7()
	}
	if item.TenantID == "" {
		item.TenantID = h.cfg.TenantID
	}
	if item.OutletID == "" {
		item.OutletID = h.cfg.OutletID
	}
	if item.Version <= 0 {
		item.Version = time.Now().UTC().UnixNano()
	}

	if err := h.catalogRepo.UpsertItem(c.Request.Context(), &item); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, item)
}

func (h *Handler) GetItemBySKU(c *gin.Context) {
	sku := c.Param("sku")
	tenantID := c.DefaultQuery("tenant_id", h.cfg.TenantID)

	item, err := h.catalogRepo.GetBySKU(c.Request.Context(), tenantID, sku)
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

func (h *Handler) GetItemByBarcode(c *gin.Context) {
	barcode := c.Param("barcode")
	tenantID := c.DefaultQuery("tenant_id", h.cfg.TenantID)

	item, err := h.catalogRepo.GetByBarcode(c.Request.Context(), tenantID, barcode)
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

	items, err := h.catalogRepo.ListItems(c.Request.Context(), tenantID, limit, offset)
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

	stock, err := h.catalogRepo.GetStock(c.Request.Context(), tenantID, outletID, sku)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, stock)
}

// --- CHECKOUT & ORDER HANDLERS ---

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
	order, err := h.orderRepo.GetOrderByID(c.Request.Context(), id)
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

// --- HOSPITALITY FOLIO HANDLERS ---

func (h *Handler) CreateFolio(c *gin.Context) {
	var folio domain.GuestFolio
	if err := c.ShouldBindJSON(&folio); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if folio.ID == "" {
		folio.ID = uid.NewV7()
	}
	if folio.TenantID == "" {
		folio.TenantID = h.cfg.TenantID
	}
	if folio.OutletID == "" {
		folio.OutletID = h.cfg.OutletID
	}
	folio.IsOpen = true
	if folio.CheckInAt.IsZero() {
		folio.CheckInAt = time.Now().UTC()
	}
	if folio.CheckOutAt.IsZero() {
		folio.CheckOutAt = time.Now().UTC().Add(24 * time.Hour)
	}

	if err := h.orderRepo.CreateFolio(c.Request.Context(), &folio); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, folio)
}

func (h *Handler) GetFolioByRoom(c *gin.Context) {
	room := c.Param("room")
	tenantID := c.DefaultQuery("tenant_id", h.cfg.TenantID)

	folio, err := h.orderRepo.GetFolioByRoom(c.Request.Context(), tenantID, room)
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

// --- CRYPTOGRAPHIC TICKETING HANDLERS ---

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

// --- SYNC ENGINE HANDLERS (Cloud Ingestor & Delta Puller) ---

func (h *Handler) PushSync(c *gin.Context) {
	// Runs on Cloud Hub
	var batch domain.SyncPushBatch
	if err := c.ShouldBindJSON(&batch); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	resp, err := usecase.IngestUpstreamBatch(
		c.Request.Context(),
		h.txMgr,
		h.catalogRepo,
		h.orderRepo,
		h.ticketRepo,
		batch,
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, resp)
}

func (h *Handler) PullSync(c *gin.Context) {
	// Runs on Cloud Hub
	tenantID := c.DefaultQuery("tenant_id", h.cfg.TenantID)
	sinceVer, _ := strconv.ParseInt(c.DefaultQuery("last_synced_version", "0"), 10, 64)
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "100"))

	items, err := h.catalogRepo.ListUpdatedSince(c.Request.Context(), tenantID, sinceVer, limit)
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
	// Runs on Edge Outlet
	if h.syncWorker == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "sync worker is not enabled on this node"})
		return
	}

	go h.syncWorker.PushOutboxBatch(context.Background())
	go h.syncWorker.PullCatalogDelta(context.Background())

	c.JSON(http.StatusAccepted, gin.H{"status": "SYNC_TRIGGERED"})
}
