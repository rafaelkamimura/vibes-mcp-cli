package claude

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"openai-cli/internal/app/testutil"
)

func TestNewExecutor(t *testing.T) {
	tempDir := testutil.TempDir(t)
	logger := testutil.TestLogger(t)
	
	// Create a mock claude executable
	claudePath := testutil.MockExecutable(t, tempDir, "claude", 
		"#!/bin/bash\nif [ \"$1\" = \"--version\" ]; then echo \"claude 1.0\"; exit 0; fi\necho \"mock claude\"")
	
	t.Run("valid executable", func(t *testing.T) {
		executor, err := NewExecutor(logger, claudePath)
		assert.NoError(t, err)
		assert.NotNil(t, executor)
		assert.Equal(t, claudePath, executor.claudePath)
		assert.NotNil(t, executor.resourceMon)
		assert.NotNil(t, executor.defaultOpts)
	})
	
	t.Run("invalid executable path", func(t *testing.T) {
		_, err := NewExecutor(logger, "/nonexistent/claude")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "validation failed")
	})
	
	t.Run("non-executable file", func(t *testing.T) {
		nonExecPath := testutil.CreateTestFile(t, tempDir, "notexec", "not executable")
		_, err := NewExecutor(logger, nonExecPath)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "not executable")
	})
	
	t.Run("empty path", func(t *testing.T) {
		_, err := NewExecutor(logger, "")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "empty")
	})
}

func TestExecutor_Execute(t *testing.T) {
	tempDir := testutil.TempDir(t)
	logger := testutil.TestLogger(t)
	
	// Create mock executables for different scenarios
	successScript := `#!/bin/bash
echo "Success output"
exit 0`
	
	failScript := `#!/bin/bash
echo "Error output" >&2
exit 1`
	
	timeoutScript := `#!/bin/bash
sleep 5
echo "Should not reach here"`
	
	inputScript := `#!/bin/bash
read input
echo "Received: $input"`
	
	successPath := testutil.MockExecutable(t, tempDir, "success", successScript)
	failPath := testutil.MockExecutable(t, tempDir, "fail", failScript)
	timeoutPath := testutil.MockExecutable(t, tempDir, "timeout", timeoutScript)
	inputPath := testutil.MockExecutable(t, tempDir, "input", inputScript)
	
	t.Run("successful execution", func(t *testing.T) {
		executor, err := NewExecutor(logger, successPath)
		require.NoError(t, err)
		
		ctx := context.Background()
		options := &CommandOptions{
			WorkingDir: tempDir,
			Timeout:    time.Second * 5,
		}
		
		result, err := executor.Execute(ctx, options)
		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, 0, result.ExitCode)
		assert.Contains(t, string(result.Output), "Success output")
		assert.Greater(t, result.Duration, time.Duration(0))
	})
	
	t.Run("failed execution", func(t *testing.T) {
		executor, err := NewExecutor(logger, failPath)
		require.NoError(t, err)
		
		ctx := context.Background()
		options := &CommandOptions{
			WorkingDir: tempDir,
			Timeout:    time.Second * 5,
		}
		
		result, err := executor.Execute(ctx, options)
		assert.Error(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, 1, result.ExitCode)
		assert.Contains(t, string(result.Output), "Error output")
	})
	
	t.Run("timeout execution", func(t *testing.T) {
		executor, err := NewExecutor(logger, timeoutPath)
		require.NoError(t, err)
		
		ctx := context.Background()
		options := &CommandOptions{
			WorkingDir: tempDir,
			Timeout:    time.Millisecond * 100, // Very short timeout
		}
		
		result, err := executor.Execute(ctx, options)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "context deadline exceeded")
		// Process should be killed due to timeout
	})
	
	t.Run("execution with input", func(t *testing.T) {
		executor, err := NewExecutor(logger, inputPath)
		require.NoError(t, err)
		
		ctx := context.Background()
		options := &CommandOptions{
			WorkingDir: tempDir,
			Timeout:    time.Second * 5,
			InputText:  "test input\n",
		}
		
		result, err := executor.Execute(ctx, options)
		assert.NoError(t, err)
		assert.Equal(t, 0, result.ExitCode)
		assert.Contains(t, string(result.Output), "Received: test input")
	})
	
	t.Run("execution with environment variables", func(t *testing.T) {
		envScript := `#!/bin/bash
echo "TEST_VAR=$TEST_VAR"`
		envPath := testutil.MockExecutable(t, tempDir, "env", envScript)
		
		executor, err := NewExecutor(logger, envPath)
		require.NoError(t, err)
		
		ctx := context.Background()
		options := &CommandOptions{
			WorkingDir: tempDir,
			Timeout:    time.Second * 5,
			Environment: map[string]string{
				"TEST_VAR": "test_value",
			},
		}
		
		result, err := executor.Execute(ctx, options)
		assert.NoError(t, err)
		assert.Contains(t, string(result.Output), "TEST_VAR=test_value")
	})
	
	t.Run("execution with working directory", func(t *testing.T) {
		pwdScript := `#!/bin/bash
pwd`
		pwdPath := testutil.MockExecutable(t, tempDir, "pwd", pwdScript)
		
		workDir := testutil.CreateTestDir(t, tempDir, "workdir")
		
		executor, err := NewExecutor(logger, pwdPath)
		require.NoError(t, err)
		
		ctx := context.Background()
		options := &CommandOptions{
			WorkingDir: workDir,
			Timeout:    time.Second * 5,
		}
		
		result, err := executor.Execute(ctx, options)
		assert.NoError(t, err)
		assert.Contains(t, string(result.Output), workDir)
	})
}

func TestExecutor_ExecuteAsync(t *testing.T) {
	tempDir := testutil.TempDir(t)
	logger := testutil.TestLogger(t)
	
	longRunningScript := `#!/bin/bash
for i in {1..3}; do
    echo "Output $i"
    sleep 0.1
done`
	
	scriptPath := testutil.MockExecutable(t, tempDir, "longrunning", longRunningScript)
	
	executor, err := NewExecutor(logger, scriptPath)
	require.NoError(t, err)
	
	t.Run("async execution", func(t *testing.T) {
		ctx := context.Background()
		options := &CommandOptions{
			WorkingDir: tempDir,
			Timeout:    time.Second * 5,
		}
		
		process, err := executor.ExecuteAsync(ctx, options)
		assert.NoError(t, err)
		assert.NotNil(t, process)
		assert.True(t, process.IsRunning())
		
		// Wait for completion
		result, err := process.Wait()
		assert.NoError(t, err)
		assert.Equal(t, 0, result.ExitCode)
		assert.Contains(t, string(result.Output), "Output 1")
		assert.Contains(t, string(result.Output), "Output 2")
		assert.Contains(t, string(result.Output), "Output 3")
	})
}

func TestExecutor_ProcessManagement(t *testing.T) {
	tempDir := testutil.TempDir(t)
	logger := testutil.TestLogger(t)
	
	longRunningScript := `#!/bin/bash
for i in {1..100}; do
    echo "Running $i"
    sleep 0.1
done`
	
	scriptPath := testutil.MockExecutable(t, tempDir, "longrunning", longRunningScript)
	
	executor, err := NewExecutor(logger, scriptPath)
	require.NoError(t, err)
	
	t.Run("get active processes", func(t *testing.T) {
		ctx := context.Background()
		options := &CommandOptions{
			WorkingDir: tempDir,
			Timeout:    time.Second * 30,
		}
		
		// Start multiple processes
		process1, err := executor.ExecuteAsync(ctx, options)
		require.NoError(t, err)
		
		process2, err := executor.ExecuteAsync(ctx, options)
		require.NoError(t, err)
		
		// Check active processes
		activeProcs := executor.GetActiveProcesses()
		assert.GreaterOrEqual(t, len(activeProcs), 2)
		
		// Clean up
		process1.Kill()
		process2.Kill()
	})
	
	t.Run("kill specific process", func(t *testing.T) {
		ctx := context.Background()
		options := &CommandOptions{
			WorkingDir: tempDir,
			Timeout:    time.Second * 30,
		}
		
		process, err := executor.ExecuteAsync(ctx, options)
		require.NoError(t, err)
		
		processID := process.ID
		assert.True(t, process.IsRunning())
		
		// Kill the process
		err = executor.KillProcess(processID)
		assert.NoError(t, err)
		
		// Wait a bit for the process to be killed
		testutil.WaitForCondition(t, time.Second*2, func() bool {
			return !process.IsRunning()
		})
		
		assert.True(t, process.IsFinished())
	})
	
	t.Run("kill all processes", func(t *testing.T) {
		ctx := context.Background()
		options := &CommandOptions{
			WorkingDir: tempDir,
			Timeout:    time.Second * 30,
		}
		
		// Start multiple processes
		process1, err := executor.ExecuteAsync(ctx, options)
		require.NoError(t, err)
		
		process2, err := executor.ExecuteAsync(ctx, options)
		require.NoError(t, err)
		
		// Kill all processes
		err = executor.KillAllProcesses()
		assert.NoError(t, err)
		
		// Wait for processes to be killed
		testutil.WaitForCondition(t, time.Second*2, func() bool {
			return process1.IsFinished() && process2.IsFinished()
		})
		
		assert.True(t, process1.IsFinished())
		assert.True(t, process2.IsFinished())
	})
	
	t.Run("send input to process", func(t *testing.T) {
		inputScript := `#!/bin/bash
while read line; do
    echo "Received: $line"
done`
		
		inputPath := testutil.MockExecutable(t, tempDir, "input", inputScript)
		
		inputExecutor, err := NewExecutor(logger, inputPath)
		require.NoError(t, err)
		
		ctx := context.Background()
		options := &CommandOptions{
			WorkingDir: tempDir,
			Timeout:    time.Second * 5,
		}
		
		process, err := inputExecutor.ExecuteAsync(ctx, options)
		require.NoError(t, err)
		
		// Send input
		err = inputExecutor.SendInputToProcess(process.ID, "test message\n")
		assert.NoError(t, err)
		
		// Give some time for processing
		time.Sleep(100 * time.Millisecond)
		
		// Close input to end the process
		process.CloseInput()
		
		result, err := process.Wait()
		assert.NoError(t, err)
		assert.Contains(t, string(result.Output), "Received: test message")
	})
	
	t.Run("get process output", func(t *testing.T) {
		outputScript := `#!/bin/bash
echo "Line 1"
sleep 0.1
echo "Line 2"
sleep 0.1
echo "Line 3"`
		
		outputPath := testutil.MockExecutable(t, tempDir, "output", outputScript)
		
		outputExecutor, err := NewExecutor(logger, outputPath)
		require.NoError(t, err)
		
		ctx := context.Background()
		options := &CommandOptions{
			WorkingDir: tempDir,
			Timeout:    time.Second * 5,
		}
		
		process, err := outputExecutor.ExecuteAsync(ctx, options)
		require.NoError(t, err)
		
		// Wait a bit for some output
		time.Sleep(200 * time.Millisecond)
		
		// Get accumulated output
		output, err := outputExecutor.GetProcessOutput(process.ID)
		assert.NoError(t, err)
		assert.NotEmpty(t, output)
		
		// Wait for completion
		process.Wait()
		
		// Get final output
		finalOutput, err := outputExecutor.GetProcessOutput(process.ID)
		assert.NoError(t, err)
		assert.Contains(t, string(finalOutput), "Line 1")
		assert.Contains(t, string(finalOutput), "Line 2")
		assert.Contains(t, string(finalOutput), "Line 3")
	})
	
	t.Run("subscribe to output", func(t *testing.T) {
		outputScript := `#!/bin/bash
for i in {1..5}; do
    echo "Message $i"
    sleep 0.1
done`
		
		outputPath := testutil.MockExecutable(t, tempDir, "output", outputScript)
		
		outputExecutor, err := NewExecutor(logger, outputPath)
		require.NoError(t, err)
		
		ctx := context.Background()
		options := &CommandOptions{
			WorkingDir: tempDir,
			Timeout:    time.Second * 5,
		}
		
		process, err := outputExecutor.ExecuteAsync(ctx, options)
		require.NoError(t, err)
		
		// Subscribe to output
		outputChan, err := outputExecutor.SubscribeToOutput(process.ID)
		assert.NoError(t, err)
		assert.NotNil(t, outputChan)
		
		// Collect some output
		var messages []string
		timeout := time.After(2 * time.Second)
		
	collectLoop:
		for {
			select {
			case output, ok := <-outputChan:
				if !ok {
					break collectLoop
				}
				if len(output) > 0 {
					messages = append(messages, string(output))
				}
			case <-timeout:
				break collectLoop
			}
		}
		
		// Should have received some messages
		assert.Greater(t, len(messages), 0)
		
		// Wait for process to complete
		process.Wait()
	})
}

func TestExecutor_ResourceLimits(t *testing.T) {
	tempDir := testutil.TempDir(t)
	logger := testutil.TestLogger(t)
	
	scriptPath := testutil.MockExecutable(t, tempDir, "test", "#!/bin/bash\necho test")
	
	executor, err := NewExecutor(logger, scriptPath)
	require.NoError(t, err)
	
	t.Run("default resource limits", func(t *testing.T) {
		defaults := DefaultResourceLimits()
		assert.NotNil(t, defaults)
		assert.Greater(t, defaults.MaxMemoryMB, 0)
		assert.Greater(t, defaults.MaxCPUPercent, 0.0)
		assert.Greater(t, defaults.MaxDiskUsageMB, 0)
		assert.Greater(t, defaults.MaxDuration, time.Duration(0))
	})
	
	t.Run("custom resource limits", func(t *testing.T) {
		limits := &ResourceLimits{
			MaxMemoryMB:    512,
			MaxCPUPercent:  50.0,
			MaxDiskUsageMB: 256,
			MaxDuration:    time.Hour,
		}
		
		ctx := context.Background()
		options := &CommandOptions{
			WorkingDir:     tempDir,
			Timeout:        time.Second * 5,
			ResourceLimits: limits,
		}
		
		result, err := executor.Execute(ctx, options)
		assert.NoError(t, err)
		assert.NotNil(t, result)
		// Resource monitoring would be tested in integration tests
		// with actual resource usage measurement
	})
}

func TestExecutor_DefaultOptions(t *testing.T) {
	tempDir := testutil.TempDir(t)
	logger := testutil.TestLogger(t)
	
	scriptPath := testutil.MockExecutable(t, tempDir, "test", "#!/bin/bash\necho test")
	
	executor, err := NewExecutor(logger, scriptPath)
	require.NoError(t, err)
	
	t.Run("set and use default options", func(t *testing.T) {
		defaultOpts := &CommandOptions{
			WorkingDir: tempDir,
			Timeout:    time.Second * 10,
			Environment: map[string]string{
				"DEFAULT_VAR": "default_value",
			},
		}
		
		executor.SetDefaultOptions(defaultOpts)
		
		ctx := context.Background()
		
		// Execute without providing options (should use defaults)
		result, err := executor.Execute(ctx, nil)
		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, 0, result.ExitCode)
	})
	
	t.Run("override default options", func(t *testing.T) {
		defaultOpts := &CommandOptions{
			WorkingDir: tempDir,
			Timeout:    time.Second * 10,
			Environment: map[string]string{
				"DEFAULT_VAR": "default_value",
				"SHARED_VAR":  "default_shared",
			},
		}
		
		executor.SetDefaultOptions(defaultOpts)
		
		// Override some options
		overrideOpts := &CommandOptions{
			Timeout: time.Second * 5, // Different timeout
			Environment: map[string]string{
				"SHARED_VAR":   "override_shared", // Override existing var
				"OVERRIDE_VAR": "override_value",  // Add new var
			},
		}
		
		ctx := context.Background()
		result, err := executor.Execute(ctx, overrideOpts)
		assert.NoError(t, err)
		assert.NotNil(t, result)
		// Verification would require environment inspection in the script
	})
}

func TestExecutor_Close(t *testing.T) {
	tempDir := testutil.TempDir(t)
	logger := testutil.TestLogger(t)
	
	longRunningScript := `#!/bin/bash
for i in {1..100}; do
    echo "Running $i"
    sleep 0.1
done`
	
	scriptPath := testutil.MockExecutable(t, tempDir, "longrunning", longRunningScript)
	
	executor, err := NewExecutor(logger, scriptPath)
	require.NoError(t, err)
	
	t.Run("close with active processes", func(t *testing.T) {
		ctx := context.Background()
		options := &CommandOptions{
			WorkingDir: tempDir,
			Timeout:    time.Second * 30,
		}
		
		// Start multiple processes
		process1, err := executor.ExecuteAsync(ctx, options)
		require.NoError(t, err)
		
		process2, err := executor.ExecuteAsync(ctx, options)
		require.NoError(t, err)
		
		assert.True(t, process1.IsRunning())
		assert.True(t, process2.IsRunning())
		
		// Close executor (should kill all processes)
		err = executor.Close()
		assert.NoError(t, err)
		
		// Wait for processes to be killed
		testutil.WaitForCondition(t, time.Second*2, func() bool {
			return process1.IsFinished() && process2.IsFinished()
		})
		
		assert.True(t, process1.IsFinished())
		assert.True(t, process2.IsFinished())
	})
}

func TestExecutor_ErrorHandling(t *testing.T) {
	tempDir := testutil.TempDir(t)
	logger := testutil.TestLogger(t)
	
	scriptPath := testutil.MockExecutable(t, tempDir, "test", "#!/bin/bash\necho test")
	
	executor, err := NewExecutor(logger, scriptPath)
	require.NoError(t, err)
	
	t.Run("kill non-existent process", func(t *testing.T) {
		err := executor.KillProcess("non-existent-id")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "not found")
	})
	
	t.Run("send input to non-existent process", func(t *testing.T) {
		err := executor.SendInputToProcess("non-existent-id", "input")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "not found")
	})
	
	t.Run("get output from non-existent process", func(t *testing.T) {
		_, err := executor.GetProcessOutput("non-existent-id")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "not found")
	})
	
	t.Run("subscribe to non-existent process", func(t *testing.T) {
		_, err := executor.SubscribeToOutput("non-existent-id")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "not found")
	})
}

func TestExecutor_ConcurrentExecution(t *testing.T) {
	tempDir := testutil.TempDir(t)
	logger := testutil.TestLogger(t)
	
	scriptPath := testutil.MockExecutable(t, tempDir, "test", "#!/bin/bash\necho \"Process $$\"")
	
	executor, err := NewExecutor(logger, scriptPath)
	require.NoError(t, err)
	defer executor.Close()
	
	t.Run("concurrent execution", func(t *testing.T) {
		const numGoroutines = 10
		resultChan := make(chan error, numGoroutines)
		
		ctx := context.Background()
		options := &CommandOptions{
			WorkingDir: tempDir,
			Timeout:    time.Second * 5,
		}
		
		// Start multiple executions concurrently
		for i := 0; i < numGoroutines; i++ {
			go func(id int) {
				_, err := executor.Execute(ctx, options)
				resultChan <- err
			}(i)
		}
		
		// Collect results
		for i := 0; i < numGoroutines; i++ {
			err := <-resultChan
			assert.NoError(t, err)
		}
	})
}

// Test validation functions
func TestValidateClaudeExecutable(t *testing.T) {
	tempDir := testutil.TempDir(t)
	
	t.Run("valid executable", func(t *testing.T) {
		validScript := `#!/bin/bash
if [ "$1" = "--version" ]; then
    echo "claude 1.0"
    exit 0
fi
echo "claude command"`
		
		validPath := testutil.MockExecutable(t, tempDir, "valid", validScript)
		err := validateClaudeExecutable(validPath)
		assert.NoError(t, err)
	})
	
	t.Run("empty path", func(t *testing.T) {
		err := validateClaudeExecutable("")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "empty")
	})
	
	t.Run("non-existent file", func(t *testing.T) {
		err := validateClaudeExecutable("/nonexistent/path")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "not found")
	})
	
	t.Run("non-executable file", func(t *testing.T) {
		nonExecPath := testutil.CreateTestFile(t, tempDir, "notexec", "not executable")
		err := validateClaudeExecutable(nonExecPath)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "not executable")
	})
	
	t.Run("executable that fails version check", func(t *testing.T) {
		failingScript := `#!/bin/bash
exit 1`
		
		failingPath := testutil.MockExecutable(t, tempDir, "failing", failingScript)
		err := validateClaudeExecutable(failingPath)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "validation failed")
	})
}

// Benchmark tests for executor performance
func BenchmarkExecutor_Execute(b *testing.B) {
	tempDir := testutil.TempDir(b)
	logger := testutil.SilentLogger()
	
	scriptPath := testutil.MockExecutable(b, tempDir, "bench", "#!/bin/bash\necho bench")
	
	executor, err := NewExecutor(logger, scriptPath)
	if err != nil {
		b.Fatal(err)
	}
	defer executor.Close()
	
	ctx := context.Background()
	options := &CommandOptions{
		WorkingDir: tempDir,
		Timeout:    time.Second * 5,
	}
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := executor.Execute(ctx, options)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkExecutor_ExecuteAsync(b *testing.B) {
	tempDir := testutil.TempDir(b)
	logger := testutil.SilentLogger()
	
	scriptPath := testutil.MockExecutable(b, tempDir, "bench", "#!/bin/bash\necho bench")
	
	executor, err := NewExecutor(logger, scriptPath)
	if err != nil {
		b.Fatal(err)
	}
	defer executor.Close()
	
	ctx := context.Background()
	options := &CommandOptions{
		WorkingDir: tempDir,
		Timeout:    time.Second * 5,
	}
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		process, err := executor.ExecuteAsync(ctx, options)
		if err != nil {
			b.Fatal(err)
		}
		process.Wait()
	}
}