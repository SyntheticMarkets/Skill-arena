package id

import (
	"crypto/rand"
	"encoding/hex"
	"strings"
	"time"
)

func New(prefix string) string {
	var random [16]byte
	if _, err := rand.Read(random[:]); err != nil {
		panic(err)
	}
	encodedTime := strings.ToLower(time.Now().UTC().Format("20060102t150405000000000"))
	randomPart := hex.EncodeToString(random[:])
	if prefix == "" {
		return encodedTime + "_" + randomPart
	}
	return prefix + "_" + encodedTime + "_" + randomPart
}

func Session() string { return New("ses") }
func Match() string   { return New("mat") }
func Replay() string  { return New("rep") }
func Audit() string   { return New("aud") }
