package engine

import (
	"errors"
	"sync"
	"time"

	"go.uber.org/zap"
)

// Pool manages multiple chess engines
type Pool struct {
	engines    map[string]*UCIEngine
	available  chan string // IDs of available engines
	maxEngines int         // Maximum number of engine to create
	enginePath string      // Path to the engine executable
	mu         sync.RWMutex
	logger     *zap.Logger
}

// NewEnginePool creates a new engine pool
func NewEnginePool(enginePath string, maxEngines int, logger *zap.Logger) *Pool {
	return &Pool{
		engines:    make(map[string]*UCIEngine),
		available:  make(chan string, maxEngines),
		maxEngines: maxEngines,
		enginePath: enginePath,
		logger:     logger,
	}
}

// Initialize creates the initial pool of engines
func (p *Pool) Initialize() error {
	p.mu.Lock()
	defer p.mu.Unlock()

	for i := 0; i < p.maxEngines; i++ {
		engine, err := NewUCIEngine(p.enginePath, p.logger)
		if err != nil {
			return err
		}

		p.engines[engine.ID.String()] = engine
		p.available <- engine.ID.String()
	}

	p.logger.Info("Engine pool initialized", zap.Int("count", len(p.engines)))
	return nil
}

// GetEngine retrieves an available engine from the pool with timeout
func (p *Pool) GetEngine() (*UCIEngine, error) {
	// Try to get an available engine with a timeout
	select {
	case engineID := <-p.available:
		p.mu.RLock()
		engine, exists := p.engines[engineID]
		p.mu.RUnlock()

		if !exists {
			return nil, errors.New("invalid engine ID from pool")
		}

		p.logger.Debug("Engine retrieved from pool", zap.String("engine_id", engineID))
		return engine, nil

	case <-time.After(5 * time.Second):
		return nil, errors.New("no engines available in the pool")
	}
}

// GetEngineByID retrieves a specific engine by ID
func (p *Pool) GetEngineByID(engineID string) (*UCIEngine, error) {
	p.mu.RLock()
	engine, exists := p.engines[engineID]
	p.mu.RUnlock()

	if !exists {
		return nil, errors.New("engine not found")
	}

	return engine, nil
}

// ReturnEngine returns an engine to the pool
func (p *Pool) ReturnEngine(engineID string) {
	p.mu.RLock()
	_, exists := p.engines[engineID]
	p.mu.RUnlock()

	if exists {
		// Non-blocking send to available channel
		select {
		case p.available <- engineID:
			p.logger.Debug("Engine returned to pool", zap.String("engine_id", engineID))
		default:
			p.logger.Warn("Failed to return engine to pool, channel full",
				zap.String("engine_id", engineID))
		}
	}
}

// Shutdown closes all engines in the pool
func (p *Pool) Shutdown() {
	p.mu.Lock()
	defer p.mu.Unlock()

	for id, engine := range p.engines {
		if err := engine.Close(); err != nil {
			p.logger.Error("Error closing engine",
				zap.String("engine_id", id),
				zap.Error(err))
		}
	}

	close(p.available)
	p.engines = make(map[string]*UCIEngine)

	p.logger.Info("Engine pool shut down")
}

// ConfigureEngine applies configuration to a specific engine
func (p *Pool) ConfigureEngine(engineID string, options map[string]string) error {
	p.mu.RLock()
	engine, exists := p.engines[engineID]
	p.mu.RUnlock()

	if !exists {
		return errors.New("engine not found")
	}

	for name, value := range options {
		if err := engine.SetOption(name, value); err != nil {
			return err
		}
	}

	return nil
}
