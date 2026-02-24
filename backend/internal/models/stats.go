package models

type Stats struct {
	ActivePeers int   `json:"active_peers"`
	TotalPeers  int   `json:"total_peers"`
	RXBytes     int64 `json:"rx_bytes"`
	TXBytes     int64 `json:"tx_bytes"`
}
