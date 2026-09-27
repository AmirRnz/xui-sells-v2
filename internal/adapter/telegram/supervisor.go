package telegram

import (
	"context"
	"errors"
	"fmt"
	"sync"
)

var (
	ErrBotAlreadyRunning = errors.New("bot instance is already running")
	ErrBotNotFound       = errors.New("bot instance not found")
)

type managedBot struct {
	instance *BotInstance
	cancel   context.CancelFunc
}

// Supervisor coordinates the multi-tenant bot lifecycle for customer, reseller, and child bots.
type Supervisor struct {
	mu   sync.RWMutex
	bots map[int64]*managedBot
	ctx  context.Context
}

// NewSupervisor creates a supervisor.
func NewSupervisor() *Supervisor {
	return &Supervisor{
		bots: make(map[int64]*managedBot),
	}
}

// SetContext sets the root context for spawned bot instances.
func (s *Supervisor) SetContext(ctx context.Context) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.ctx = ctx
}

// RegisterBot adds and starts a bot instance.
func (s *Supervisor) RegisterBot(b *BotInstance) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.bots[b.Instance.ID]; exists {
		return fmt.Errorf("%w: ID %d", ErrBotAlreadyRunning, b.Instance.ID)
	}

	baseCtx := s.ctx
	if baseCtx == nil {
		baseCtx = context.Background()
	}
	botCtx, cancel := context.WithCancel(baseCtx)

	s.bots[b.Instance.ID] = &managedBot{
		instance: b,
		cancel:   cancel,
	}

	go b.StartPolling(botCtx)
	return nil
}

// StopBot halts and unregisters a bot instance.
func (s *Supervisor) StopBot(instanceID int64) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	mb, exists := s.bots[instanceID]
	if !exists {
		return ErrBotNotFound
	}

	mb.cancel()
	delete(s.bots, instanceID)
	return nil
}

// GetBot retrieves a running bot instance.
func (s *Supervisor) GetBot(instanceID int64) (*BotInstance, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	mb, ok := s.bots[instanceID]
	if !ok {
		return nil, false
	}
	return mb.instance, true
}

// ListRunning returns the IDs of all running bots.
func (s *Supervisor) ListRunning() []int64 {
	s.mu.RLock()
	defer s.mu.RUnlock()
	ids := make([]int64, 0, len(s.bots))
	for id := range s.bots {
		ids = append(ids, id)
	}
	return ids
}

// CountChildBots returns the count of child customer bots running under a parent reseller instance (P19 & P22).
func (s *Supervisor) CountChildBots(parentInstanceID int64) int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	count := 0
	for _, mb := range s.bots {
		b := mb.instance
		if b.Instance.IsChild() && b.Instance.ParentInstanceID != nil && *b.Instance.ParentInstanceID == parentInstanceID {
			count++
		}
	}
	return count
}

// Count returns the total number of registered bots.
func (s *Supervisor) Count() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.bots)
}

// StartAll begins polling for all registered bots.
func (s *Supervisor) StartAll(ctx context.Context) error {
	s.SetContext(ctx)
	return nil
}

// StopAll halts and unregisters all running bots.
func (s *Supervisor) StopAll() {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, mb := range s.bots {
		mb.cancel()
	}
	s.bots = make(map[int64]*managedBot)
}
