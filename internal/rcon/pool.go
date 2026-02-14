package rcon

import (
	"fmt"
	"sync"

	"cs2admin/internal/pkg/logger"
)

// Pool manages multiple RCON connections keyed by instance ID.
type Pool struct {
	mu      sync.RWMutex
	clients map[string]*Client
}

// NewPool creates a new connection pool.
func NewPool() *Pool {
	return &Pool{
		clients: make(map[string]*Client),
	}
}

// Get returns an existing client for the instance ID, or nil and false if not found.
func (p *Pool) Get(instanceID string) (*Client, bool) {
	p.mu.RLock()
	defer p.mu.RUnlock()
	client, ok := p.clients[instanceID]
	return client, ok
}

// Connect creates and connects a new RCON client for the given instance.
func (p *Pool) Connect(instanceID, addr, password string) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if _, exists := p.clients[instanceID]; exists {
		return fmt.Errorf("rcon: instance %s already connected", instanceID)
	}

	client := NewClient(addr, password)
	if err := client.Connect(); err != nil {
		logger.Log.Debug().Err(err).Str("instance", instanceID).Str("addr", addr).Msg("rcon: pool connect failed")
		return fmt.Errorf("rcon: connect %s: %w", instanceID, err)
	}

	p.clients[instanceID] = client
	logger.Log.Info().Str("instance", instanceID).Str("addr", addr).Msg("rcon: pool added connection")
	return nil
}

// Execute runs a command on the specified instance.
func (p *Pool) Execute(instanceID, command string) (string, error) {
	p.mu.RLock()
	client, ok := p.clients[instanceID]
	p.mu.RUnlock()

	if !ok || client == nil {
		return "", fmt.Errorf("rcon: instance %s not connected", instanceID)
	}

	result, err := client.Execute(command)
	if err != nil {
		logger.Log.Debug().Err(err).Str("instance", instanceID).Str("command", command).Msg("rcon: execute failed")
		return "", fmt.Errorf("rcon: execute on %s: %w", instanceID, err)
	}

	return result, nil
}

// Disconnect closes and removes the client for the given instance.
func (p *Pool) Disconnect(instanceID string) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	client, ok := p.clients[instanceID]
	if !ok {
		return nil
	}

	delete(p.clients, instanceID)
	if err := client.Close(); err != nil {
		logger.Log.Debug().Err(err).Str("instance", instanceID).Msg("rcon: pool disconnect close failed")
		return err
	}

	logger.Log.Info().Str("instance", instanceID).Msg("rcon: pool disconnected")
	return nil
}

// DisconnectAll closes all clients in the pool.
func (p *Pool) DisconnectAll() {
	p.mu.Lock()
	defer p.mu.Unlock()

	for id, client := range p.clients {
		if client != nil {
			_ = client.Close()
		}
		delete(p.clients, id)
	}
	logger.Log.Info().Msg("rcon: pool disconnected all")
}
