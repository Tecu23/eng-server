// Package chess defines the game entities
package chess

// TODO: Handle different timing methods
// TODO: Handle classical time controls where after some 40 moves
//		the time increments by a specified amount

import (
	"fmt"
	"sync"
	"time"
)

// TimeControl defines the time settings for a game
type TimeControl struct {
	WhiteTime       int64 // Initial time in milliseconds
	BlackTime       int64
	WhiteIncrement  int64 // Increment per move in milliseconds
	BlackIncrement  int64
	TimingMethod    TimingMethod // Increment, Delay, or Bronstein
	MovesPerControl int          // For classical time controls (e.g., 40 moves in 2 hours)
}

// TimingMethod defines the different ways to time a chess game
type TimingMethod int

// All the possible timing methods that will be implemented
const (
	IncrementTiming TimingMethod = iota
	DelayTiming
	BronsteinTiming
)

// Clock manages the chess clock for both players
type Clock struct {
	whiteTimeMs int64
	blackTimeMs int64

	whiteIncrement int64
	blackIncrement int64

	activeColor Color

	timingMethod TimingMethod

	movesPerControl int
	moveCount       int

	startTime time.Time
	isRunning bool

	// delay fields for the DelayTiming method
	delayStartTime time.Time
	delayRemaining int64

	mutex sync.RWMutex

	// For external events
	timeupChan chan Color
	tickChan   chan ClockTick
}

// ClockTick defines a single clock tick
type ClockTick struct {
	White       int64
	Black       int64
	ActiveColor Color
}

// NewClock creates a new chess clock with the given time controls
func NewClock(tc TimeControl) *Clock {
	return &Clock{
		whiteTimeMs:     tc.WhiteTime,
		blackTimeMs:     tc.BlackTime,
		whiteIncrement:  tc.WhiteIncrement,
		blackIncrement:  tc.BlackIncrement,
		activeColor:     White,
		timingMethod:    tc.TimingMethod,
		movesPerControl: tc.MovesPerControl,
		timeupChan:      make(chan Color, 1),
		tickChan:        make(chan ClockTick, 10),
	}
}

// Start starts the clock for the current player
func (c *Clock) Start() {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	if c.isRunning {
		return
	}

	c.startTime = time.Now()
	c.isRunning = true

	go c.tickRoutine()
}

// Stop stops the clock
func (c *Clock) Stop() {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	if !c.isRunning {
		return
	}

	c.updateTime()
	c.isRunning = false
}

// Switch switches the active player and handles time increments
func (c *Clock) Switch() {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	if c.isRunning {
		c.updateTime()
	}

	if c.timingMethod == IncrementTiming {
		if c.activeColor == White {
			c.whiteTimeMs += c.whiteIncrement
		} else {
			c.blackIncrement += c.blackIncrement
		}
	}

	c.activeColor = c.activeColor.Opp()

	if c.activeColor == White {
		c.moveCount++
	}

	if c.isRunning {
		c.startTime = time.Now()
	}
}

// updateTime updates the time based on elapsed time
func (c *Clock) updateTime() {
	elapsed := time.Since(c.startTime).Milliseconds()

	if c.activeColor == White {
		c.whiteTimeMs -= elapsed
	} else {
		c.blackTimeMs -= elapsed
	}

	if (c.activeColor == White && c.whiteTimeMs <= 0) ||
		(c.activeColor == Black && c.blackTimeMs <= 0) {
		select {
		case c.timeupChan <- c.activeColor:
		default:
			// Channel buffer is full
		}

		if c.activeColor == White {
			c.whiteTimeMs = 0
		} else {
			c.blackTimeMs = 0
		}

		c.isRunning = false
	}
}

// GetRemainingTime returns the current remaining time for both players
func (c *Clock) GetRemainingTime() struct{ White, Black int64 } {
	c.mutex.RLock()
	defer c.mutex.RUnlock()

	whiteTime := c.whiteTimeMs
	blackTime := c.blackTimeMs

	// If clock is running, calculate current time
	if c.isRunning {
		elapsed := time.Since(c.startTime).Milliseconds()

		if c.activeColor == White {
			whiteTime -= elapsed
		} else {
			blackTime -= elapsed
		}
	}

	// Ensure times don't go negative
	if whiteTime < 0 {
		whiteTime = 0
	}
	if blackTime < 0 {
		blackTime = 0
	}

	return struct{ White, Black int64 }{whiteTime, blackTime}
}

// IsTimeUp checks if a player has run out of time
func (c *Clock) IsTimeUp(color Color) bool {
	c.mutex.RLock()
	defer c.mutex.RUnlock()

	if color == White {
		return c.whiteTimeMs <= 0
	}
	return c.blackTimeMs <= 0
}

// GetTimeupChannel returns a channel that signals when time is up
func (c *Clock) GetTimeupChannel() <-chan Color {
	return c.timeupChan
}

// GetTickChannel returns a channel that provides periodic clock updates
func (c *Clock) GetTickChannel() <-chan ClockTick {
	return c.tickChan
}

// TickRoutine sends periodic updates of the clock state
func (c *Clock) tickRoutine() {
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	for range ticker.C {
		c.mutex.RLock()
		if !c.isRunning {
			c.mutex.RUnlock()
			return
		}

		times := c.GetRemainingTime()
		tick := ClockTick{
			White:       times.White,
			Black:       times.Black,
			ActiveColor: c.activeColor,
		}
		c.mutex.RUnlock()

		// Send tick update
		select {
		case c.tickChan <- tick:
		default:
			// Channel buffer is full
		}
	}
}

// FormatClockTime formats a duration in milliseconds to a user-friendly string (e.g., "1:30")
func FormatClockTime(timeMs int64) string {
	if timeMs < 0 {
		timeMs = 0
	}

	totalSeconds := timeMs / 1000
	minutes := totalSeconds / 60
	seconds := totalSeconds % 60

	// For times less than 10 seconds, show decimal
	if timeMs < 10000 {
		tenths := (timeMs % 1000) / 100
		return fmt.Sprintf("%d.%d", totalSeconds, tenths)
	}

	return fmt.Sprintf("%d:%02d", minutes, seconds)
}
