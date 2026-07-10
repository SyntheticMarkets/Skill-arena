package workers

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"time"
)

type RecoveryReport struct {
	Path                 string            `json:"path"`
	DatabaseRestore      bool              `json:"databaseRestore"`
	ReplayRestore        bool              `json:"replayRestore"`
	ConfigurationRestore bool              `json:"configurationRestore"`
	JobQueueRestore      bool              `json:"jobQueueRestore"`
	Passed               bool              `json:"passed"`
	CheckedAt            time.Time         `json:"checkedAt"`
	Errors               map[string]string `json:"errors,omitempty"`
}

func ValidateRecovery(ctx context.Context, backupPath string) (*RecoveryReport, error) {
	report := &RecoveryReport{
		Path:      backupPath,
		CheckedAt: time.Now().UTC(),
		Errors:    map[string]string{},
	}
	checkJSON := func(name string, required bool) bool {
		select {
		case <-ctx.Done():
			report.Errors[name] = ctx.Err().Error()
			return false
		default:
		}
		path := filepath.Join(backupPath, name)
		content, err := os.ReadFile(path)
		if err != nil {
			if required {
				report.Errors[name] = err.Error()
			}
			return false
		}
		var payload any
		if err := json.Unmarshal(content, &payload); err != nil {
			report.Errors[name] = err.Error()
			return false
		}
		return true
	}

	report.DatabaseRestore = checkOptionalJSONGroup(checkJSON, []string{"users.json", "wallets.json", "sessions.json", "ledger.json"})
	report.JobQueueRestore = checkJSON("jobs.json", true)
	report.ConfigurationRestore = checkJSON("backup_manifest.json", true)
	report.ReplayRestore = pathExists(filepath.Join(backupPath, "replay_exports")) || checkJSON("replays.json", false)
	if !report.ReplayRestore {
		report.Errors["replay_restore"] = "no replay_exports directory or replays.json found"
	}
	report.Passed = report.DatabaseRestore && report.JobQueueRestore && report.ConfigurationRestore && report.ReplayRestore
	if len(report.Errors) == 0 {
		report.Errors = nil
	}
	return report, nil
}

func checkOptionalJSONGroup(check func(string, bool) bool, names []string) bool {
	seen := false
	for _, name := range names {
		if check(name, false) {
			seen = true
		}
	}
	return seen
}

func pathExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}
