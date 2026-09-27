package instance

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"xui-sells-v2/internal/domain"
)

var (
	ErrNotFound = errors.New("instance not found")
)

// Store defines operations for instance persistence and hierarchy queries.
type Store interface {
	List(ctx context.Context) ([]domain.Instance, error)
	Get(ctx context.Context, id int64) (*domain.Instance, error)
	Create(ctx context.Context, inst *domain.Instance) error
	Update(ctx context.Context, inst *domain.Instance) error
	Delete(ctx context.Context, id int64) error
	CountChildBots(ctx context.Context, parentID int64) (int, error)
	ListChildBots(ctx context.Context, parentID int64) ([]domain.Instance, error)
}

// MemoryStore provides in-memory thread-safe instance storage.
type MemoryStore struct {
	mu        sync.RWMutex
	instances map[int64]domain.Instance
	nextID    int64
	filePath  string
}

// NewMemoryStore creates an empty in-memory store.
func NewMemoryStore() *MemoryStore {
	return &MemoryStore{
		instances: make(map[int64]domain.Instance),
		nextID:    1,
	}
}

// NewFileStore creates or loads a persistent JSON file-backed instance store.
func NewFileStore(filePath string) (*MemoryStore, error) {
	store := &MemoryStore{
		instances: make(map[int64]domain.Instance),
		nextID:    1,
		filePath:  filePath,
	}

	if filePath != "" {
		if _, err := os.Stat(filePath); err == nil {
			data, err := os.ReadFile(filePath)
			if err != nil {
				return nil, fmt.Errorf("failed to read instances file: %w", err)
			}
			var list []domain.Instance
			if len(data) > 0 {
				if err := json.Unmarshal(data, &list); err != nil {
					return nil, fmt.Errorf("failed to parse instances json: %w", err)
				}
				for _, inst := range list {
					store.instances[inst.ID] = inst
					if inst.ID >= store.nextID {
						store.nextID = inst.ID + 1
					}
				}
			}
		}
	}

	return store, nil
}

func (s *MemoryStore) saveToFileLocked() error {
	if s.filePath == "" {
		return nil
	}
	dir := filepath.Dir(s.filePath)
	if dir != "" && dir != "." {
		_ = os.MkdirAll(dir, 0755)
	}

	list := make([]domain.Instance, 0, len(s.instances))
	for _, inst := range s.instances {
		list = append(list, inst)
	}
	bytes, err := json.MarshalIndent(list, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(s.filePath, bytes, 0600)
}

func (s *MemoryStore) List(ctx context.Context) ([]domain.Instance, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	result := make([]domain.Instance, 0, len(s.instances))
	for _, inst := range s.instances {
		result = append(result, inst)
	}
	return result, nil
}

func (s *MemoryStore) Get(ctx context.Context, id int64) (*domain.Instance, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	inst, ok := s.instances[id]
	if !ok {
		return nil, ErrNotFound
	}
	return &inst, nil
}

func (s *MemoryStore) Create(ctx context.Context, inst *domain.Instance) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if inst.ID <= 0 {
		inst.ID = s.nextID
		s.nextID++
	} else if inst.ID >= s.nextID {
		s.nextID = inst.ID + 1
	}

	now := time.Now().UTC()
	if inst.CreatedAt.IsZero() {
		inst.CreatedAt = now
	}
	inst.UpdatedAt = now

	s.instances[inst.ID] = *inst
	return s.saveToFileLocked()
}

func (s *MemoryStore) Update(ctx context.Context, inst *domain.Instance) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.instances[inst.ID]; !ok {
		return ErrNotFound
	}

	inst.UpdatedAt = time.Now().UTC()
	s.instances[inst.ID] = *inst
	return s.saveToFileLocked()
}

func (s *MemoryStore) Delete(ctx context.Context, id int64) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.instances[id]; !ok {
		return ErrNotFound
	}
	delete(s.instances, id)
	return s.saveToFileLocked()
}

func (s *MemoryStore) CountChildBots(ctx context.Context, parentID int64) (int, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	count := 0
	for _, inst := range s.instances {
		if inst.ParentInstanceID != nil && *inst.ParentInstanceID == parentID {
			count++
		}
	}
	return count, nil
}

func (s *MemoryStore) ListChildBots(ctx context.Context, parentID int64) ([]domain.Instance, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var children []domain.Instance
	for _, inst := range s.instances {
		if inst.ParentInstanceID != nil && *inst.ParentInstanceID == parentID {
			children = append(children, inst)
		}
	}
	return children, nil
}
