package models

import "time"

type Peer struct {
	ID            string     `json:"id"`
	Name          string     `json:"name"`
	PublicKey     string     `json:"public_key"`
	PrivateKey    string     `json:"private_key"`
	AllowedIP     string     `json:"allowed_ip"`
	CreatedAt     time.Time  `json:"created_at"`
	LastHandshake *time.Time `json:"last_handshake"`
	RXBytes       int64      `json:"rx_bytes"`
	TXBytes       int64      `json:"tx_bytes"`
}
