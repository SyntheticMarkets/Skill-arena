package replay

import (
	"context"
	"errors"
	"path"
	"strings"

	"skill-arena/internal/storage"
)

const replayContentType = "application/vnd.skill-arena.maze-replay+json"

type ObjectRepository struct {
	store  storage.ObjectStore
	prefix string
}

func NewObjectRepository(store storage.ObjectStore, prefix string) (*ObjectRepository, error) {
	if store == nil {
		return nil, errors.New("replay object store is required")
	}
	prefix = strings.Trim(strings.TrimSpace(prefix), "/")
	if prefix == "" {
		prefix = "replays/maze"
	}
	if strings.Contains(prefix, "..") {
		return nil, errors.New("replay object prefix is invalid")
	}
	return &ObjectRepository{store: store, prefix: prefix}, nil
}

func (r *ObjectRepository) Save(ctx context.Context, artifact Artifact) error {
	if err := validateReplayID(artifact.Genesis.ReplayID); err != nil {
		return err
	}
	data, err := Marshal(artifact)
	if err != nil {
		return err
	}
	return r.store.Put(ctx, r.key(artifact.Genesis.ReplayID), data, replayContentType)
}

func (r *ObjectRepository) Load(ctx context.Context, replayID string) (Artifact, error) {
	if err := validateReplayID(replayID); err != nil {
		return Artifact{}, err
	}
	data, err := r.store.Get(ctx, r.key(replayID))
	if err != nil {
		return Artifact{}, err
	}
	return Unmarshal(data)
}

func (r *ObjectRepository) key(replayID string) string {
	return path.Join(r.prefix, replayID+".json")
}

func (r *ObjectRepository) StorageKey(replayID string) (string, error) {
	if err := validateReplayID(replayID); err != nil {
		return "", err
	}
	return r.key(replayID), nil
}

func validateReplayID(replayID string) error {
	if strings.TrimSpace(replayID) == "" || len(replayID) > maxIdentitySize ||
		strings.ContainsAny(replayID, `/\`) || strings.Contains(replayID, "..") {
		return errors.New("replay identifier is invalid")
	}
	return nil
}
