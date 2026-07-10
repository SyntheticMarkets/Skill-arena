package models

import "time"

type Device struct {
	ID          string    `json:"id"`
	UserID      string    `json:"userId"`
	Fingerprint string    `json:"fingerprint"`
	DeviceName  string    `json:"deviceName,omitempty"`
	OS          string    `json:"os,omitempty"`
	Browser     string    `json:"browser,omitempty"`
	LastSeen    time.Time `json:"lastSeen"`
	CreatedAt   time.Time `json:"createdAt"`
}
