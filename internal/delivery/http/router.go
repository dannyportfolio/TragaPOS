package http

import (
	"github.com/gin-gonic/gin"

	"pos-engine/internal/delivery/ws"
)

func SetupRouter(handler *Handler, hub *ws.Hub) *gin.Engine {
	gin.SetMode(gin.ReleaseMode)
	router := gin.New()

	router.Use(gin.Recovery())
	router.Use(CORSMiddleware())

	idemMgr := NewIdempotencyManager()

	// Probes
	router.GET("/healthz", handler.Healthz)
	router.GET("/readyz", handler.Readyz)

	// Real-Time Event Hub (WebSocket for KDS, Turnstile, QRIS)
	router.GET("/ws", func(c *gin.Context) {
		hub.HandleWS(c.Writer, c.Request)
	})

	v1 := router.Group("/v1")
	{
		// Catalog endpoints
		v1.POST("/catalog/items", idemMgr.Middleware(), handler.UpsertItem)
		v1.GET("/catalog/items", handler.ListItems)
		v1.GET("/catalog/items/:sku", handler.GetItemBySKU)
		v1.GET("/catalog/barcode/:barcode", handler.GetItemByBarcode)
		v1.GET("/catalog/stock/:sku", handler.GetStock)

		// Checkout & Orders
		v1.POST("/checkout", idemMgr.Middleware(), handler.Checkout)
		v1.GET("/orders/:id", handler.GetOrder)

		// Hospitality Folios
		v1.POST("/folios", idemMgr.Middleware(), handler.CreateFolio)
		v1.GET("/folios/room/:room", handler.GetFolioByRoom)

		// Cryptographic Offline Ticketing
		v1.POST("/tickets/issue", idemMgr.Middleware(), handler.IssueTicket)
		v1.POST("/tickets/redeem", idemMgr.Middleware(), handler.RedeemTicket)

		// Sync Engine
		v1.POST("/sync/push", idemMgr.Middleware(), handler.PushSync)
		v1.GET("/sync/pull", handler.PullSync)
		v1.POST("/sync/trigger", handler.TriggerSync)
	}

	return router
}
