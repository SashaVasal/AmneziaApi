package auth

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"sync"
	"time"

	"golang.org/x/crypto/bcrypt"
)

type Service struct {
	username     string
	passwordHash []byte
	ttl          time.Duration
	mu           sync.RWMutex
	sessions     map[string]time.Time
}

func NewService(username, plainPassword string, ttl time.Duration) (*Service, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(plainPassword), bcrypt.DefaultCost)
	if err != nil {
		return nil, fmt.Errorf("hash password: %w", err)
	}

	return &Service{
		username:     username,
		passwordHash: hash,
		ttl:          ttl,
		sessions:     make(map[string]time.Time),
	}, nil
}

func (s *Service) Login(username, password string) (string, error) {
	if username != s.username {
		return "", fmt.Errorf("invalid credentials")
	}
	return s.LoginWithPassword(password)
}

func (s *Service) LoginWithPassword(password string) (string, error) {
	if err := bcrypt.CompareHashAndPassword(s.passwordHash, []byte(password)); err != nil {
		return "", fmt.Errorf("invalid credentials")
	}

	sessionID, err := randomToken(32)
	if err != nil {
		return "", fmt.Errorf("generate session token: %w", err)
	}

	s.mu.Lock()
	s.sessions[sessionID] = time.Now().UTC().Add(s.ttl)
	s.mu.Unlock()

	return sessionID, nil
}

func (s *Service) ValidateSession(sessionID string) bool {
	now := time.Now().UTC()

	s.mu.Lock()
	defer s.mu.Unlock()

	expiresAt, ok := s.sessions[sessionID]
	if !ok {
		return false
	}
	if now.After(expiresAt) {
		delete(s.sessions, sessionID)
		return false
	}

	// Sliding expiration.
	s.sessions[sessionID] = now.Add(s.ttl)
	return true
}

func (s *Service) Logout(sessionID string) {
	s.mu.Lock()
	delete(s.sessions, sessionID)
	s.mu.Unlock()
}

func randomToken(bytesCount int) (string, error) {
	buf := make([]byte, bytesCount)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return hex.EncodeToString(buf), nil
}
