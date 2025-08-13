// Package repeatable предоставляет функции для повторного выполнения операций с задержкой.
package repeatable

import (
	"fmt"
	"time"
)

// DoWithTries запускает функцию fn несколько раз, пока она не выполнится успешно или не исчерпает максимальное количество попыток.
func DoWithTries(fn func() error, maxAttempts int, delay time.Duration) (err error) {
	for i := 0; i < maxAttempts; i++ {
		err = fn()
		if err == nil {
			return nil
		}
		time.Sleep(delay)
	}
	return fmt.Errorf("failed after %d attempts: %w", maxAttempts, err)
}
