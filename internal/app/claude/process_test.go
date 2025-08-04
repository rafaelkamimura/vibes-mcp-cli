package claude

import (
	"context"
	"os"
	"os/exec"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"openai-cli/internal/app/testutil"
)

func TestNewProcess(t *testing.T) {
	logger := testutil.TestLogger(t)
	cmd := exec.Command("echo", "test")

	process := NewProcess(cmd, logger)

	assert.NotNil(t, process)
	assert.NotEmpty(t, process.ID)
	assert.Equal(t, cmd, process.cmd)
	assert.Equal(t, ProcessStateCreated, process.GetState())
	assert.Equal(t, -1, process.exitCode)
	assert.NotNil(t, process.outputBuffer)
	assert.NotNil(t, process.outputChan)
	assert.NotNil(t, process.done)
}

func TestProcess_StateManagement(t *testing.T) {
	logger := testutil.TestLogger(t)

	t.Run("state transitions", func(t *testing.T) {
		cmd := exec.Command("echo", "test")
		process := NewProcess(cmd, logger)

		// Initial state
		assert.Equal(t, ProcessStateCreated, process.GetState())
		assert.False(t, process.IsRunning())
		assert.False(t, process.IsFinished())

		// Start process
		err := process.Start()
		require.NoError(t, err)

		// Should be running
		assert.Equal(t, ProcessStateRunning, process.GetState())
		assert.True(t, process.IsRunning())
		assert.False(t, process.IsFinished())

		// Wait for completion
		result, err := process.Wait()
		assert.NoError(t, err)
		assert.NotNil(t, result)

		// Should be finished
		assert.Equal(t, ProcessStateFinished, process.GetState())
		assert.False(t, process.IsRunning())
		assert.True(t, process.IsFinished())
		assert.Equal(t, 0, process.GetExitCode())
	})

	t.Run("failed process state", func(t *testing.T) {
		cmd := exec.Command("sh", "-c", "exit 1")
		process := NewProcess(cmd, logger)

		err := process.Start()
		require.NoError(t, err)

		result, err := process.Wait()
		assert.Error(t, err)
		assert.NotNil(t, result)

		assert.Equal(t, ProcessStateError, process.GetState())
		assert.True(t, process.IsFinished())
		assert.Equal(t, 1, process.GetExitCode())
		assert.NotNil(t, process.GetError())
	})
}

func TestProcess_OutputHandling(t *testing.T) {
	logger := testutil.TestLogger(t)

	t.Run("basic output capture", func(t *testing.T) {
		cmd := exec.Command("echo", "test output")
		process := NewProcess(cmd, logger)

		err := process.Start()
		require.NoError(t, err)

		result, err := process.Wait()
		assert.NoError(t, err)

		output := process.GetOutput()
		assert.Contains(t, string(output), "test output")

		outputString := process.GetOutputString()
		assert.Contains(t, outputString, "test output")
	})

	t.Run("multiline output capture", func(t *testing.T) {
		cmd := exec.Command("sh", "-c", "echo line1; echo line2; echo line3")
		process := NewProcess(cmd, logger)

		err := process.Start()
		require.NoError(t, err)

		result, err := process.Wait()
		assert.NoError(t, err)

		output := process.GetOutputString()
		assert.Contains(t, output, "line1")
		assert.Contains(t, output, "line2")
		assert.Contains(t, output, "line3")
	})

	t.Run("stderr capture", func(t *testing.T) {
		cmd := exec.Command("sh", "-c", "echo stdout; echo stderr >&2")
		process := NewProcess(cmd, logger)

		err := process.Start()
		require.NoError(t, err)

		result, err := process.Wait()
		assert.NoError(t, err)

		output := process.GetOutputString()
		assert.Contains(t, output, "stdout")
		assert.Contains(t, output, "stderr")
	})

	t.Run("large output handling", func(t *testing.T) {
		// Generate large output
		cmd := exec.Command("sh", "-c", "for i in $(seq 1 1000); do echo \"Line $i with some additional content to make it longer\"; done")
		process := NewProcess(cmd, logger)

		err := process.Start()
		require.NoError(t, err)

		result, err := process.Wait()
		assert.NoError(t, err)

		output := process.GetOutputString()
		assert.Contains(t, output, "Line 1 with")
		assert.Contains(t, output, "Line 1000 with")
		assert.Greater(t, len(output), 50000) // Should be substantial
	})

	t.Run("output subscription", func(t *testing.T) {
		cmd := exec.Command("sh", "-c", "for i in 1 2 3; do echo \"Message $i\"; sleep 0.1; done")
		process := NewProcess(cmd, logger)

		// Subscribe before starting
		outputChan := process.SubscribeToOutput()

		err := process.Start()
		require.NoError(t, err)

		// Collect output from subscription
		var messages []string
		go func() {
			for output := range outputChan {
				if len(output) > 0 {
					messages = append(messages, strings.TrimSpace(string(output)))
				}
			}
		}()

		result, err := process.Wait()
		assert.NoError(t, err)

		// Give time for goroutine to process
		time.Sleep(100 * time.Millisecond)

		// Should have received messages
		assert.Greater(t, len(messages), 0)

		// Check that at least some expected messages were received
		allMessages := strings.Join(messages, " ")
		assert.Contains(t, allMessages, "Message")
	})
}

func TestProcess_InputHandling(t *testing.T) {
	logger := testutil.TestLogger(t)

	t.Run("send input to process", func(t *testing.T) {
		cmd := exec.Command("cat") // cat reads from stdin and outputs to stdout
		process := NewProcess(cmd, logger)

		err := process.Start()
		require.NoError(t, err)

		// Send input
		err = process.SendInput("test input")
		assert.NoError(t, err)

		// Close input to signal EOF
		err = process.CloseInput()
		assert.NoError(t, err)

		result, err := process.Wait()
		assert.NoError(t, err)

		output := process.GetOutputString()
		assert.Contains(t, output, "test input")
	})

	t.Run("send input line", func(t *testing.T) {
		cmd := exec.Command("sh", "-c", "read line; echo \"Received: $line\"")
		process := NewProcess(cmd, logger)

		err := process.Start()
		require.NoError(t, err)

		// Send input line (automatically adds newline)
		err = process.SendInputLine("test line")
		assert.NoError(t, err)

		result, err := process.Wait()
		assert.NoError(t, err)

		output := process.GetOutputString()
		assert.Contains(t, output, "Received: test line")
	})

	t.Run("send input to non-running process", func(t *testing.T) {
		cmd := exec.Command("echo", "test")
		process := NewProcess(cmd, logger)

		// Try to send input before starting
		err := process.SendInput("input")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "not running")
	})

	t.Run("multiple input sends", func(t *testing.T) {
		cmd := exec.Command("sh", "-c", "while read line; do echo \"Got: $line\"; done")
		process := NewProcess(cmd, logger)

		err := process.Start()
		require.NoError(t, err)

		// Send multiple inputs
		err = process.SendInputLine("first")
		assert.NoError(t, err)

		err = process.SendInputLine("second")
		assert.NoError(t, err)

		err = process.SendInputLine("third")
		assert.NoError(t, err)

		// Close input to end the loop
		err = process.CloseInput()
		assert.NoError(t, err)

		result, err := process.Wait()
		assert.NoError(t, err)

		output := process.GetOutputString()
		assert.Contains(t, output, "Got: first")
		assert.Contains(t, output, "Got: second")
		assert.Contains(t, output, "Got: third")
	})
}

func TestProcess_ProcessControl(t *testing.T) {
	logger := testutil.TestLogger(t)

	t.Run("kill process", func(t *testing.T) {
		cmd := exec.Command("sleep", "10") // Long running process
		process := NewProcess(cmd, logger)

		err := process.Start()
		require.NoError(t, err)

		assert.True(t, process.IsRunning())
		assert.Greater(t, process.GetPID(), 0)

		// Kill the process
		err = process.Kill()
		assert.NoError(t, err)

		// Wait for the process to be killed
		testutil.WaitForCondition(t, time.Second*2, func() bool {
			return process.GetState() == ProcessStateKilled
		})

		assert.Equal(t, ProcessStateKilled, process.GetState())
		assert.True(t, process.IsFinished())
		assert.False(t, process.IsRunning())
	})

	t.Run("kill already finished process", func(t *testing.T) {
		cmd := exec.Command("echo", "test")
		process := NewProcess(cmd, logger)

		err := process.Start()
		require.NoError(t, err)

		// Wait for completion
		result, err := process.Wait()
		assert.NoError(t, err)
		assert.True(t, process.IsFinished())

		// Killing already finished process should not error
		err = process.Kill()
		assert.NoError(t, err)
	})

	t.Run("signal process", func(t *testing.T) {
		cmd := exec.Command("sleep", "10")
		process := NewProcess(cmd, logger)

		err := process.Start()
		require.NoError(t, err)

		assert.True(t, process.IsRunning())

		// Send SIGTERM
		err = process.Signal(os.Interrupt)
		assert.NoError(t, err)

		// Wait for process to handle signal
		testutil.WaitForCondition(t, time.Second*2, func() bool {
			return process.IsFinished()
		})

		assert.True(t, process.IsFinished())
	})

	t.Run("signal non-running process", func(t *testing.T) {
		cmd := exec.Command("echo", "test")
		process := NewProcess(cmd, logger)

		// Try to signal before starting
		err := process.Signal(os.Interrupt)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "not running")
	})
}

func TestProcess_WaitWithContext(t *testing.T) {
	logger := testutil.TestLogger(t)

	t.Run("wait with timeout", func(t *testing.T) {
		cmd := exec.Command("sleep", "5") // Long running process
		process := NewProcess(cmd, logger)

		err := process.Start()
		require.NoError(t, err)

		// Wait with short timeout
		ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
		defer cancel()

		result, err := process.WaitWithContext(ctx)
		assert.Error(t, err)
		assert.Equal(t, context.DeadlineExceeded, err)
		assert.Nil(t, result)

		// Process should be killed due to timeout
		testutil.WaitForCondition(t, time.Second*2, func() bool {
			return process.GetState() == ProcessStateKilled
		})

		assert.Equal(t, ProcessStateKilled, process.GetState())
	})

	t.Run("wait with cancelled context", func(t *testing.T) {
		cmd := exec.Command("sleep", "5")
		process := NewProcess(cmd, logger)

		err := process.Start()
		require.NoError(t, err)

		ctx, cancel := context.WithCancel(context.Background())

		// Cancel context immediately
		cancel()

		result, err := process.WaitWithContext(ctx)
		assert.Error(t, err)
		assert.Equal(t, context.Canceled, err)
		assert.Nil(t, result)

		// Process should be killed due to cancellation
		testutil.WaitForCondition(t, time.Second*2, func() bool {
			return process.GetState() == ProcessStateKilled
		})

		assert.Equal(t, ProcessStateKilled, process.GetState())
	})

	t.Run("wait completes before timeout", func(t *testing.T) {
		cmd := exec.Command("echo", "fast")
		process := NewProcess(cmd, logger)

		err := process.Start()
		require.NoError(t, err)

		// Wait with generous timeout
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		result, err := process.WaitWithContext(ctx)
		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, 0, result.ExitCode)
		assert.Contains(t, string(result.Output), "fast")
	})
}

func TestProcess_ResourceLimits(t *testing.T) {
	logger := testutil.TestLogger(t)

	t.Run("set resource limits", func(t *testing.T) {
		cmd := exec.Command("echo", "test")
		process := NewProcess(cmd, logger)

		limits := &ResourceLimits{
			MaxMemoryMB:    256,
			MaxCPUPercent:  50.0,
			MaxDiskUsageMB: 100,
			MaxDuration:    time.Minute,
		}

		process.SetResourceLimits(limits)

		// Start process (resource limits would be applied)
		err := process.Start()
		assert.NoError(t, err)

		result, err := process.Wait()
		assert.NoError(t, err)
		assert.NotNil(t, result)

		// Resource limit enforcement would be tested in integration tests
		// with actual resource monitoring
	})
}

func TestProcess_Timing(t *testing.T) {
	logger := testutil.TestLogger(t)

	t.Run("process timing", func(t *testing.T) {
		cmd := exec.Command("sleep", "0.2") // 200ms sleep
		process := NewProcess(cmd, logger)

		startTime := time.Now()

		err := process.Start()
		require.NoError(t, err)

		// Check start time
		assert.True(t, process.GetStartTime().After(startTime))
		assert.True(t, process.GetStartTime().Before(time.Now()))

		// Initially, duration should be minimal
		initialDuration := process.GetDuration()
		assert.Greater(t, initialDuration, time.Duration(0))
		assert.Less(t, initialDuration, 50*time.Millisecond)

		result, err := process.Wait()
		assert.NoError(t, err)

		// Final duration should be at least the sleep time
		finalDuration := process.GetDuration()
		assert.GreaterOrEqual(t, finalDuration, 150*time.Millisecond)
		assert.Less(t, finalDuration, 500*time.Millisecond) // Allow some overhead

		assert.Equal(t, finalDuration, result.Duration)
	})

	t.Run("timing for non-started process", func(t *testing.T) {
		cmd := exec.Command("echo", "test")
		process := NewProcess(cmd, logger)

		assert.True(t, process.GetStartTime().IsZero())
		assert.Equal(t, time.Duration(0), process.GetDuration())
	})
}

func TestProcess_Close(t *testing.T) {
	logger := testutil.TestLogger(t)

	t.Run("close running process", func(t *testing.T) {
		cmd := exec.Command("sleep", "10")
		process := NewProcess(cmd, logger)

		err := process.Start()
		require.NoError(t, err)

		assert.True(t, process.IsRunning())

		err = process.Close()
		assert.NoError(t, err)

		// Process should be killed and cleaned up
		testutil.WaitForCondition(t, time.Second*2, func() bool {
			return process.IsFinished()
		})

		assert.True(t, process.IsFinished())
	})

	t.Run("close finished process", func(t *testing.T) {
		cmd := exec.Command("echo", "test")
		process := NewProcess(cmd, logger)

		err := process.Start()
		require.NoError(t, err)

		result, err := process.Wait()
		assert.NoError(t, err)
		assert.True(t, process.IsFinished())

		err = process.Close()
		assert.NoError(t, err)
	})

	t.Run("close with subscribers", func(t *testing.T) {
		cmd := exec.Command("echo", "test")
		process := NewProcess(cmd, logger)

		// Create multiple subscribers
		sub1 := process.SubscribeToOutput()
		sub2 := process.SubscribeToOutput()

		err := process.Start()
		require.NoError(t, err)

		result, err := process.Wait()
		assert.NoError(t, err)

		err = process.Close()
		assert.NoError(t, err)

		// Subscriber channels should be closed
		_, ok1 := <-sub1
		_, ok2 := <-sub2
		assert.False(t, ok1)
		assert.False(t, ok2)
	})
}

func TestProcess_MultipleSubscribers(t *testing.T) {
	logger := testutil.TestLogger(t)

	t.Run("multiple output subscribers", func(t *testing.T) {
		cmd := exec.Command("sh", "-c", "for i in 1 2 3; do echo \"Message $i\"; sleep 0.1; done")
		process := NewProcess(cmd, logger)

		// Create multiple subscribers
		sub1 := process.SubscribeToOutput()
		sub2 := process.SubscribeToOutput()
		sub3 := process.SubscribeToOutput()

		err := process.Start()
		require.NoError(t, err)

		// Collect from all subscribers
		var messages1, messages2, messages3 []string

		done := make(chan struct{}, 3)

		go func() {
			for output := range sub1 {
				if len(output) > 0 {
					messages1 = append(messages1, strings.TrimSpace(string(output)))
				}
			}
			done <- struct{}{}
		}()

		go func() {
			for output := range sub2 {
				if len(output) > 0 {
					messages2 = append(messages2, strings.TrimSpace(string(output)))
				}
			}
			done <- struct{}{}
		}()

		go func() {
			for output := range sub3 {
				if len(output) > 0 {
					messages3 = append(messages3, strings.TrimSpace(string(output)))
				}
			}
			done <- struct{}{}
		}()

		result, err := process.Wait()
		assert.NoError(t, err)

		// Wait for all subscribers to finish
		for i := 0; i < 3; i++ {
			<-done
		}

		// All subscribers should have received messages
		assert.Greater(t, len(messages1), 0)
		assert.Greater(t, len(messages2), 0)
		assert.Greater(t, len(messages3), 0)

		// Messages should be similar (though may vary due to timing)
		allMessages1 := strings.Join(messages1, " ")
		allMessages2 := strings.Join(messages2, " ")
		allMessages3 := strings.Join(messages3, " ")

		assert.Contains(t, allMessages1, "Message")
		assert.Contains(t, allMessages2, "Message")
		assert.Contains(t, allMessages3, "Message")
	})
}

func TestProcessState_String(t *testing.T) {
	tests := []struct {
		state    ProcessState
		expected string
	}{
		{ProcessStateCreated, "created"},
		{ProcessStateStarting, "starting"},
		{ProcessStateRunning, "running"},
		{ProcessStateFinished, "finished"},
		{ProcessStateKilled, "killed"},
		{ProcessStateError, "error"},
		{ProcessState(999), "unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.state.String())
		})
	}
}

func TestGenerateProcessID(t *testing.T) {
	// Generate multiple IDs and ensure they're unique
	ids := make(map[string]bool)

	for i := 0; i < 100; i++ {
		id := generateProcessID()
		assert.NotEmpty(t, id)
		assert.True(t, strings.HasPrefix(id, "claude-proc-"))

		// Should not have duplicates
		assert.False(t, ids[id], "Duplicate process ID generated: %s", id)
		ids[id] = true

		// Small delay to ensure different timestamps
		time.Sleep(time.Nanosecond)
	}
}

// Benchmark tests for process performance
func BenchmarkProcess_Start(b *testing.B) {
	logger := testutil.SilentLogger()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		cmd := exec.Command("echo", "bench")
		process := NewProcess(cmd, logger)

		err := process.Start()
		if err != nil {
			b.Fatal(err)
		}

		process.Wait()
		process.Close()
	}
}

func BenchmarkProcess_OutputHandling(b *testing.B) {
	logger := testutil.SilentLogger()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		cmd := exec.Command("sh", "-c", "for i in $(seq 1 100); do echo \"Line $i\"; done")
		process := NewProcess(cmd, logger)

		err := process.Start()
		if err != nil {
			b.Fatal(err)
		}

		result, err := process.Wait()
		if err != nil {
			b.Fatal(err)
		}

		// Access output to ensure it's processed
		_ = len(result.Output)

		process.Close()
	}
}

func BenchmarkProcess_MultipleSubscribers(b *testing.B) {
	logger := testutil.SilentLogger()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		cmd := exec.Command("sh", "-c", "for i in $(seq 1 10); do echo \"Message $i\"; done")
		process := NewProcess(cmd, logger)

		// Create multiple subscribers
		subs := make([]<-chan []byte, 5)
		for j := range subs {
			subs[j] = process.SubscribeToOutput()
		}

		err := process.Start()
		if err != nil {
			b.Fatal(err)
		}

		// Drain all subscribers
		for _, sub := range subs {
			go func(ch <-chan []byte) {
				for range ch {
					// Consume output
				}
			}(sub)
		}

		process.Wait()
		process.Close()
	}
}
