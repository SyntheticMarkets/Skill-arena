package realtime

import (
	"context"
	"database/sql"
	"fmt"
	"sort"
	"sync"
	"time"
)

type phase9PostgresStats struct {
	transactions      int64
	blocksRead        int64
	blocksHit         int64
	tempFiles         int64
	tempBytes         int64
	deadlocks         int64
	blockReadMS       float64
	blockWriteMS      float64
	walRecords        int64
	walFPI            int64
	walBytes          int64
	walWrites         int64
	walSyncs          int64
	walBuffersFull    int64
	walWriteMS        float64
	walSyncMS         float64
	checkpoints       int64
	checkpointWriteMS float64
	checkpointSyncMS  float64
	checkpointBuffers int64
	cleanBuffers      int64
	backendWrites     int64
	backendFsyncs     int64
	ioWrites          int64
	ioWriteMS         float64
	ioFsyncs          int64
	ioFsyncMS         float64
}

type phase9PostgresDiagnostics struct {
	db *sql.DB

	stop     chan struct{}
	stopped  chan struct{}
	stopOnce sync.Once

	mu                 sync.Mutex
	waitObservations   map[string]int64
	samples            int64
	maxActiveBackends  int64
	maxUngrantedLocks  int64
	samplingErrorCount int64
	before             phase9PostgresStats
	settings           string
}

func startPhase9PostgresDiagnostics(
	ctx context.Context,
	databaseURL string,
) (*phase9PostgresDiagnostics, error) {
	diagnosticDB, err := sql.Open("pgx", databaseURL)
	if err != nil {
		return nil, err
	}
	diagnosticDB.SetMaxOpenConns(1)
	diagnosticDB.SetMaxIdleConns(1)
	if err := diagnosticDB.PingContext(ctx); err != nil {
		_ = diagnosticDB.Close()
		return nil, err
	}
	diagnostics := &phase9PostgresDiagnostics{
		db: diagnosticDB, stop: make(chan struct{}), stopped: make(chan struct{}),
		waitObservations: map[string]int64{},
	}
	diagnostics.settings, err = diagnostics.loadSettings(ctx)
	if err != nil {
		_ = diagnosticDB.Close()
		return nil, err
	}
	diagnostics.before, err = diagnostics.stats(ctx)
	if err != nil {
		_ = diagnosticDB.Close()
		return nil, err
	}
	go diagnostics.sample(ctx)
	return diagnostics, nil
}

func (d *phase9PostgresDiagnostics) loadSettings(ctx context.Context) (string, error) {
	var version, synchronousCommit, walSyncMethod string
	var fullPageWrites, trackIO, trackWALIO string
	err := d.db.QueryRowContext(ctx, `SELECT
current_setting('server_version'),
current_setting('synchronous_commit'),
current_setting('wal_sync_method'),
current_setting('full_page_writes'),
current_setting('track_io_timing'),
current_setting('track_wal_io_timing')`).Scan(
		&version, &synchronousCommit, &walSyncMethod, &fullPageWrites,
		&trackIO, &trackWALIO,
	)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf(
		"pg_settings version=%s synchronous_commit=%s wal_sync_method=%s full_page_writes=%s track_io_timing=%s track_wal_io_timing=%s",
		version, synchronousCommit, walSyncMethod, fullPageWrites, trackIO, trackWALIO,
	), nil
}

func (d *phase9PostgresDiagnostics) sample(ctx context.Context) {
	defer close(d.stopped)
	ticker := time.NewTicker(50 * time.Millisecond)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-d.stop:
			return
		case <-ticker.C:
			d.sampleOnce(ctx)
		}
	}
}

func (d *phase9PostgresDiagnostics) sampleOnce(ctx context.Context) {
	rows, err := d.db.QueryContext(ctx, `SELECT
COALESCE(wait_event_type,'none'),COALESCE(wait_event,'none'),count(*)
FROM pg_stat_activity
WHERE datname=current_database() AND pid<>pg_backend_pid() AND state='active'
GROUP BY wait_event_type,wait_event`)
	if err != nil {
		d.recordSamplingError()
		return
	}
	active := int64(0)
	observations := map[string]int64{}
	for rows.Next() {
		var waitType string
		var waitEvent string
		var count int64
		if err := rows.Scan(&waitType, &waitEvent, &count); err != nil {
			_ = rows.Close()
			d.recordSamplingError()
			return
		}
		active += count
		observations[waitType+"."+waitEvent] += count
	}
	if err := rows.Close(); err != nil {
		d.recordSamplingError()
		return
	}
	var ungranted int64
	if err := d.db.QueryRowContext(
		ctx, `SELECT count(*) FROM pg_locks WHERE NOT granted`,
	).Scan(&ungranted); err != nil {
		d.recordSamplingError()
		return
	}
	d.mu.Lock()
	defer d.mu.Unlock()
	d.samples++
	d.maxActiveBackends = max(d.maxActiveBackends, active)
	d.maxUngrantedLocks = max(d.maxUngrantedLocks, ungranted)
	for name, count := range observations {
		d.waitObservations[name] += count
	}
}

func (d *phase9PostgresDiagnostics) recordSamplingError() {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.samplingErrorCount++
}

func (d *phase9PostgresDiagnostics) Stop(
	ctx context.Context,
) (phase9PostgresStats, error) {
	d.stopOnce.Do(func() { close(d.stop) })
	<-d.stopped
	after, err := d.stats(ctx)
	closeErr := d.db.Close()
	if err != nil {
		return phase9PostgresStats{}, err
	}
	if closeErr != nil {
		return phase9PostgresStats{}, closeErr
	}
	return after.delta(d.before), nil
}

func (d *phase9PostgresDiagnostics) stats(
	ctx context.Context,
) (phase9PostgresStats, error) {
	var result phase9PostgresStats
	err := d.db.QueryRowContext(ctx, `SELECT
xact_commit,blks_read,blks_hit,temp_files,temp_bytes,deadlocks,
blk_read_time,blk_write_time
FROM pg_stat_database WHERE datname=current_database()`).Scan(
		&result.transactions, &result.blocksRead, &result.blocksHit,
		&result.tempFiles, &result.tempBytes, &result.deadlocks,
		&result.blockReadMS, &result.blockWriteMS,
	)
	if err != nil {
		return phase9PostgresStats{}, err
	}
	err = d.db.QueryRowContext(ctx, `SELECT
wal_records,wal_fpi,wal_bytes::bigint,wal_write,wal_sync,wal_buffers_full,
wal_write_time,wal_sync_time
FROM pg_stat_wal`).Scan(
		&result.walRecords, &result.walFPI, &result.walBytes, &result.walWrites,
		&result.walSyncs, &result.walBuffersFull,
		&result.walWriteMS, &result.walSyncMS,
	)
	if err != nil {
		return phase9PostgresStats{}, err
	}
	err = d.db.QueryRowContext(ctx, `SELECT
num_timed+num_requested,write_time,sync_time,buffers_written
FROM pg_stat_checkpointer`).Scan(
		&result.checkpoints, &result.checkpointWriteMS,
		&result.checkpointSyncMS, &result.checkpointBuffers,
	)
	if err != nil {
		return phase9PostgresStats{}, err
	}
	err = d.db.QueryRowContext(ctx, `SELECT
buffers_clean,buffers_backend,buffers_backend_fsync
FROM pg_stat_bgwriter`).Scan(
		&result.cleanBuffers, &result.backendWrites, &result.backendFsyncs,
	)
	if err != nil {
		return phase9PostgresStats{}, err
	}
	err = d.db.QueryRowContext(ctx, `SELECT
COALESCE(sum(writes),0),COALESCE(sum(write_time),0),
COALESCE(sum(fsyncs),0),COALESCE(sum(fsync_time),0)
FROM pg_stat_io`).Scan(
		&result.ioWrites, &result.ioWriteMS, &result.ioFsyncs, &result.ioFsyncMS,
	)
	return result, err
}

func (s phase9PostgresStats) delta(before phase9PostgresStats) phase9PostgresStats {
	return phase9PostgresStats{
		transactions:      s.transactions - before.transactions,
		blocksRead:        s.blocksRead - before.blocksRead,
		blocksHit:         s.blocksHit - before.blocksHit,
		tempFiles:         s.tempFiles - before.tempFiles,
		tempBytes:         s.tempBytes - before.tempBytes,
		deadlocks:         s.deadlocks - before.deadlocks,
		blockReadMS:       s.blockReadMS - before.blockReadMS,
		blockWriteMS:      s.blockWriteMS - before.blockWriteMS,
		walRecords:        s.walRecords - before.walRecords,
		walFPI:            s.walFPI - before.walFPI,
		walBytes:          s.walBytes - before.walBytes,
		walWrites:         s.walWrites - before.walWrites,
		walSyncs:          s.walSyncs - before.walSyncs,
		walBuffersFull:    s.walBuffersFull - before.walBuffersFull,
		walWriteMS:        s.walWriteMS - before.walWriteMS,
		walSyncMS:         s.walSyncMS - before.walSyncMS,
		checkpoints:       s.checkpoints - before.checkpoints,
		checkpointWriteMS: s.checkpointWriteMS - before.checkpointWriteMS,
		checkpointSyncMS:  s.checkpointSyncMS - before.checkpointSyncMS,
		checkpointBuffers: s.checkpointBuffers - before.checkpointBuffers,
		cleanBuffers:      s.cleanBuffers - before.cleanBuffers,
		backendWrites:     s.backendWrites - before.backendWrites,
		backendFsyncs:     s.backendFsyncs - before.backendFsyncs,
		ioWrites:          s.ioWrites - before.ioWrites,
		ioWriteMS:         s.ioWriteMS - before.ioWriteMS,
		ioFsyncs:          s.ioFsyncs - before.ioFsyncs,
		ioFsyncMS:         s.ioFsyncMS - before.ioFsyncMS,
	}
}

func (d *phase9PostgresDiagnostics) logLines() []string {
	d.mu.Lock()
	defer d.mu.Unlock()
	lines := []string{fmt.Sprintf(
		"pg_samples=%d max_active_backends=%d max_ungranted_locks=%d sampling_errors=%d",
		d.samples, d.maxActiveBackends, d.maxUngrantedLocks,
		d.samplingErrorCount,
	)}
	lines = append(lines, d.settings)
	names := make([]string, 0, len(d.waitObservations))
	for name := range d.waitObservations {
		names = append(names, name)
	}
	sort.Strings(names)
	for _, name := range names {
		lines = append(lines, fmt.Sprintf(
			"pg_wait=%s backend_observations=%d",
			name, d.waitObservations[name],
		))
	}
	return lines
}
