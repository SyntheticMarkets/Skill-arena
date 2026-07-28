package generator

import (
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"hash"
	"strconv"
	"strings"
)

func HashFields(domain string, fields ...string) string {
	target := sha256.New()
	writeField(target, domain)
	for _, field := range fields {
		writeField(target, field)
	}
	return "sha256:" + hex.EncodeToString(target.Sum(nil))
}

func ValidHash(value string) bool {
	if !strings.HasPrefix(value, "sha256:") || len(value) != len("sha256:")+sha256.Size*2 {
		return false
	}
	_, err := hex.DecodeString(strings.TrimPrefix(value, "sha256:"))
	return err == nil
}

func writeField(target hash.Hash, value string) {
	var size [8]byte
	binary.BigEndian.PutUint64(size[:], uint64(len(value)))
	_, _ = target.Write(size[:])
	_, _ = target.Write([]byte(value))
}

func intString(value int) string {
	return strconv.Itoa(value)
}
