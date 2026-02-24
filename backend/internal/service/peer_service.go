package service

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"strings"
	"time"

	appconfig "amnezia_go/backend/internal/config"
	"amnezia_go/backend/internal/ipam"
	"amnezia_go/backend/internal/models"
	"amnezia_go/backend/internal/store"
	"amnezia_go/backend/internal/wg"
	"golang.zx2c4.com/wireguard/wgctrl/wgtypes"
)

type PeerService struct {
	repo        store.PeerRepository
	wgService   wg.Service
	ipAllocator *ipam.Allocator
	template    appconfig.AWGTemplate
	serverHost  string
	serverPort  int
	dns         string
}

type CreatePeerInput struct {
	Name string `json:"name"`
}

func NewPeerService(
	repo store.PeerRepository,
	wgService wg.Service,
	ipAllocator *ipam.Allocator,
	template appconfig.AWGTemplate,
	serverHost string,
	serverPort int,
	dns string,
) *PeerService {
	return &PeerService{
		repo:        repo,
		wgService:   wgService,
		ipAllocator: ipAllocator,
		template:    template,
		serverHost:  serverHost,
		serverPort:  serverPort,
		dns:         dns,
	}
}

func (s *PeerService) ListPeers(ctx context.Context) ([]models.Peer, error) {
	savedPeers, err := s.repo.ListPeers(ctx)
	if err != nil {
		return nil, err
	}

	runtimePeers, err := s.wgService.ListRuntimePeers(ctx)
	if err != nil {
		// Keep API available even if runtime interface metrics are unavailable.
		return savedPeers, nil
	}

	for i := range savedPeers {
		runtimePeer, ok := runtimePeers[savedPeers[i].PublicKey]
		if !ok {
			continue
		}
		savedPeers[i].LastHandshake = runtimePeer.LastHandshake
		savedPeers[i].RXBytes = runtimePeer.RXBytes
		savedPeers[i].TXBytes = runtimePeer.TXBytes
	}

	return savedPeers, nil
}

func (s *PeerService) CreatePeer(ctx context.Context, input CreatePeerInput) (models.Peer, error) {
	name := strings.TrimSpace(input.Name)
	if name == "" {
		return models.Peer{}, fmt.Errorf("name is required")
	}

	allocated, err := s.repo.ListAllocatedIPs(ctx)
	if err != nil {
		return models.Peer{}, err
	}
	allowedIP, err := s.ipAllocator.NextFreeIP(allocated)
	if err != nil {
		return models.Peer{}, err
	}

	privateKey, err := wgtypes.GeneratePrivateKey()
	if err != nil {
		return models.Peer{}, fmt.Errorf("generate private key: %w", err)
	}
	publicKey := privateKey.PublicKey()

	peer := models.Peer{
		ID:         randomID(10),
		Name:       name,
		PublicKey:  publicKey.String(),
		PrivateKey: privateKey.String(),
		AllowedIP:  allowedIP,
		CreatedAt:  time.Now().UTC(),
	}

	if err := s.wgService.AddPeer(ctx, peer.PublicKey, peer.AllowedIP); err != nil {
		return models.Peer{}, err
	}

	if err := s.repo.CreatePeer(ctx, peer); err != nil {
		_ = s.wgService.RemovePeer(ctx, peer.PublicKey)
		return models.Peer{}, err
	}

	return peer, nil
}

func (s *PeerService) DeletePeer(ctx context.Context, id string) error {
	peer, err := s.repo.GetPeerByID(ctx, id)
	if err != nil {
		return err
	}

	if err := s.wgService.RemovePeer(ctx, peer.PublicKey); err != nil {
		return err
	}
	return s.repo.DeletePeer(ctx, id)
}

func (s *PeerService) PeerConfig(ctx context.Context, id string) (string, error) {
	peer, err := s.repo.GetPeerByID(ctx, id)
	if err != nil {
		return "", err
	}

	serverPub, err := s.wgService.ServerPublicKey(ctx)
	if err != nil {
		return "", err
	}

	config := fmt.Sprintf(`[Interface]
PrivateKey = %s
Address = %s
DNS = %s
Jc = %s
Jmin = %s
Jmax = %s
S1 = %s
S2 = %s
H1 = %s
H2 = %s
H3 = %s
H4 = %s

[Peer]
PublicKey = %s
AllowedIPs = 0.0.0.0/0, ::/0
Endpoint = %s:%d
PersistentKeepalive = 25
`, peer.PrivateKey, peer.AllowedIP, s.dns,
		s.template.Jc, s.template.Jmin, s.template.Jmax, s.template.S1, s.template.S2,
		s.template.H1, s.template.H2, s.template.H3, s.template.H4,
		serverPub, s.serverHost, s.serverPort)

	return config, nil
}

func (s *PeerService) Stats(ctx context.Context) (models.Stats, error) {
	peers, err := s.ListPeers(ctx)
	if err != nil {
		return models.Stats{}, err
	}

	stats := models.Stats{
		TotalPeers: len(peers),
	}
	cutoff := time.Now().UTC().Add(-3 * time.Minute)
	for _, peer := range peers {
		stats.RXBytes += peer.RXBytes
		stats.TXBytes += peer.TXBytes
		if peer.LastHandshake != nil && peer.LastHandshake.After(cutoff) {
			stats.ActivePeers++
		}
	}

	return stats, nil
}

func randomID(bytesCount int) string {
	buf := make([]byte, bytesCount)
	if _, err := rand.Read(buf); err != nil {
		panic(err)
	}
	return hex.EncodeToString(buf)
}
