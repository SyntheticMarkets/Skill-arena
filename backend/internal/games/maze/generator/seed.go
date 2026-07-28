package generator

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"strings"
)

const seedBytes = 32

type SeedScope struct {
	Mode             string
	ScopeType        string
	ScopeID          string
	ParticipantID    string
	DifficultyID     string
	GeneratorVersion string
}

func (s SeedScope) Validate() error {
	if strings.TrimSpace(s.Mode) == "" || strings.TrimSpace(s.ScopeType) == "" ||
		strings.TrimSpace(s.ScopeID) == "" || strings.TrimSpace(s.DifficultyID) == "" ||
		strings.TrimSpace(s.GeneratorVersion) == "" {
		return errors.New("seed scope is incomplete")
	}
	if (s.Mode == "practice" || s.Mode == "house" || s.Mode == "training") && strings.TrimSpace(s.ParticipantID) == "" {
		return errors.New("participant id is required for private puzzle modes")
	}
	return nil
}

type SeedMaterial struct {
	value [seedBytes]byte
}

func (s SeedMaterial) Bytes() []byte {
	result := make([]byte, len(s.value))
	copy(result, s.value[:])
	return result
}

type SealedSeed struct {
	Reference  string
	KeyID      string
	Hash       string
	Ciphertext []byte
	Nonce      []byte
}

type SeedVault struct {
	derivationKey []byte
	encryptionKey [32]byte
	keyID         string
	random        io.Reader
}

func NewSeedVault(derivationSecret, encryptionSecret string) (*SeedVault, error) {
	return newSeedVault(derivationSecret, encryptionSecret, rand.Reader)
}

func newSeedVault(derivationSecret, encryptionSecret string, random io.Reader) (*SeedVault, error) {
	if len(derivationSecret) < 32 || len(encryptionSecret) < 32 {
		return nil, errors.New("puzzle derivation and encryption secrets must each contain at least 32 characters")
	}
	if hmac.Equal([]byte(derivationSecret), []byte(encryptionSecret)) {
		return nil, errors.New("puzzle derivation and encryption secrets must be distinct")
	}
	if random == nil {
		return nil, errors.New("cryptographic random source is required")
	}
	encryptionKey := sha256.Sum256([]byte(encryptionSecret))
	keyHash := sha256.Sum256(encryptionKey[:])
	return &SeedVault{
		derivationKey: []byte(derivationSecret),
		encryptionKey: encryptionKey,
		keyID:         "puzzle-key:" + fmt.Sprintf("%x", keyHash[:8]),
		random:        random,
	}, nil
}

func (v *SeedVault) Create(scope SeedScope, aad string) (SeedMaterial, SealedSeed, error) {
	if err := scope.Validate(); err != nil {
		return SeedMaterial{}, SealedSeed{}, err
	}
	var entropy [seedBytes]byte
	if _, err := io.ReadFull(v.random, entropy[:]); err != nil {
		return SeedMaterial{}, SealedSeed{}, fmt.Errorf("generate puzzle seed entropy: %w", err)
	}
	mac := hmac.New(sha256.New, v.derivationKey)
	writeField(mac, "skill-arena:puzzle-seed:v1")
	writeField(mac, scope.Mode)
	writeField(mac, scope.ScopeType)
	writeField(mac, scope.ScopeID)
	writeField(mac, scope.ParticipantID)
	writeField(mac, scope.DifficultyID)
	writeField(mac, scope.GeneratorVersion)
	_, _ = mac.Write(entropy[:])
	sum := mac.Sum(nil)
	var material SeedMaterial
	copy(material.value[:], sum)

	sealed, err := v.seal(material, aad)
	if err != nil {
		return SeedMaterial{}, SealedSeed{}, err
	}
	return material, sealed, nil
}

func (v *SeedVault) Open(sealed SealedSeed, aad string) (SeedMaterial, error) {
	if sealed.KeyID != v.keyID {
		return SeedMaterial{}, errors.New("puzzle seed key is unavailable")
	}
	block, err := aes.NewCipher(v.encryptionKey[:])
	if err != nil {
		return SeedMaterial{}, err
	}
	aead, err := cipher.NewGCM(block)
	if err != nil {
		return SeedMaterial{}, err
	}
	plain, err := aead.Open(nil, sealed.Nonce, sealed.Ciphertext, []byte(aad))
	if err != nil {
		return SeedMaterial{}, errors.New("puzzle seed integrity verification failed")
	}
	if len(plain) != seedBytes {
		return SeedMaterial{}, errors.New("puzzle seed length is invalid")
	}
	var material SeedMaterial
	copy(material.value[:], plain)
	if HashFields("skill-arena:puzzle-seed-hash:v1", base64.RawURLEncoding.EncodeToString(plain)) != sealed.Hash {
		return SeedMaterial{}, errors.New("puzzle seed hash verification failed")
	}
	return material, nil
}

func (v *SeedVault) seal(material SeedMaterial, aad string) (SealedSeed, error) {
	block, err := aes.NewCipher(v.encryptionKey[:])
	if err != nil {
		return SealedSeed{}, err
	}
	aead, err := cipher.NewGCM(block)
	if err != nil {
		return SealedSeed{}, err
	}
	nonce := make([]byte, aead.NonceSize())
	if _, err := io.ReadFull(v.random, nonce); err != nil {
		return SealedSeed{}, fmt.Errorf("generate puzzle seed nonce: %w", err)
	}
	ciphertext := aead.Seal(nil, nonce, material.value[:], []byte(aad))
	hash := HashFields("skill-arena:puzzle-seed-hash:v1", base64.RawURLEncoding.EncodeToString(material.value[:]))
	return SealedSeed{
		Reference:  HashFields("skill-arena:puzzle-seed-reference:v1", v.keyID, hash),
		KeyID:      v.keyID,
		Hash:       hash,
		Ciphertext: ciphertext,
		Nonce:      nonce,
	}, nil
}
