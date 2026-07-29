package replay

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
)

func Marshal(artifact Artifact) ([]byte, error) {
	if err := validateArtifactEnvelope(artifact); err != nil {
		return nil, err
	}
	return json.Marshal(artifact)
}

func Unmarshal(data []byte) (Artifact, error) {
	if len(data) == 0 || len(data) > 16<<20 {
		return Artifact{}, errors.New("replay artifact size is invalid")
	}
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	var artifact Artifact
	if err := decoder.Decode(&artifact); err != nil {
		return Artifact{}, err
	}
	var trailing any
	if err := decoder.Decode(&trailing); err != io.EOF {
		return Artifact{}, errors.New("replay artifact contains trailing data")
	}
	if err := validateArtifactEnvelope(artifact); err != nil {
		return Artifact{}, err
	}
	return artifact, nil
}
