package instance

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os/exec"
	"strings"
	"sync"
	"time"

	"cs2admin/internal/pkg/logger"
)

// Process manages a single CS2 server process.
type Process struct {
	cmd        *exec.Cmd
	pid        int
	running    bool
	mu         sync.Mutex
	stdin      io.WriteCloser
	stdout     io.ReadCloser
	onOutput   func(line string)
	onExit     func(exitCode int)
	cancelFunc context.CancelFunc
}

// NewProcess creates a new Process for the given executable path and arguments.
func NewProcess(execPath string, args []string) *Process {
	return &Process{
		cmd:      exec.Command(execPath, args...),
		onOutput: func(string) {},
		onExit:   func(int) {},
	}
}

// Start starts the process, pipes stdout/stderr, spawns goroutines to read output
// and wait for exit, calling the respective callbacks.
func (p *Process) Start() error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.running {
		return fmt.Errorf("process already running")
	}

	_, cancel := context.WithCancel(context.Background())
	p.cancelFunc = cancel

	hideWindow(p.cmd) // prevent visible console window on Windows

	stdinPipe, err := p.cmd.StdinPipe()
	if err != nil {
		cancel()
		return fmt.Errorf("stdin pipe: %w", err)
	}
	p.stdin = stdinPipe

	stdoutPipe, err := p.cmd.StdoutPipe()
	if err != nil {
		stdinPipe.Close()
		cancel()
		return fmt.Errorf("stdout pipe: %w", err)
	}
	p.stdout = stdoutPipe

	stderrPipe, err := p.cmd.StderrPipe()
	if err != nil {
		stdoutPipe.Close()
		cancel()
		return fmt.Errorf("stderr pipe: %w", err)
	}

	if err := p.cmd.Start(); err != nil {
		stdinPipe.Close()
		stdoutPipe.Close()
		stderrPipe.Close()
		cancel()
		return fmt.Errorf("start: %w", err)
	}

	p.pid = p.cmd.Process.Pid
	p.running = true

	onOutput := p.onOutput
	if onOutput == nil {
		onOutput = func(string) {}
	}
	onExit := p.onExit
	if onExit == nil {
		onExit = func(int) {}
	}

	// Read stdout line by line
	go func() {
		reader := bufio.NewReader(stdoutPipe)
		for {
			line, err := reader.ReadString('\n')
			if line != "" {
				line = strings.TrimRight(line, "\r\n")
				if line != "" {
					onOutput(line)
				}
			}
			if err != nil {
				if err != io.EOF {
					logger.Log.Warn().Err(err).Msg("stdout read error")
				}
				break
			}
		}
		_ = stdoutPipe.Close()
	}()

	// Read stderr line by line
	go func() {
		reader := bufio.NewReader(stderrPipe)
		for {
			line, err := reader.ReadString('\n')
			if line != "" {
				line = strings.TrimRight(line, "\r\n")
				if line != "" {
					onOutput(line)
				}
			}
			if err != nil {
				if err != io.EOF {
					logger.Log.Warn().Err(err).Msg("stderr read error")
				}
				break
			}
		}
		_ = stderrPipe.Close()
	}()

	// Wait for process exit
	go func() {
		err := p.cmd.Wait()
		exitCode := 0
		if err != nil {
			if exitErr, ok := err.(*exec.ExitError); ok {
				exitCode = exitErr.ExitCode()
			} else {
				exitCode = -1
			}
		}

		p.mu.Lock()
		p.running = false
		p.pid = 0
		p.mu.Unlock()

		onExit(exitCode)
	}()

	return nil
}

// Stop performs a graceful stop: sends "quit" to stdin, waits up to 5s, then kills.
func (p *Process) Stop() error {
	p.mu.Lock()
	if !p.running {
		p.mu.Unlock()
		return nil
	}
	stdin := p.stdin
	proc := p.cmd.Process
	p.mu.Unlock()

	if stdin != nil {
		_, _ = io.WriteString(stdin, "quit\n")
		_ = stdin.Close()
	}

	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		p.mu.Lock()
		stillRunning := p.running
		p.mu.Unlock()
		if !stillRunning {
			return nil
		}
		time.Sleep(100 * time.Millisecond)
	}

	p.mu.Lock()
	if p.running && proc != nil {
		_ = proc.Kill()
	}
	p.mu.Unlock()

	// Give the process a moment to exit
	time.Sleep(200 * time.Millisecond)
	return nil
}

// Kill force-kills the process immediately.
func (p *Process) Kill() error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if !p.running || p.cmd == nil || p.cmd.Process == nil {
		return nil
	}

	if p.cancelFunc != nil {
		p.cancelFunc()
	}

	return p.cmd.Process.Kill()
}

// IsRunning returns true if the process is running.
func (p *Process) IsRunning() bool {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.running
}

// PID returns the process ID, or 0 if not running.
func (p *Process) PID() int {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.pid
}

// SetOnOutput sets the callback invoked for each stdout/stderr line.
func (p *Process) SetOnOutput(fn func(string)) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if fn != nil {
		p.onOutput = fn
	} else {
		p.onOutput = func(string) {}
	}
}

// SetOnExit sets the callback invoked when the process exits.
func (p *Process) SetOnExit(fn func(int)) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if fn != nil {
		p.onExit = fn
	} else {
		p.onExit = func(int) {}
	}
}
