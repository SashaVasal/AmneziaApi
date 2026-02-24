package wg

import (
	"context"
	"fmt"
	"net"
	"time"

	"golang.zx2c4.com/wireguard/wgctrl"
	"golang.zx2c4.com/wireguard/wgctrl/wgtypes"
)

type Service interface {
	ListRuntimePeers(ctx context.Context) (map[string]RuntimePeer, error)
	AddPeer(ctx context.Context, publicKey string, allowedIP string) error
	RemovePeer(ctx context.Context, publicKey string) error
	ServerPublicKey(ctx context.Context) (string, error)
	Close() error
}

type RuntimePeer struct {
	PublicKey     string
	AllowedIP     string
	LastHandshake *time.Time
	RXBytes       int64
	TXBytes       int64
}

type WGCtrlService struct {
	client        *wgctrl.Client
	interfaceName string
}

func NewWGCtrlService(interfaceName string) (*WGCtrlService, error) {
	client, err := wgctrl.New()
	if err != nil {
		return nil, fmt.Errorf("init wgctrl: %w", err)
	}

	return &WGCtrlService{
		client:        client,
		interfaceName: interfaceName,
	}, nil
}

func (s *WGCtrlService) Close() error {
	if s.client == nil {
		return nil
	}
	return s.client.Close()
}

func (s *WGCtrlService) ListRuntimePeers(_ context.Context) (map[string]RuntimePeer, error) {
	device, err := s.client.Device(s.interfaceName)
	if err != nil {
		return nil, fmt.Errorf("get device %q: %w", s.interfaceName, err)
	}

	peers := make(map[string]RuntimePeer, len(device.Peers))
	for _, peer := range device.Peers {
		mapped := mapPeer(peer)
		peers[mapped.PublicKey] = mapped
	}

	return peers, nil
}

func (s *WGCtrlService) AddPeer(_ context.Context, publicKey string, allowedIP string) error {
	key, err := wgtypes.ParseKey(publicKey)
	if err != nil {
		return fmt.Errorf("parse public key: %w", err)
	}

	_, network, err := net.ParseCIDR(allowedIP)
	if err != nil {
		return fmt.Errorf("parse allowed ip: %w", err)
	}

	cfg := wgtypes.Config{
		Peers: []wgtypes.PeerConfig{
			{
				PublicKey:         key,
				ReplaceAllowedIPs: true,
				AllowedIPs:        []net.IPNet{*network},
			},
		},
	}

	if err := s.client.ConfigureDevice(s.interfaceName, cfg); err != nil {
		return fmt.Errorf("configure device add peer: %w", err)
	}
	return nil
}

func (s *WGCtrlService) RemovePeer(_ context.Context, publicKey string) error {
	key, err := wgtypes.ParseKey(publicKey)
	if err != nil {
		return fmt.Errorf("parse public key: %w", err)
	}

	cfg := wgtypes.Config{
		Peers: []wgtypes.PeerConfig{
			{
				PublicKey: key,
				Remove:    true,
			},
		},
	}

	if err := s.client.ConfigureDevice(s.interfaceName, cfg); err != nil {
		return fmt.Errorf("configure device remove peer: %w", err)
	}
	return nil
}

func (s *WGCtrlService) ServerPublicKey(_ context.Context) (string, error) {
	device, err := s.client.Device(s.interfaceName)
	if err != nil {
		return "", fmt.Errorf("get device %q: %w", s.interfaceName, err)
	}
	return device.PublicKey.String(), nil
}

func mapPeer(peer wgtypes.Peer) RuntimePeer {
	var allowedIP string
	if len(peer.AllowedIPs) > 0 {
		allowedIP = peer.AllowedIPs[0].String()
	}

	var lastHandshake *time.Time
	if !peer.LastHandshakeTime.IsZero() {
		ts := peer.LastHandshakeTime.UTC()
		lastHandshake = &ts
	}

	return RuntimePeer{
		PublicKey:     peer.PublicKey.String(),
		AllowedIP:     allowedIP,
		LastHandshake: lastHandshake,
		RXBytes:       int64(peer.ReceiveBytes),
		TXBytes:       int64(peer.TransmitBytes),
	}
}
