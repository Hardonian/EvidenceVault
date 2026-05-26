package persistence

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
)

type FileStore struct {
	mu    sync.RWMutex
	dir   string
	state State
}

func NewFileStore(dir string) (*FileStore, error) {
	if dir == "" {
		return nil, ErrDataDirRequired
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, fmt.Errorf("create DATA_DIR: %w", err)
	}
	testFile := filepath.Join(dir, ".writecheck")
	if err := os.WriteFile(testFile, []byte("ok"), 0o644); err != nil {
		return nil, fmt.Errorf("DATA_DIR is not writable: %w", err)
	}
	_ = os.Remove(testFile)
	fs := &FileStore{dir: dir, state: emptyState()}
	if err := fs.load(); err != nil {
		return nil, err
	}
	return fs, nil
}
func (f *FileStore) Read(fn func(*State) error) error {
	f.mu.RLock()
	defer f.mu.RUnlock()
	return fn(&f.state)
}
func (f *FileStore) Write(fn func(*State) error) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if err := fn(&f.state); err != nil {
		return err
	}
	return f.persistLocked()
}
func (f *FileStore) load() error {
	files := map[string]any{"tenants.json": &f.state.Tenants, "evidence_items.json": &f.state.Evidence, "evidence_files.json": &f.state.EvidenceFile, "reminder_logs.json": &f.state.ReminderSent, "proofpacks.json": &f.state.Proofpacks, "audit_logs.json": &f.state.AuditLogs, "stripe_events.json": &f.state.StripeEvents, "review_snapshots.json": &f.state.ReviewSnapshots, "operational_events.json": &f.state.OperationalEvents, "activation_milestones.json": &f.state.Activation, "operational_snapshots.json": &f.state.OperationalSnapshots, "review_reports.json": &f.state.ReviewReports, "unresolved_issues.json": &f.state.UnresolvedIssues, "adjudication_events.json": &f.state.AdjudicationEvents}
	for n, tgt := range files {
		p := filepath.Join(f.dir, n)
		b, err := os.ReadFile(p)
		if os.IsNotExist(err) {
			continue
		}
		if err != nil {
			return fmt.Errorf("read %s: %w", n, err)
		}
		if len(b) == 0 {
			continue
		}
		if err := json.Unmarshal(b, tgt); err != nil {
			return fmt.Errorf("decode %s: %w", n, err)
		}
	}
	return nil
}
func (f *FileStore) persistLocked() error {
	files := map[string]any{"tenants.json": f.state.Tenants, "evidence_items.json": f.state.Evidence, "evidence_files.json": f.state.EvidenceFile, "reminder_logs.json": f.state.ReminderSent, "proofpacks.json": f.state.Proofpacks, "audit_logs.json": f.state.AuditLogs, "stripe_events.json": f.state.StripeEvents, "review_snapshots.json": f.state.ReviewSnapshots, "operational_events.json": f.state.OperationalEvents, "activation_milestones.json": f.state.Activation, "operational_snapshots.json": f.state.OperationalSnapshots, "review_reports.json": f.state.ReviewReports, "unresolved_issues.json": f.state.UnresolvedIssues, "adjudication_events.json": f.state.AdjudicationEvents}
	for n, v := range files {
		b, err := json.MarshalIndent(v, "", "  ")
		if err != nil {
			return err
		}
		tmp := filepath.Join(f.dir, n+".tmp")
		final := filepath.Join(f.dir, n)
		
		file, err := os.OpenFile(tmp, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0o644)
		if err != nil {
			return err
		}
		if _, err := file.Write(b); err != nil {
			file.Close()
			return err
		}
		if err := file.Sync(); err != nil {
			file.Close()
			return err
		}
		if err := file.Close(); err != nil {
			return err
		}

		if err := os.Rename(tmp, final); err != nil {
			return err
		}
	}
	return nil
}
