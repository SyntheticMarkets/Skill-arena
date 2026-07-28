package generator

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/binary"
	"errors"
	"io"
)

type RandomStream struct {
	seed    [seedBytes]byte
	domain  string
	counter uint64
	block   []byte
	offset  int
}

func NewRandomStream(seed SeedMaterial, domain string) (*RandomStream, error) {
	if domain == "" {
		return nil, errors.New("random stream domain is required")
	}
	return &RandomStream{seed: seed.value, domain: domain}, nil
}

func (s *RandomStream) Read(target []byte) (int, error) {
	for written := 0; written < len(target); {
		if s.offset == len(s.block) {
			s.block = s.nextBlock()
			s.offset = 0
		}
		count := copy(target[written:], s.block[s.offset:])
		written += count
		s.offset += count
	}
	return len(target), nil
}

func (s *RandomStream) Uint64n(bound uint64) (uint64, error) {
	if bound == 0 {
		return 0, errors.New("random bound must be positive")
	}
	limit := ^uint64(0) - (^uint64(0) % bound)
	var data [8]byte
	for {
		if _, err := io.ReadFull(s, data[:]); err != nil {
			return 0, err
		}
		value := binary.BigEndian.Uint64(data[:])
		if value < limit {
			return value % bound, nil
		}
	}
}

func (s *RandomStream) nextBlock() []byte {
	mac := hmac.New(sha256.New, s.seed[:])
	writeField(mac, "skill-arena:puzzle-random-stream:v1")
	writeField(mac, s.domain)
	var counter [8]byte
	binary.BigEndian.PutUint64(counter[:], s.counter)
	_, _ = mac.Write(counter[:])
	s.counter++
	return mac.Sum(nil)
}
