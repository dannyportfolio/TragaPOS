package http

import (
	"github.com/gin-gonic/gin"

	"pos-engine/internal/config"
	"pos-engine/internal/delivery/ws"
)

func SetupRouter(handler *Handler, hub *ws.Hub, cfg *config.Config) *gin.Engine {
	gin.SetMode(gin.ReleaseMode)
	router := gin.New()

	router.Use(gin.Recovery())
	router.Use(StructuredLoggerMiddleware())
	router.Use(MetricsMiddleware())

	var allowedOrigins []string
	var apiToken string
	var edgeToken string
	if cfg != nil {
		allowedOrigins = cfg.AllowedOrigins
		apiToken = cfg.APIAuthToken
		edgeToken = cfg.EdgeAuthToken
	}

	router.Use(CORSMiddleware(allowedOrigins))

	idemMgr := NewIdempotencyManager()

	router.GET("/healthz", handler.Healthz)
	router.GET("/readyz", handler.Readyz)
	router.GET("/metrics", MetricsHandler)

	router.GET("/ws", AuthMiddleware(apiToken), func(c *gin.Context) {
		hub.HandleWS(c.Writer, c.Request)
	})

	v1 := router.Group("/v1")
	v1.Use(AuthMiddleware(apiToken))
	{
		// Catalog & Inventory CRUD
		v1.POST("/catalog/items", idemMgr.Middleware(), handler.CreateItem)
		v1.GET("/catalog/items", handler.ListItems)
		v1.GET("/catalog/items/:id", handler.GetItem)
		v1.PUT("/catalog/items/:id", idemMgr.Middleware(), handler.UpdateItem)
		v1.DELETE("/catalog/items/:id", handler.DeleteItem)
		v1.GET("/catalog/barcode/:barcode", handler.GetItemByBarcode)
		v1.GET("/catalog/stock/:sku", handler.GetStock)
		v1.POST("/catalog/stock/adjust", idemMgr.Middleware(), handler.AdjustStock)

		// Orders & Checkout
		v1.POST("/checkout", idemMgr.Middleware(), handler.Checkout)
		v1.GET("/orders", handler.ListOrders)
		v1.GET("/orders/:id", handler.GetOrder)
		v1.PUT("/orders/:id/status", handler.UpdateOrderStatus)
		v1.POST("/orders/:id/void", idemMgr.Middleware(), handler.VoidOrder)
		v1.DELETE("/orders/:id", idemMgr.Middleware(), handler.VoidOrder)
		v1.GET("/orders/:id/receipt", handler.GetOrderReceipt)
		v1.POST("/orders/hold", idemMgr.Middleware(), handler.HoldOrder)
		v1.GET("/orders/held", handler.ListHeldOrders)
		v1.GET("/kds/orders", handler.ListKdsOrders)

		// Shift Management & Cash Movements
		v1.POST("/shifts/open", idemMgr.Middleware(), handler.OpenShift)
		v1.GET("/shifts", handler.ListShifts)
		v1.GET("/shifts/current", handler.GetCurrentShift)
		v1.POST("/shifts/movements", idemMgr.Middleware(), handler.RecordCashMovement)
		v1.POST("/shifts/close", idemMgr.Middleware(), handler.CloseShift)

		// Hospitality Folios
		v1.POST("/folios", idemMgr.Middleware(), handler.CreateFolio)
		v1.GET("/folios", handler.ListFolios)
		v1.GET("/folios/room/:room", handler.GetFolioByRoom)
		v1.POST("/folios/:id/settle", idemMgr.Middleware(), handler.SettleFolio)

		// Offline-Verifiable Tickets
		v1.POST("/tickets/issue", idemMgr.Middleware(), handler.IssueTicket)
		v1.POST("/tickets/redeem", idemMgr.Middleware(), handler.RedeemTicket)

		// Data Synchronization
		syncGroup := v1.Group("/sync")
		syncGroup.Use(AuthMiddleware(edgeToken))
		{
			syncGroup.POST("/push", idemMgr.Middleware(), handler.PushSync)
			syncGroup.GET("/pull", handler.PullSync)
			syncGroup.POST("/trigger", handler.TriggerSync)
		}
	}

	return router
}
