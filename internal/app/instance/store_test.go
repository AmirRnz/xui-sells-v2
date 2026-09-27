package instance_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"xui-sells-v2/internal/app/instance"
	"xui-sells-v2/internal/domain"
)

func TestInstanceStoreOperations(t *testing.T) {
	ctx := context.Background()
	store := instance.NewMemoryStore()

	// 1. Create Parent Reseller
	parent := domain.Instance{
		Name:            "Reseller Parent",
		Type:            domain.InstanceTypeReseller,
		BotToken:        "123:PARENT",
		AdminTelegramID: 1001,
		IsActive:        true,
	}
	err := store.Create(ctx, &parent)
	if err != nil {
		t.Fatalf("failed to create parent instance: %v", err)
	}
	if parent.ID == 0 {
		t.Fatal("expected assigned ID")
	}

	// 2. Create Child Bots
	child1 := domain.Instance{
		Name:             "Child Bot 1",
		Type:             domain.InstanceTypeChildCustomer,
		BotToken:         "123:CHILD1",
		ParentInstanceID: &parent.ID,
		AdminTelegramID:  1001,
		IsActive:         true,
	}
	child2 := domain.Instance{
		Name:             "Child Bot 2",
		Type:             domain.InstanceTypeChildCustomer,
		BotToken:         "123:CHILD2",
		ParentInstanceID: &parent.ID,
		AdminTelegramID:  1001,
		IsActive:         true,
	}
	_ = store.Create(ctx, &child1)
	_ = store.Create(ctx, &child2)

	// 3. Check child count (P22)
	count, err := store.CountChildBots(ctx, parent.ID)
	if err != nil {
		t.Fatalf("failed to count child bots: %v", err)
	}
	if count != 2 {
		t.Errorf("expected 2 child bots, got %d", count)
	}

	children, err := store.ListChildBots(ctx, parent.ID)
	if err != nil || len(children) != 2 {
		t.Errorf("expected 2 children returned, got %d (err: %v)", len(children), err)
	}

	// 4. Update
	parent.Name = "Updated Reseller Name"
	err = store.Update(ctx, &parent)
	if err != nil {
		t.Fatalf("failed to update parent: %v", err)
	}
	fetched, _ := store.Get(ctx, parent.ID)
	if fetched.Name != "Updated Reseller Name" {
		t.Errorf("expected updated name, got %s", fetched.Name)
	}

	// 5. Delete child
	err = store.Delete(ctx, child1.ID)
	if err != nil {
		t.Fatalf("failed to delete child: %v", err)
	}
	count, _ = store.CountChildBots(ctx, parent.ID)
	if count != 1 {
		t.Errorf("expected 1 child bot after delete, got %d", count)
	}
}

func TestFileInstanceStorePersistence(t *testing.T) {
	ctx := context.Background()
	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "instances.json")

	store1, err := instance.NewFileStore(filePath)
	if err != nil {
		t.Fatalf("failed to create file store: %v", err)
	}

	inst := domain.Instance{
		Name:            "Persistent Bot",
		Type:            domain.InstanceTypeCustomer,
		BotToken:        "123:BOT",
		AdminTelegramID: 555,
	}
	_ = store1.Create(ctx, &inst)

	// Verify file was written
	if _, err := os.Stat(filePath); err != nil {
		t.Fatalf("expected file to exist: %v", err)
	}

	// Reload with new store instance
	store2, err := instance.NewFileStore(filePath)
	if err != nil {
		t.Fatalf("failed to load file store: %v", err)
	}

	list, _ := store2.List(ctx)
	if len(list) != 1 {
		t.Fatalf("expected 1 instance persisted, got %d", len(list))
	}
	if list[0].Name != "Persistent Bot" {
		t.Errorf("expected Persistent Bot, got %s", list[0].Name)
	}
}
