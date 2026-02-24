package api

import (
	"net/http"

	"amnezia_go/backend/internal/auth"
	"amnezia_go/backend/internal/service"
	"github.com/gin-gonic/gin"
)

type Router struct {
	authService *auth.Service
	peerService *service.PeerService
}

func NewRouter(authService *auth.Service, peerService *service.PeerService) *Router {
	return &Router{
		authService: authService,
		peerService: peerService,
	}
}

func (r *Router) Build() *gin.Engine {
	engine := gin.New()
	engine.Use(gin.Recovery())

	engine.GET("/healthz", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	engine.POST("/api/session", r.createSession)
	engine.GET("/api/session", r.getSession)
	engine.DELETE("/api/session", r.deleteSession)

	protected := engine.Group("/api")
	protected.Use(r.sessionAuthMiddleware())
	{
		// wg-easy compatible routes
		protected.GET("/wireguard/client", r.listPeersWG)
		protected.POST("/wireguard/client", r.createPeerWG)
		protected.DELETE("/wireguard/client/:id", r.deletePeerWG)
		protected.GET("/wireguard/client/:id/configuration", r.peerConfigWG)

		// project native routes
		protected.GET("/peers", r.listPeers)
		protected.POST("/peers", r.createPeer)
		protected.DELETE("/peers/:id", r.deletePeer)
		protected.GET("/peers/:id/config", r.peerConfig)
		protected.GET("/stats", r.stats)
	}

	return engine
}

type createSessionRequest struct {
	Password string `json:"password"`
}

type createPeerRequest struct {
	Name string `json:"name"`
}
