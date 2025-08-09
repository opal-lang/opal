package decorators

import (
	"context"
	"fmt"
	"os/exec"
	"sync"
	"time"
)

// CommandConnection represents a reusable command execution context
type CommandConnection struct {
	ID          string
	CreatedAt   time.Time
	LastUsedAt  time.Time
	UsageCount  int64
	WorkingDir  string
	Environment []string
	IsIdle      bool
	ctx         context.Context
	cancel      context.CancelFunc
	mu          sync.RWMutex
}

// NewCommandConnection creates a new command connection
func NewCommandConnection(id, workingDir string, env []string) *CommandConnection {
	ctx, cancel := context.WithCancel(context.Background())

	return &CommandConnection{
		ID:          id,
		CreatedAt:   time.Now(),
		LastUsedAt:  time.Now(),
		UsageCount:  0,
		WorkingDir:  workingDir,
		Environment: env,
		IsIdle:      true,
		ctx:         ctx,
		cancel:      cancel,
	}
}

// Execute runs a command using this connection
func (cc *CommandConnection) Execute(command string, args ...string) (*exec.Cmd, error) {
	cc.mu.Lock()
	defer cc.mu.Unlock()

	if cc.ctx.Err() != nil {
		return nil, fmt.Errorf("connection is closed")
	}

	cc.IsIdle = false
	cc.LastUsedAt = time.Now()
	cc.UsageCount++

	cmd := exec.CommandContext(cc.ctx, command, args...)
	cmd.Dir = cc.WorkingDir
	cmd.Env = cc.Environment

	return cmd, nil
}

// MarkIdle marks the connection as idle and available for reuse
func (cc *CommandConnection) MarkIdle() {
	cc.mu.Lock()
	defer cc.mu.Unlock()
	cc.IsIdle = true
	cc.LastUsedAt = time.Now()
}

// Close closes the connection and cancels any running commands
func (cc *CommandConnection) Close() {
	cc.mu.Lock()
	defer cc.mu.Unlock()

	if cc.cancel != nil {
		cc.cancel()
	}
}

// IsExpired checks if the connection has expired based on idle time
func (cc *CommandConnection) IsExpired(maxIdleTime time.Duration) bool {
	cc.mu.RLock()
	defer cc.mu.RUnlock()

	return cc.IsIdle && time.Since(cc.LastUsedAt) > maxIdleTime
}

// ConnectionPool manages a pool of reusable command connections
type ConnectionPool struct {
	mu              sync.RWMutex
	connections     map[string]*CommandConnection
	maxConnections  int
	maxIdleTime     time.Duration
	cleanupInterval time.Duration
	nextID          int64
	cleanupTicker   *time.Ticker
	cleanupDone     chan bool
}

// NewConnectionPool creates a new connection pool
func NewConnectionPool(maxConnections int, maxIdleTime, cleanupInterval time.Duration) *ConnectionPool {
	pool := &ConnectionPool{
		connections:     make(map[string]*CommandConnection),
		maxConnections:  maxConnections,
		maxIdleTime:     maxIdleTime,
		cleanupInterval: cleanupInterval,
		nextID:          0,
		cleanupTicker:   time.NewTicker(cleanupInterval),
		cleanupDone:     make(chan bool),
	}

	// Start cleanup routine
	go pool.cleanupRoutine()

	return pool
}

// GetConnection gets or creates a connection for the specified working directory and environment
func (cp *ConnectionPool) GetConnection(workingDir string, env []string) (*CommandConnection, error) {
	cp.mu.Lock()
	defer cp.mu.Unlock()

	// Try to find an existing idle connection with matching context
	for _, conn := range cp.connections {
		if conn.IsIdle && conn.WorkingDir == workingDir && envEqual(conn.Environment, env) {
			return conn, nil
		}
	}

	// Check if we can create a new connection
	if len(cp.connections) >= cp.maxConnections {
		// Try to evict an expired connection
		if !cp.evictExpiredConnection() {
			return nil, fmt.Errorf("connection pool is full and no connections can be evicted")
		}
	}

	// Create new connection
	cp.nextID++
	id := fmt.Sprintf("conn_%d", cp.nextID)
	conn := NewCommandConnection(id, workingDir, env)
	cp.connections[id] = conn

	return conn, nil
}

// ReturnConnection returns a connection to the pool
func (cp *ConnectionPool) ReturnConnection(conn *CommandConnection) {
	conn.MarkIdle()
}

// CloseConnection closes and removes a connection from the pool
func (cp *ConnectionPool) CloseConnection(conn *CommandConnection) {
	cp.mu.Lock()
	defer cp.mu.Unlock()

	conn.Close()
	delete(cp.connections, conn.ID)
}

// Close closes all connections and stops the cleanup routine
func (cp *ConnectionPool) Close() {
	cp.mu.Lock()
	defer cp.mu.Unlock()

	// Stop cleanup routine
	if cp.cleanupTicker != nil {
		cp.cleanupTicker.Stop()
		close(cp.cleanupDone)
	}

	// Close all connections
	for _, conn := range cp.connections {
		conn.Close()
	}
	cp.connections = make(map[string]*CommandConnection)
}

// GetStats returns pool statistics
func (cp *ConnectionPool) GetStats() map[string]interface{} {
	cp.mu.RLock()
	defer cp.mu.RUnlock()

	idleCount := 0
	activeCount := 0
	totalUsage := int64(0)

	for _, conn := range cp.connections {
		if conn.IsIdle {
			idleCount++
		} else {
			activeCount++
		}
		totalUsage += conn.UsageCount
	}

	return map[string]interface{}{
		"total_connections":  len(cp.connections),
		"idle_connections":   idleCount,
		"active_connections": activeCount,
		"max_connections":    cp.maxConnections,
		"total_usage":        totalUsage,
	}
}

// cleanupRoutine periodically cleans up expired connections
func (cp *ConnectionPool) cleanupRoutine() {
	for {
		select {
		case <-cp.cleanupTicker.C:
			cp.cleanupExpiredConnections()
		case <-cp.cleanupDone:
			return
		}
	}
}

// cleanupExpiredConnections removes expired connections from the pool
func (cp *ConnectionPool) cleanupExpiredConnections() {
	cp.mu.Lock()
	defer cp.mu.Unlock()

	expiredIDs := make([]string, 0)

	for id, conn := range cp.connections {
		if conn.IsExpired(cp.maxIdleTime) {
			expiredIDs = append(expiredIDs, id)
		}
	}

	// Close and remove expired connections
	for _, id := range expiredIDs {
		if conn, exists := cp.connections[id]; exists {
			conn.Close()
			delete(cp.connections, id)
		}
	}
}

// evictExpiredConnection tries to evict one expired connection
func (cp *ConnectionPool) evictExpiredConnection() bool {
	for id, conn := range cp.connections {
		if conn.IsExpired(cp.maxIdleTime) {
			conn.Close()
			delete(cp.connections, id)
			return true
		}
	}
	return false
}

// envEqual checks if two environment slices are equal
func envEqual(env1, env2 []string) bool {
	if len(env1) != len(env2) {
		return false
	}

	env1Map := make(map[string]string)
	env2Map := make(map[string]string)

	for _, env := range env1 {
		if idx := findEquals(env); idx > 0 {
			env1Map[env[:idx]] = env[idx+1:]
		}
	}

	for _, env := range env2 {
		if idx := findEquals(env); idx > 0 {
			env2Map[env[:idx]] = env[idx+1:]
		}
	}

	if len(env1Map) != len(env2Map) {
		return false
	}

	for key, val1 := range env1Map {
		if val2, exists := env2Map[key]; !exists || val1 != val2 {
			return false
		}
	}

	return true
}

// findEquals finds the first '=' character in a string
func findEquals(s string) int {
	for i, r := range s {
		if r == '=' {
			return i
		}
	}
	return -1
}

// PooledCommandExecutor provides command execution with connection pooling
type PooledCommandExecutor struct {
	pool        *ConnectionPool
	defaultPool bool
}

// NewPooledCommandExecutor creates a new pooled command executor
func NewPooledCommandExecutor(pool *ConnectionPool) *PooledCommandExecutor {
	if pool == nil {
		// Create default pool
		pool = NewConnectionPool(
			20,            // max 20 connections
			5*time.Minute, // 5 minute idle timeout
			1*time.Minute, // cleanup every minute
		)
		return &PooledCommandExecutor{
			pool:        pool,
			defaultPool: true,
		}
	}

	return &PooledCommandExecutor{
		pool:        pool,
		defaultPool: false,
	}
}

// ExecuteCommand executes a command using a pooled connection
func (pce *PooledCommandExecutor) ExecuteCommand(workingDir string, env []string, command string, args ...string) (*exec.Cmd, error) {
	// Get connection from pool
	conn, err := pce.pool.GetConnection(workingDir, env)
	if err != nil {
		return nil, fmt.Errorf("failed to get connection: %w", err)
	}

	// Execute command
	cmd, err := conn.Execute(command, args...)
	if err != nil {
		pce.pool.CloseConnection(conn) // Close on error
		return nil, err
	}

	// Return connection to pool after command completes
	go func() {
		// Wait for command to complete
		if err := cmd.Wait(); err != nil {
			// Log the error but don't fail - connection cleanup should still happen
			fmt.Printf("Warning: command execution failed: %v\n", err)
		}
		pce.pool.ReturnConnection(conn)
	}()

	return cmd, nil
}

// GetPoolStats returns connection pool statistics
func (pce *PooledCommandExecutor) GetPoolStats() map[string]interface{} {
	return pce.pool.GetStats()
}

// Cleanup closes the connection pool if it's a default pool
func (pce *PooledCommandExecutor) Cleanup() {
	if pce.defaultPool && pce.pool != nil {
		pce.pool.Close()
	}
}

// Global connection pool for shared use
var globalConnectionPool = NewConnectionPool(
	50,             // max 50 connections
	10*time.Minute, // 10 minute idle timeout
	2*time.Minute,  // cleanup every 2 minutes
)

// GetGlobalConnectionPool returns the global connection pool
func GetGlobalConnectionPool() *ConnectionPool {
	return globalConnectionPool
}

// GetPooledCommandExecutor returns a command executor that uses the global connection pool
func GetPooledCommandExecutor() *PooledCommandExecutor {
	return NewPooledCommandExecutor(globalConnectionPool)
}
