package engine

import (
	"bufio"
	"fmt"
	"io"
	"log"
	"os/exec"
	"strings"
	"sync"

	"github.com/google/uuid"
	"go.uber.org/zap"
)

// UCIEngine represents a UCI-compatible chess engine
type UCIEngine struct {
	ID uuid.UUID

	cmd *exec.Cmd

	stdinPipe  io.WriteCloser
	stdoutPipe io.ReadCloser
	reader     *bufio.Reader

	mutex        sync.Mutex
	quitChan     chan struct{}
	BestMoveChan chan string

	logger *zap.Logger
}

// NewUCIEngine starts the engine process and returns a UCIEngine instance.
// enginePath is the path yo tou engine executable (e.g. "argo")
func NewUCIEngine(enginePath string, logger *zap.Logger) (*UCIEngine, error) {
	cmd := exec.Command(enginePath)

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, fmt.Errorf("StdoutPipe error: %w", err)
	}

	stdin, err := cmd.StdinPipe()
	if err != nil {
		return nil, fmt.Errorf("StdinPipe error: %w", err)
	}

	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("error starting engine: %w", err)
	}

	e := &UCIEngine{
		cmd:          cmd,
		stdinPipe:    stdin,
		stdoutPipe:   stdout,
		reader:       bufio.NewReader(stdout),
		quitChan:     make(chan struct{}),
		BestMoveChan: make(chan string, 1),
		logger:       logger,
	}

	// Initialize UCI mode
	if err := e.writeCommand("uci"); err != nil {
		return nil, fmt.Errorf("error sending uci cmd: %w", err)
	}

	// Some engines print info on startup; you might need to read until you see "uciok"
	go e.readLoop()

	return e, nil
}

func (e *UCIEngine) readLoop() {
	for {
		select {
		case <-e.quitChan:
			return
		default:
			line, err := e.reader.ReadString('\n')
			if err != nil {
				if err == io.EOF {
					log.Println("Engine closed stdout.")
				} else {
					log.Println("Error reading engine output:", err)
				}
				return
			}
			line = strings.TrimSpace(line)

			log.Println("ENGINE>", line)

			// Check if the engine sent a best move.
			if strings.HasPrefix(line, "bestmove") {
				fields := strings.Fields(line)
				if len(fields) >= 2 {
					bestMove := fields[1]
					// Send bestMove into the channel without blocking.
					select {
					case e.BestMoveChan <- bestMove:
					default:
					}
				}
			}

		}
	}
}

func (e *UCIEngine) writeCommand(cmd string) error {
	e.mutex.Lock()
	defer e.mutex.Unlock()

	_, err := io.WriteString(e.stdinPipe, cmd+"\n")
	return err
}

func (e *UCIEngine) Close() error {
	close(e.quitChan)
	_ = e.writeCommand("quit")
	if err := e.cmd.Wait(); err != nil {
		return err
	}
	return nil
}

func (e *UCIEngine) SendCommand(cmd string) error {
	err := e.writeCommand(cmd)
	if err != nil {
		return err
	}

	return nil
}

func (e *UCIEngine) SetOption(name, value string) error {
	return nil
}
