// Package util includes common, shared functions
package util

import (
	"database/sql"
	"math/rand"
	"time"

	"go.uber.org/zap"
)

// RetryDBCommitMaxSleep - max sleep time during DB commit retries, in seconds
const RetryDBCommitMaxSleep = 3

// RetryDBCommit commits a db transaction, using up to maxRetries retries if necessary
func RetryDBCommit(tx *sql.Tx, maxRetries int, logger *zap.Logger) error {
	var err error
	for i := 0; i < maxRetries; i++ {
		err = tx.Commit()
		if err == nil {
			return nil
		}

		logger.Sugar().Warnw("Unable to commit database transaction",
			"attempt #", i,
			"max_attempts", maxRetries,
			"error", err,
		)

		// Exponential backoff would be overkill here, but adding a bit of jitter
		// to sleep a short time is reasonable
		jitter := time.Duration(rand.Int63n(int64(RetryDBCommitMaxSleep * time.Second)))
		time.Sleep(jitter)
	}

	return err
}
