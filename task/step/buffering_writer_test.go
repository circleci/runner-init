package step

import (
	"bytes"
	"testing"
	"time"

	"gotest.tools/v3/assert"
	"gotest.tools/v3/assert/cmp"
)

func TestBufferingWriter(t *testing.T) {
	t.Run("Calls to Write and Close do not block", func(t *testing.T) {
		var b bytes.Buffer
		writer := NewBufferingWriter(&b, time.Millisecond, 1024, nil)

		go func() {
			for i := 0; i < 100; i++ {
				_, err := writer.Write([]byte("content"))
				if err != nil {
					assert.Check(t, cmp.ErrorIs(err, ErrWriterClosed), "Unexpected error on write")
				}
			}
		}()

		// Close writer concurrently
		go func() {
			err := writer.Close()
			assert.NilError(t, err)
		}()

		// Give the goroutines some time to finish their work
		time.Sleep(500 * time.Millisecond)

		// Test again after closing
		_, err := writer.Write([]byte("content"))
		assert.Check(t, cmp.ErrorIs(err, ErrWriterClosed), "Future writes should fail with ErrWriterClosed")
	})
}
