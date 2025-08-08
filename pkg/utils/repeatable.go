package repeatable

import (
	"fmt"
	"time"
)

func DoWithTries(fn func() error, maxAttempts int, delay time.Duration) (err error) {
	for i := 0; i < maxAttempts; i++ {
		err = fn()
		if err == nil {
			return nil
		}
		time.Sleep(delay) // Wait before retrying
	}
	return fmt.Errorf("failed after %d attempts: %w", maxAttempts, err)
}
