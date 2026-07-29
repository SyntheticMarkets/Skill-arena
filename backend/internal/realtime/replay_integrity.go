package realtime

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"hash"
	"strings"

	"skill-arena/internal/games/interfaces"
)

const replayIntegrityAlgorithm = "hmac-sha256-v1"

func (s *Service) SignReplayIntegrity(
	ctx context.Context,
	request interfaces.ReplayIntegrityRequest,
) (interfaces.ReplayIntegrityProof, error) {
	if err := ctx.Err(); err != nil {
		return interfaces.ReplayIntegrityProof{}, err
	}
	if err := validateReplayIntegrityRequest(request); err != nil {
		return interfaces.ReplayIntegrityProof{}, err
	}
	key, keyID, err := s.replayIntegrityKey()
	if err != nil {
		return interfaces.ReplayIntegrityProof{}, err
	}
	signature := replayIntegritySignature(key, request)
	return interfaces.ReplayIntegrityProof{
		Algorithm: replayIntegrityAlgorithm, KeyID: keyID,
		Signature: hex.EncodeToString(signature),
	}, nil
}

func (s *Service) VerifyReplayIntegrity(
	ctx context.Context,
	request interfaces.ReplayIntegrityRequest,
	proof interfaces.ReplayIntegrityProof,
) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if err := validateReplayIntegrityRequest(request); err != nil {
		return err
	}
	key, err := s.replayVerificationKey(proof.KeyID)
	if err != nil {
		return err
	}
	if proof.Algorithm != replayIntegrityAlgorithm {
		return errors.New("replay integrity proof version or key is unavailable")
	}
	provided, err := hex.DecodeString(proof.Signature)
	if err != nil || len(provided) != sha256.Size {
		return errors.New("replay integrity signature is malformed")
	}
	if !hmac.Equal(provided, replayIntegritySignature(key, request)) {
		return errors.New("replay integrity signature is invalid")
	}
	return nil
}

func (s *Service) replayVerificationKey(keyID string) ([]byte, error) {
	settings := s.store.RuntimeSettings()
	if settings == nil {
		return nil, errors.New("replay signing configuration is unavailable")
	}
	if keyID == settings.Security.ReplaySigningKeyID {
		if len(settings.Security.ReplaySigningKey) < 32 {
			return nil, errors.New("replay signing configuration is unavailable")
		}
		return []byte(settings.Security.ReplaySigningKey), nil
	}
	key := settings.Security.ReplayVerificationKeys[keyID]
	if len(key) < 32 {
		return nil, errors.New("replay integrity proof version or key is unavailable")
	}
	return []byte(key), nil
}

func (s *Service) replayIntegrityKey() ([]byte, string, error) {
	settings := s.store.RuntimeSettings()
	if settings == nil || len(settings.Security.ReplaySigningKey) < 32 ||
		strings.TrimSpace(settings.Security.ReplaySigningKeyID) == "" {
		return nil, "", errors.New("replay signing configuration is unavailable")
	}
	return []byte(settings.Security.ReplaySigningKey), settings.Security.ReplaySigningKeyID, nil
}

func validateReplayIntegrityRequest(request interfaces.ReplayIntegrityRequest) error {
	if strings.TrimSpace(request.MatchID) == "" || strings.TrimSpace(request.GameID) == "" ||
		!canonicalSHA256(request.ReplayHash) || !canonicalSHA256(request.EventRootHash) ||
		request.EventCount < 0 {
		return errors.New("replay integrity request is invalid")
	}
	return nil
}

func replayIntegritySignature(key []byte, request interfaces.ReplayIntegrityRequest) []byte {
	mac := hmac.New(sha256.New, key)
	writeReplayField(mac, []byte("skill-arena:replay-integrity:v1"))
	writeReplayField(mac, []byte(request.MatchID))
	writeReplayField(mac, []byte(request.GameID))
	writeReplayField(mac, []byte(request.ReplayHash))
	writeReplayField(mac, []byte(request.EventRootHash))
	var count [8]byte
	binary.BigEndian.PutUint64(count[:], uint64(request.EventCount))
	writeReplayField(mac, count[:])
	return mac.Sum(nil)
}

func writeReplayField(target hash.Hash, value []byte) {
	var size [8]byte
	binary.BigEndian.PutUint64(size[:], uint64(len(value)))
	_, _ = target.Write(size[:])
	_, _ = target.Write(value)
}

func canonicalSHA256(value string) bool {
	if len(value) != len("sha256:")+sha256.Size*2 || !strings.HasPrefix(value, "sha256:") {
		return false
	}
	_, err := hex.DecodeString(strings.TrimPrefix(value, "sha256:"))
	return err == nil
}
