package shared

import (
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"hash"
)

// HashFields hashes unambiguous length-prefixed fields under a required domain.
func HashFields(domain string, fields ...string) string {
	sum := sha256.New()
	writeField(sum, domain)
	for _, field := range fields {
		writeField(sum, field)
	}
	return "sha256:" + hex.EncodeToString(sum.Sum(nil))
}

func writeField(target hash.Hash, value string) {
	var size [8]byte
	binary.BigEndian.PutUint64(size[:], uint64(len(value)))
	_, _ = target.Write(size[:])
	_, _ = target.Write([]byte(value))
}
