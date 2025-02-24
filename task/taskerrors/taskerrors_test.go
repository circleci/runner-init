package taskerrors

import (
	"errors"
	"fmt"
	"testing"

	"gotest.tools/v3/assert"
	"gotest.tools/v3/assert/cmp"
)

func TestNewHandledError(t *testing.T) {
	t.Run("Is a handled error", func(t *testing.T) {
		err := NewHandledError(fmt.Errorf("something wrong happened"))

		assert.Check(t, errors.As(err, &HandledError{}))
		assert.Check(t, cmp.ErrorContains(err, "handled: something wrong happened"))
	})

	t.Run("Isn't a handled error", func(t *testing.T) {
		err := fmt.Errorf("fatal")

		assert.Check(t, !errors.As(err, &HandledError{}))
	})
}
