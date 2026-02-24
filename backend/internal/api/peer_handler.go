package api

import (
	"errors"
	"net/http"
	"strings"

	"amnezia_go/backend/internal/models"
	"amnezia_go/backend/internal/service"
	"amnezia_go/backend/internal/store"
	"github.com/gin-gonic/gin"
)

const sessionCookieName = "connect.sid"

func (r *Router) listPeers(c *gin.Context) {
	peers, err := r.peerService.ListPeers(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "failed to list peers",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{"items": peers})
}

func (r *Router) createPeer(c *gin.Context) {
	var req createPeerRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}

	peer, err := r.peerService.CreatePeer(c.Request.Context(), service.CreatePeerInput{Name: req.Name})
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, peer)
}

func (r *Router) deletePeer(c *gin.Context) {
	id := strings.TrimSpace(c.Param("id"))
	if id == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "id is required"})
		return
	}

	err := r.peerService.DeletePeer(c.Request.Context(), id)
	if err != nil {
		if errors.Is(err, store.ErrPeerNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "peer not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to delete peer"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"deleted": true})
}

func (r *Router) peerConfig(c *gin.Context) {
	id := strings.TrimSpace(c.Param("id"))
	if id == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "id is required"})
		return
	}

	cfg, err := r.peerService.PeerConfig(c.Request.Context(), id)
	if err != nil {
		if errors.Is(err, store.ErrPeerNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "peer not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to build peer config"})
		return
	}

	c.Header("Content-Type", "text/plain; charset=utf-8")
	c.String(http.StatusOK, cfg)
}

func (r *Router) stats(c *gin.Context) {
	stats, err := r.peerService.Stats(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to collect stats"})
		return
	}
	c.JSON(http.StatusOK, stats)
}

func (r *Router) createSession(c *gin.Context) {
	var req createSessionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}

	sessionID, err := r.authService.LoginWithPassword(req.Password)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid credentials"})
		return
	}

	c.SetSameSite(http.SameSiteLaxMode)
	c.SetCookie(sessionCookieName, sessionID, 60*60*12, "/", "", false, true)
	c.JSON(http.StatusOK, gin.H{"success": true})
}

func (r *Router) getSession(c *gin.Context) {
	sessionID, err := c.Cookie(sessionCookieName)
	if err != nil || !r.authService.ValidateSession(sessionID) {
		c.JSON(http.StatusOK, gin.H{"authenticated": false})
		return
	}
	c.JSON(http.StatusOK, gin.H{"authenticated": true})
}

func (r *Router) deleteSession(c *gin.Context) {
	sessionID, err := c.Cookie(sessionCookieName)
	if err == nil {
		r.authService.Logout(sessionID)
	}
	c.SetCookie(sessionCookieName, "", -1, "/", "", false, true)
	c.JSON(http.StatusOK, gin.H{"success": true})
}

func (r *Router) sessionAuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		sessionID, err := c.Cookie(sessionCookieName)
		if err != nil || !r.authService.ValidateSession(sessionID) {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
			return
		}
		c.Next()
	}
}

func (r *Router) listPeersWG(c *gin.Context) {
	peers, err := r.peerService.ListPeers(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list clients"})
		return
	}
	c.JSON(http.StatusOK, mapPeersForWGEasy(peers))
}

func (r *Router) createPeerWG(c *gin.Context) {
	r.createPeer(c)
}

func (r *Router) deletePeerWG(c *gin.Context) {
	r.deletePeer(c)
}

func (r *Router) peerConfigWG(c *gin.Context) {
	r.peerConfig(c)
}

func mapPeersForWGEasy(peers []models.Peer) []gin.H {
	result := make([]gin.H, 0, len(peers))
	for _, peer := range peers {
		result = append(result, gin.H{
			"id":              peer.ID,
			"name":            peer.Name,
			"publicKey":       peer.PublicKey,
			"address":         peer.AllowedIP,
			"createdAt":       peer.CreatedAt,
			"latestHandshake": peer.LastHandshake,
			"transferRx":      peer.RXBytes,
			"transferTx":      peer.TXBytes,
		})
	}
	return result
}
