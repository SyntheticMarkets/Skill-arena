package db

import (
	"context"
	"database/sql"
	"hash/fnv"
	"sync"
	"time"
)

type gameActionBatchJob struct {
	query    string
	args     []any
	response chan gameActionBatchResult
}

type gameActionBatchResult struct {
	affected int64
	err      error
}

type gameActionBatcher struct {
	database *sql.DB
	size     int
	window   time.Duration
	shards   []chan gameActionBatchJob
	wait     sync.WaitGroup
	once     sync.Once
}

func newGameActionBatcher(
	database *sql.DB,
	workers, size int,
	window time.Duration,
) *gameActionBatcher {
	if workers < 1 {
		workers = 1
	}
	if size < 1 {
		size = 16
	}
	if window <= 0 {
		window = 500 * time.Microsecond
	}
	batcher := &gameActionBatcher{
		database: database, size: size, window: window,
		shards: make([]chan gameActionBatchJob, workers),
	}
	for index := range batcher.shards {
		batcher.shards[index] = make(chan gameActionBatchJob, size*4)
		batcher.wait.Add(1)
		go batcher.run(batcher.shards[index])
	}
	return batcher
}

func (b *gameActionBatcher) Execute(
	ctx context.Context,
	matchID, query string,
	args []any,
) (int64, error) {
	job := gameActionBatchJob{
		query: query, args: args,
		response: make(chan gameActionBatchResult, 1),
	}
	shard := b.shards[gameActionShard(matchID, len(b.shards))]
	select {
	case shard <- job:
	case <-ctx.Done():
		return 0, ctx.Err()
	}
	// Once admitted, wait for the durable outcome. Returning on cancellation here
	// would make a committed action indistinguishable from an abandoned one.
	result := <-job.response
	return result.affected, result.err
}

func (b *gameActionBatcher) Close() {
	b.once.Do(func() {
		for _, shard := range b.shards {
			close(shard)
		}
		b.wait.Wait()
	})
}

func (b *gameActionBatcher) run(input <-chan gameActionBatchJob) {
	defer b.wait.Done()
	for first := range input {
		batch := []gameActionBatchJob{first}
		timer := time.NewTimer(b.window)
	collect:
		for len(batch) < b.size {
			select {
			case job, ok := <-input:
				if !ok {
					break collect
				}
				batch = append(batch, job)
			case <-timer.C:
				break collect
			}
		}
		if !timer.Stop() {
			select {
			case <-timer.C:
			default:
			}
		}
		b.commit(batch)
	}
}

func (b *gameActionBatcher) commit(batch []gameActionBatchJob) {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	tx, err := b.database.BeginTx(ctx, nil)
	if err != nil {
		completeGameActionBatch(batch, nil, err)
		return
	}
	affected := make([]int64, len(batch))
	for index, job := range batch {
		result, execErr := tx.ExecContext(ctx, job.query, job.args...)
		if execErr != nil {
			_ = tx.Rollback()
			completeGameActionBatch(batch, nil, execErr)
			return
		}
		affected[index], _ = result.RowsAffected()
	}
	if err := tx.Commit(); err != nil {
		completeGameActionBatch(batch, nil, err)
		return
	}
	completeGameActionBatch(batch, affected, nil)
}

func completeGameActionBatch(
	batch []gameActionBatchJob,
	affected []int64,
	err error,
) {
	for index, job := range batch {
		result := gameActionBatchResult{err: err}
		if err == nil {
			result.affected = affected[index]
		}
		job.response <- result
	}
}

func gameActionShard(matchID string, count int) int {
	hash := fnv.New32a()
	_, _ = hash.Write([]byte(matchID))
	return int(hash.Sum32() % uint32(count))
}
