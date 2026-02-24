package wg

import "context"

type NoopService struct {
	serverPublicKey string
}

func NewNoopService(serverPublicKey string) *NoopService {
	return &NoopService{serverPublicKey: serverPublicKey}
}

func (s *NoopService) ListRuntimePeers(_ context.Context) (map[string]RuntimePeer, error) {
	return map[string]RuntimePeer{}, nil
}

func (s *NoopService) AddPeer(_ context.Context, _ string, _ string) error {
	return nil
}

func (s *NoopService) RemovePeer(_ context.Context, _ string) error {
	return nil
}

func (s *NoopService) ServerPublicKey(_ context.Context) (string, error) {
	return s.serverPublicKey, nil
}

func (s *NoopService) Close() error {
	return nil
}
