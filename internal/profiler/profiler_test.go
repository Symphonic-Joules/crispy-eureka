package profiler

import (
	"context"
	"errors"
	"log/slog"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestProfileString(t *testing.T) {
	t.Run("formats profile data correctly", func(t *testing.T) {
		// Given a profile with data
		now := time.Now()
		profile := &Profile{
			Operation:    "test_operation",
			Duration:     100 * time.Millisecond,
			MemoryBefore: 1000,
			MemoryAfter:  2000,
			MemoryDelta:  1000,
			LinesCount:   50,
			BytesCount:   1024,
			Timestamp:    now,
		}

		// When we convert it to string
		result := profile.String()

		// Then it should contain all relevant information
		assert.Contains(t, result, "test_operation")
		assert.Contains(t, result, "100ms")
		assert.Contains(t, result, "1000B")
		assert.Contains(t, result, "lines=50")
		assert.Contains(t, result, "bytes=1024")
	})

	t.Run("handles negative memory delta", func(t *testing.T) {
		// Given a profile with negative memory delta
		profile := &Profile{
			Operation:    "test",
			Duration:     1 * time.Millisecond,
			MemoryBefore: 5000,
			MemoryAfter:  3000,
			MemoryDelta:  -2000,
			Timestamp:    time.Now(),
		}

		// When we convert it to string
		result := profile.String()

		// Then it should show negative delta
		assert.Contains(t, result, "-2000B")
	})

	t.Run("formats timestamp correctly", func(t *testing.T) {
		// Given a profile with a specific timestamp
		timestamp := time.Date(2024, 1, 1, 15, 30, 45, 123456789, time.UTC)
		profile := &Profile{
			Operation: "test",
			Duration:  1 * time.Millisecond,
			Timestamp: timestamp,
		}

		// When we convert it to string
		result := profile.String()

		// Then it should contain formatted timestamp
		assert.Contains(t, result, "15:30:45")
	})
}

func TestSafeMemoryDelta(t *testing.T) {
	t.Run("calculates positive delta correctly", func(t *testing.T) {
		// Given memory values where after > before
		before := uint64(1000)
		after := uint64(2000)

		// When we calculate the delta
		delta := safeMemoryDelta(after, before)

		// Then it should return the positive difference
		assert.Equal(t, int64(1000), delta)
	})

	t.Run("calculates negative delta correctly", func(t *testing.T) {
		// Given memory values where before > after
		before := uint64(5000)
		after := uint64(3000)

		// When we calculate the delta
		delta := safeMemoryDelta(after, before)

		// Then it should return the negative difference
		assert.Equal(t, int64(-2000), delta)
	})

	t.Run("handles zero delta", func(t *testing.T) {
		// Given equal memory values
		value := uint64(1000)

		// When we calculate the delta
		delta := safeMemoryDelta(value, value)

		// Then it should return zero
		assert.Equal(t, int64(0), delta)
	})

	t.Run("handles very large positive delta safely", func(t *testing.T) {
		// Given very large memory values that would overflow int64
		before := uint64(0)
		after := uint64(9223372036854775808) // > MaxInt64

		// When we calculate the delta
		delta := safeMemoryDelta(after, before)

		// Then it should cap at MaxInt64
		assert.Equal(t, int64(9223372036854775807), delta) // MaxInt64
	})

	t.Run("handles very large negative delta safely", func(t *testing.T) {
		// Given very large memory values that would overflow int64
		before := uint64(9223372036854775808) // > MaxInt64
		after := uint64(0)

		// When we calculate the delta
		delta := safeMemoryDelta(after, before)

		// Then it should cap at -MaxInt64
		assert.Equal(t, int64(-9223372036854775807), delta) // -MaxInt64
	})

	t.Run("handles both values greater than MaxInt64", func(t *testing.T) {
		// Given both values exceed MaxInt64
		before := uint64(10000000000000000000)
		after := uint64(10000000000000001000)

		// When we calculate the delta
		delta := safeMemoryDelta(after, before)

		// Then it should return the capped difference
		assert.Equal(t, int64(1000), delta)
	})

	t.Run("handles max uint64 values", func(t *testing.T) {
		// Given maximum uint64 value
		before := uint64(18446744073709551615) // Max uint64
		after := uint64(0)

		// When we calculate the delta
		delta := safeMemoryDelta(after, before)

		// Then it should return capped negative value
		assert.Equal(t, int64(-9223372036854775807), delta) // -MaxInt64
	})
}

func TestNew(t *testing.T) {
	t.Run("creates profiler with logger", func(t *testing.T) {
		// Given a logger
		logger := slog.Default()

		// When we create a profiler
		profiler := New(logger, true)

		// Then it should be properly initialized
		assert.NotNil(t, profiler)
		assert.Equal(t, logger, profiler.logger)
		assert.True(t, profiler.enabled)
	})

	t.Run("creates disabled profiler", func(t *testing.T) {
		// Given a logger
		logger := slog.Default()

		// When we create a disabled profiler
		profiler := New(logger, false)

		// Then it should be initialized but disabled
		assert.NotNil(t, profiler)
		assert.False(t, profiler.enabled)
	})

	t.Run("creates profiler with nil logger", func(t *testing.T) {
		// When we create a profiler with nil logger
		profiler := New(nil, true)

		// Then it should still be valid
		assert.NotNil(t, profiler)
		assert.Nil(t, profiler.logger)
	})
}

func TestProfileFunc(t *testing.T) {
	t.Run("executes function and returns profile when enabled", func(t *testing.T) {
		// Given an enabled profiler
		profiler := New(nil, true)
		executed := false

		// When we profile a function
		profile, err := profiler.ProfileFunc(context.Background(), "test_op", func() error {
			executed = true
			time.Sleep(10 * time.Millisecond)
			return nil
		})

		// Then the function should be executed
		assert.True(t, executed)
		assert.NoError(t, err)
		assert.NotNil(t, profile)
		assert.Equal(t, "test_op", profile.Operation)
		assert.Greater(t, profile.Duration, time.Duration(0))
	})

	t.Run("executes function without profiling when disabled", func(t *testing.T) {
		// Given a disabled profiler
		profiler := New(nil, false)
		executed := false

		// When we profile a function
		profile, err := profiler.ProfileFunc(context.Background(), "test_op", func() error {
			executed = true
			return nil
		})

		// Then the function should be executed but no profile returned
		assert.True(t, executed)
		assert.NoError(t, err)
		assert.Nil(t, profile)
	})

	t.Run("captures function errors", func(t *testing.T) {
		// Given an enabled profiler and a function that errors
		profiler := New(nil, true)
		expectedErr := errors.New("test error")

		// When we profile the function
		profile, err := profiler.ProfileFunc(context.Background(), "error_op", func() error {
			return expectedErr
		})

		// Then the error should be returned along with profile
		assert.Equal(t, expectedErr, err)
		assert.NotNil(t, profile)
		assert.Equal(t, "error_op", profile.Operation)
	})

	t.Run("measures memory usage", func(t *testing.T) {
		// Given an enabled profiler
		profiler := New(nil, true)

		// When we profile a function that allocates memory
		profile, err := profiler.ProfileFunc(context.Background(), "memory_op", func() error {
			// Allocate some memory
			_ = make([]byte, 1024*1024) // 1MB
			return nil
		})

		// Then memory delta should be captured
		assert.NoError(t, err)
		assert.NotNil(t, profile)
		assert.NotEqual(t, uint64(0), profile.MemoryBefore)
		assert.NotEqual(t, uint64(0), profile.MemoryAfter)
	})

	t.Run("measures execution duration", func(t *testing.T) {
		// Given an enabled profiler
		profiler := New(nil, true)

		// When we profile a function with known duration
		sleepDuration := 50 * time.Millisecond
		profile, err := profiler.ProfileFunc(context.Background(), "duration_op", func() error {
			time.Sleep(sleepDuration)
			return nil
		})

		// Then duration should be measured accurately
		assert.NoError(t, err)
		assert.NotNil(t, profile)
		assert.GreaterOrEqual(t, profile.Duration, sleepDuration)
		// Allow some overhead, but should be close
		assert.LessOrEqual(t, profile.Duration, sleepDuration+20*time.Millisecond)
	})

	t.Run("sets timestamp", func(t *testing.T) {
		// Given an enabled profiler
		profiler := New(nil, true)
		before := time.Now()

		// When we profile a function
		profile, err := profiler.ProfileFunc(context.Background(), "timestamp_op", func() error {
			return nil
		})

		after := time.Now()

		// Then timestamp should be within the execution window
		assert.NoError(t, err)
		assert.NotNil(t, profile)
		assert.True(t, profile.Timestamp.After(before) || profile.Timestamp.Equal(before))
		assert.True(t, profile.Timestamp.Before(after) || profile.Timestamp.Equal(after))
	})

	t.Run("works with context", func(t *testing.T) {
		// Given an enabled profiler and a context
		profiler := New(nil, true)
		ctx := context.Background()

		// When we profile a function that uses context
		profile, err := profiler.ProfileFunc(ctx, "context_op", func() error {
			return nil
		})

		// Then it should work correctly
		assert.NoError(t, err)
		assert.NotNil(t, profile)
	})

	t.Run("handles panics gracefully", func(t *testing.T) {
		// Given an enabled profiler
		profiler := New(nil, true)

		// When we profile a function that panics
		// Then it should panic (as expected in Go)
		assert.Panics(t, func() {
			_, _ = profiler.ProfileFunc(context.Background(), "panic_op", func() error {
				panic("test panic")
			})
		})
	})

	t.Run("profile has all fields populated when enabled", func(t *testing.T) {
		// Given an enabled profiler
		profiler := New(nil, true)

		// When we profile a function
		profile, err := profiler.ProfileFunc(context.Background(), "complete_op", func() error {
			return nil
		})

		// Then all profile fields should be populated
		assert.NoError(t, err)
		require.NotNil(t, profile)
		assert.Equal(t, "complete_op", profile.Operation)
		assert.NotEqual(t, time.Duration(0), profile.Duration)
		assert.NotEqual(t, uint64(0), profile.MemoryBefore)
		assert.NotEqual(t, uint64(0), profile.MemoryAfter)
		assert.NotEqual(t, time.Time{}, profile.Timestamp)
	})

	t.Run("multiple sequential profiles work correctly", func(t *testing.T) {
		// Given an enabled profiler
		profiler := New(nil, true)

		// When we profile multiple operations
		profile1, err1 := profiler.ProfileFunc(context.Background(), "op1", func() error {
			return nil
		})
		profile2, err2 := profiler.ProfileFunc(context.Background(), "op2", func() error {
			return nil
		})

		// Then both should succeed independently
		assert.NoError(t, err1)
		assert.NoError(t, err2)
		assert.NotNil(t, profile1)
		assert.NotNil(t, profile2)
		assert.Equal(t, "op1", profile1.Operation)
		assert.Equal(t, "op2", profile2.Operation)
		assert.True(t, profile2.Timestamp.After(profile1.Timestamp) || profile2.Timestamp.Equal(profile1.Timestamp))
	})

	t.Run("function returning nil error is handled", func(t *testing.T) {
		// Given an enabled profiler
		profiler := New(nil, true)

		// When we profile a function that returns nil
		profile, err := profiler.ProfileFunc(context.Background(), "nil_error_op", func() error {
			return nil
		})

		// Then no error should be reported
		assert.NoError(t, err)
		assert.NotNil(t, profile)
	})
}

func TestProfilerWithLogger(t *testing.T) {
	t.Run("logs profile information when logger is provided", func(t *testing.T) {
		// This test verifies that the profiler works with a logger
		// In a real scenario, you'd use a test logger to capture output
		logger := slog.Default()
		profiler := New(logger, true)

		// When we profile a function
		profile, err := profiler.ProfileFunc(context.Background(), "logged_op", func() error {
			return nil
		})

		// Then it should complete successfully
		// (logging is a side effect, we just verify it doesn't break)
		assert.NoError(t, err)
		assert.NotNil(t, profile)
	})
}