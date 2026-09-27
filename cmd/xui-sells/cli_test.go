package main

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"xui-sells-v2/internal/app/backup"
	"xui-sells-v2/internal/app/instance"
	"xui-sells-v2/internal/domain"
)

func TestMenuAddInstanceWizard(t *testing.T) {
	ctx := context.Background()
	store := instance.NewMemoryStore()
	provider := backup.NewInstanceStoreDataProvider(store)
	bkp := backup.NewService(provider)

	// Simulated inputs for Add Instance (Reseller type with Web Panel):
	// Main menu: 2 (Add New Instance)
	// 1. Type: 2 (Reseller)
	// Name: "Alpha Reseller"
	// 2. Token: "999:RESELLER_TOKEN"
	// 3. Admin TG ID: "123456"
	// 4. Panel URL: "https://panel.alpha.com:2053"
	// 5. API Key: "apikey-xyz"
	// 6. Lang: "1" (Persian)
	// 7. Bring Web Panel Online: "y"
	// Main menu: 5 (Exit)
	inputs := strings.Join([]string{
		"2",
		"2",
		"Alpha Reseller",
		"999:RESELLER_TOKEN",
		"123456",
		"https://panel.alpha.com:2053",
		"apikey-xyz",
		"1",
		"y",
		"5",
	}, "\n") + "\n"

	in := strings.NewReader(inputs)
	out := new(bytes.Buffer)

	err := RunMenu(ctx, in, out, store, bkp)
	if err != nil {
		t.Fatalf("RunMenu returned error: %v", err)
	}

	outputStr := out.String()
	if !strings.Contains(outputStr, "Instance 'Alpha Reseller' created successfully") {
		t.Errorf("expected success message in output, got: %s", outputStr)
	}
	if !strings.Contains(outputStr, "Reseller Web Panel scheduled for online deployment") {
		t.Errorf("expected web panel notice in output, got: %s", outputStr)
	}

	// Verify store has the new instance
	list, err := store.List(ctx)
	if err != nil || len(list) != 1 {
		t.Fatalf("expected 1 instance in store, got %d", len(list))
	}
	inst := list[0]
	if inst.Name != "Alpha Reseller" || inst.Type != domain.InstanceTypeReseller {
		t.Errorf("unexpected instance saved: %+v", inst)
	}
	if inst.AdminTelegramID != 123456 {
		t.Errorf("admin telegram ID mismatch: %d", inst.AdminTelegramID)
	}
}

func TestMenuViewManageInstancesChildCountDisplay(t *testing.T) {
	ctx := context.Background()
	store := instance.NewMemoryStore()
	provider := backup.NewInstanceStoreDataProvider(store)
	bkp := backup.NewService(provider)

	// Create Reseller parent
	parent := domain.Instance{
		Name:            "VIP Reseller",
		Type:            domain.InstanceTypeReseller,
		BotToken:        "111:PARENT",
		AdminTelegramID: 8888,
		IsActive:        true,
	}
	_ = store.Create(ctx, &parent)

	// Create 2 Child bots
	ch1 := domain.Instance{
		Name:             "Child 1",
		Type:             domain.InstanceTypeChildCustomer,
		BotToken:         "111:CH1",
		ParentInstanceID: &parent.ID,
		AdminTelegramID:  8888,
		IsActive:         true,
	}
	ch2 := domain.Instance{
		Name:             "Child 2",
		Type:             domain.InstanceTypeChildCustomer,
		BotToken:         "111:CH2",
		ParentInstanceID: &parent.ID,
		AdminTelegramID:  8888,
		IsActive:         false,
	}
	_ = store.Create(ctx, &ch1)
	_ = store.Create(ctx, &ch2)

	// Input:
	// 1 (View / Manage Instances)
	// b (Back to instances menu / main menu)
	// 5 (Exit)
	inputs := "1\nb\n5\n"
	in := strings.NewReader(inputs)
	out := new(bytes.Buffer)

	err := RunMenu(ctx, in, out, store, bkp)
	if err != nil {
		t.Fatalf("RunMenu error: %v", err)
	}

	outputStr := out.String()
	// P22: Prominently displaying child-bot count for each parent reseller instance!
	if !strings.Contains(outputStr, "Child Bots: 2") {
		t.Errorf("expected 'Child Bots: 2' in output for reseller parent, got: %s", outputStr)
	}
	if !strings.Contains(outputStr, "[Child ID: 2] Child 1") || !strings.Contains(outputStr, "[Child ID: 3] Child 2") {
		t.Errorf("expected child bots listed under parent, got: %s", outputStr)
	}
}

func TestMenuBackupAndRestore(t *testing.T) {
	ctx := context.Background()
	store := instance.NewMemoryStore()
	provider := backup.NewInstanceStoreDataProvider(store)
	bkp := backup.NewService(provider)

	inst := domain.Instance{
		Name:            "Production Customer Bot",
		Type:            domain.InstanceTypeCustomer,
		BotToken:        "555:TOKEN",
		AdminTelegramID: 9999,
		IsActive:        true,
	}
	_ = store.Create(ctx, &inst)

	tmpDir := t.TempDir()
	backupPath := filepath.Join(tmpDir, "global-test-backup.tar.gz.enc")

	// Menu actions:
	// 3 (Backup & Restore)
	// 1 (Create Global Backup)
	// password: "SafePassword123!"
	// file path: backupPath
	// 4 (Back to main menu)
	// 5 (Exit)
	inputs := strings.Join([]string{
		"3",
		"1",
		"SafePassword123!",
		backupPath,
		"4",
		"5",
	}, "\n") + "\n"

	in := strings.NewReader(inputs)
	out := new(bytes.Buffer)

	err := RunMenu(ctx, in, out, store, bkp)
	if err != nil {
		t.Fatalf("RunMenu backup error: %v", err)
	}

	if _, err := os.Stat(backupPath); err != nil {
		t.Fatalf("expected backup file to be created at %s, err: %v", backupPath, err)
	}

	// Now test restore via backup service directly
	newStore := instance.NewMemoryStore()
	newProvider := backup.NewInstanceStoreDataProvider(newStore)
	newBkp := backup.NewService(newProvider)

	manifest, err := newBkp.RestoreBackupFromFile(ctx, backupPath, "SafePassword123!", nil)
	if err != nil {
		t.Fatalf("failed to restore created backup: %v", err)
	}

	if manifest.Scope != backup.ScopeGlobal {
		t.Errorf("expected global scope, got %s", manifest.Scope)
	}

	restoredList, _ := newStore.List(ctx)
	if len(restoredList) != 1 || restoredList[0].Name != "Production Customer Bot" {
		t.Errorf("restored instances mismatch: %+v", restoredList)
	}
}

func TestNonInteractiveCLICommands(t *testing.T) {
	ctx := context.Background()
	store := instance.NewMemoryStore()
	appStore = store
	provider := backup.NewInstanceStoreDataProvider(appStore)
	backupSvc = backup.NewService(provider)

	// 1. instance add
	rootCmd.SetArgs([]string{
		"instance", "add",
		"--name", "Direct CLI Bot",
		"--type", "customer",
		"--token", "777:CLI_TOKEN",
		"--admin-tg", "4321",
	})
	if err := rootCmd.ExecuteContext(ctx); err != nil {
		t.Fatalf("instance add failed: %v", err)
	}

	list, _ := store.List(ctx)
	if len(list) != 1 || list[0].Name != "Direct CLI Bot" {
		t.Fatalf("expected 1 instance added, got %+v", list)
	}
	botID := list[0].ID

	// 2. instance list
	rootCmd.SetArgs([]string{"instance", "list"})
	if err := rootCmd.ExecuteContext(ctx); err != nil {
		t.Fatalf("instance list failed: %v", err)
	}

	// 3. instance get
	rootCmd.SetArgs([]string{"instance", "get", "1"})
	if err := rootCmd.ExecuteContext(ctx); err != nil {
		t.Fatalf("instance get failed: %v", err)
	}

	// 4. instance edit
	rootCmd.SetArgs([]string{"instance", "edit", "1", "--name", "Renamed Bot"})
	if err := rootCmd.ExecuteContext(ctx); err != nil {
		t.Fatalf("instance edit failed: %v", err)
	}
	updated, _ := store.Get(ctx, botID)
	if updated.Name != "Renamed Bot" {
		t.Errorf("expected Renamed Bot, got %s", updated.Name)
	}

	// 5. backup create and restore
	tmpDir := t.TempDir()
	bkpFile := filepath.Join(tmpDir, "cli-backup.tar.gz.enc")

	rootCmd.SetArgs([]string{
		"backup", "create",
		"--scope", "global",
		"--password", "CLIPassword999!",
		"--out", bkpFile,
	})
	if err := rootCmd.ExecuteContext(ctx); err != nil {
		t.Fatalf("backup create failed: %v", err)
	}

	if _, err := os.Stat(bkpFile); err != nil {
		t.Fatalf("backup file not found: %s", bkpFile)
	}

	rootCmd.SetArgs([]string{
		"backup", "restore",
		"--file", bkpFile,
		"--password", "CLIPassword999!",
	})
	if err := rootCmd.ExecuteContext(ctx); err != nil {
		t.Fatalf("backup restore failed: %v", err)
	}

	// 6. update --check-only
	rootCmd.SetArgs([]string{"update", "--check-only"})
	if err := rootCmd.ExecuteContext(ctx); err != nil {
		t.Fatalf("update check-only failed: %v", err)
	}

	// 7. instance delete
	rootCmd.SetArgs([]string{"instance", "delete", "1"})
	if err := rootCmd.ExecuteContext(ctx); err != nil {
		t.Fatalf("instance delete failed: %v", err)
	}
	remaining, _ := store.List(ctx)
	if len(remaining) != 0 {
		t.Errorf("expected 0 instances after delete, got %d", len(remaining))
	}
}
