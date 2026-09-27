package backup_test

import (
	"context"
	"errors"
	"testing"

	"xui-sells-v2/internal/app/backup"
	"xui-sells-v2/internal/domain"
	"xui-sells-v2/internal/domain/money"
)

type mockDataProvider struct {
	globalData   *backup.BackupData
	instanceData map[int64]*backup.BackupData

	restoredGlobalData   *backup.BackupData
	restoredInstanceData map[int64]*backup.BackupData
}

func newMockDataProvider() *mockDataProvider {
	return &mockDataProvider{
		instanceData:         make(map[int64]*backup.BackupData),
		restoredInstanceData: make(map[int64]*backup.BackupData),
	}
}

func (m *mockDataProvider) GetGlobalData(ctx context.Context) (*backup.BackupData, error) {
	if m.globalData == nil {
		return nil, errors.New("no global data")
	}
	return m.globalData, nil
}

func (m *mockDataProvider) GetInstanceData(ctx context.Context, instanceID int64) (*backup.BackupData, error) {
	data, ok := m.instanceData[instanceID]
	if !ok {
		return nil, errors.New("instance data not found")
	}
	return data, nil
}

func (m *mockDataProvider) RestoreGlobalData(ctx context.Context, data *backup.BackupData) error {
	m.restoredGlobalData = data
	return nil
}

func (m *mockDataProvider) RestoreInstanceData(ctx context.Context, instanceID int64, data *backup.BackupData) error {
	m.restoredInstanceData[instanceID] = data
	return nil
}

func TestGlobalBackupAndRestoreRoundtrip(t *testing.T) {
	ctx := context.Background()
	provider := newMockDataProvider()

	parentID := int64(1)
	childID := int64(2)
	provider.globalData = &backup.BackupData{
		Instances: []domain.Instance{
			{ID: parentID, Name: "Reseller Master", Type: domain.InstanceTypeReseller, BotToken: "123:PARENT", AdminTelegramID: 100},
			{ID: childID, Name: "Child Bot", Type: domain.InstanceTypeChildCustomer, BotToken: "123:CHILD", ParentInstanceID: &parentID, AdminTelegramID: 100},
		},
		Plans: []domain.Plan{
			{ID: 1, InstanceID: parentID, Name: "VIP Plan", TrafficBytes: 50 * 1024 * 1024 * 1024, BasePrice: money.NewToman(150_000)},
		},
		Services: []domain.Service{
			{ID: 1, InstanceID: childID, ClientEmail: "user1@vpn.com", SubID: "sub-12345"},
		},
		Orders: []domain.Order{
			{ID: 1, InstanceID: childID, UserID: 10, Amount: money.NewToman(150_000), Status: domain.OrderStatusApproved},
		},
		Transactions: []domain.WalletTransaction{
			{ID: 1, InstanceID: parentID, UserID: 100, Amount: money.NewToman(500_000), Type: domain.TxTypeTopup},
		},
		Reservations: []domain.ResellerFundingReservation{
			{ID: 1, ParentInstanceID: parentID, ChildInstanceID: childID, Amount: money.NewToman(150_000), Status: domain.ReservationStatusSettled},
		},
		Tickets: []domain.Ticket{
			{ID: 1, InstanceID: childID, Subject: "Speed issue", Status: domain.TicketStatusClosed},
		},
		MediaFiles: map[string][]byte{
			"receipt_order_1.jpg": []byte("fake_jpeg_content_here"),
		},
	}

	svc := backup.NewService(provider)
	password := "SecretP@ssw0rd2026!"

	// 1. Create global backup
	archiveBytes, err := svc.CreateBackup(ctx, backup.ScopeGlobal, nil, password)
	if err != nil {
		t.Fatalf("failed to create global backup: %v", err)
	}

	if len(archiveBytes) == 0 {
		t.Fatal("backup output is empty")
	}

	// 2. Restore global backup
	manifest, err := svc.RestoreBackup(ctx, archiveBytes, password, nil)
	if err != nil {
		t.Fatalf("failed to restore global backup: %v", err)
	}

	if manifest.Scope != backup.ScopeGlobal {
		t.Errorf("expected scope global, got %s", manifest.Scope)
	}
	if manifest.ChecksumSHA == "" {
		t.Error("expected non-empty checksum")
	}

	// Verify restored data matches original
	restored := provider.restoredGlobalData
	if restored == nil {
		t.Fatal("restored global data is nil")
	}

	if len(restored.Instances) != 2 {
		t.Errorf("expected 2 instances, got %d", len(restored.Instances))
	}
	if len(restored.Transactions) != 1 {
		t.Errorf("expected 1 transaction, got %d", len(restored.Transactions))
	}
	if len(restored.Reservations) != 1 {
		t.Errorf("expected 1 reservation, got %d", len(restored.Reservations))
	}
	if string(restored.MediaFiles["receipt_order_1.jpg"]) != "fake_jpeg_content_here" {
		t.Errorf("media file content mismatch")
	}
}

func TestInstanceBackupPreservesHierarchyWithoutDuplicatingLedger(t *testing.T) {
	ctx := context.Background()
	provider := newMockDataProvider()

	parentID := int64(10)
	childID := int64(20)

	// Instance data for parent reseller, including child instance metadata (P23)
	provider.instanceData[parentID] = &backup.BackupData{
		Instances: []domain.Instance{
			{ID: parentID, Name: "Reseller Bot", Type: domain.InstanceTypeReseller, BotToken: "token_p"},
			{ID: childID, Name: "Child Customer Bot", Type: domain.InstanceTypeChildCustomer, BotToken: "token_c", ParentInstanceID: &parentID},
		},
		Plans: []domain.Plan{
			{ID: 1, InstanceID: parentID, Name: "Reseller Custom Plan"},
		},
		Services: []domain.Service{
			{ID: 5, InstanceID: childID, SubID: "sub-child"},
		},
		Orders: []domain.Order{
			{ID: 101, InstanceID: childID, UserID: 50, Status: domain.OrderStatusApproved},
		},
		// Even if provider populated transactions and reservations, P23 dictates they must NOT be included in instance backup!
		Transactions: []domain.WalletTransaction{
			{ID: 99, InstanceID: parentID, Amount: money.NewToman(999_000)},
		},
		Reservations: []domain.ResellerFundingReservation{
			{ID: 88, ParentInstanceID: parentID, Amount: money.NewToman(100_000)},
		},
		Tickets: []domain.Ticket{
			{ID: 1, InstanceID: parentID, Subject: "Reseller inquiry"},
		},
	}

	svc := backup.NewService(provider)
	password := "ResellerSafe#1"

	// Create instance backup
	archiveBytes, err := svc.CreateBackup(ctx, backup.ScopeInstance, &parentID, password)
	if err != nil {
		t.Fatalf("failed to create instance backup: %v", err)
	}

	// Restore into another environment / instance
	targetID := int64(999)
	manifest, err := svc.RestoreBackup(ctx, archiveBytes, password, &targetID)
	if err != nil {
		t.Fatalf("failed to restore instance backup: %v", err)
	}

	if manifest.Scope != backup.ScopeInstance {
		t.Errorf("expected scope instance, got %s", manifest.Scope)
	}

	restored := provider.restoredInstanceData[targetID]
	if restored == nil {
		t.Fatal("restored instance data is nil")
	}

	// P23 Verification:
	// 1. Parent and child instances are preserved with parent-child relationship intact
	if len(restored.Instances) != 2 {
		t.Errorf("expected 2 instances (parent + child), got %d", len(restored.Instances))
	}
	if restored.Instances[1].ParentInstanceID == nil || *restored.Instances[1].ParentInstanceID != parentID {
		t.Errorf("child instance parent relation missing or modified")
	}

	// 2. Shared financial state (transactions and reservations) was NOT duplicated
	if len(restored.Transactions) != 0 {
		t.Errorf("P23 violation: instance backup must not duplicate wallet transactions, got %d", len(restored.Transactions))
	}
	if len(restored.Reservations) != 0 {
		t.Errorf("P23 violation: instance backup must not duplicate reservations, got %d", len(restored.Reservations))
	}
}

func TestBackupSecurityAndIntegrityFailures(t *testing.T) {
	ctx := context.Background()
	provider := newMockDataProvider()
	provider.globalData = &backup.BackupData{
		Instances: []domain.Instance{{ID: 1, Name: "Test"}},
	}
	svc := backup.NewService(provider)
	correctPassword := "CorrectPassword123!"

	archiveBytes, err := svc.CreateBackup(ctx, backup.ScopeGlobal, nil, correctPassword)
	if err != nil {
		t.Fatalf("failed to create backup: %v", err)
	}

	// 1. Wrong password
	_, err = svc.RestoreBackup(ctx, archiveBytes, "WrongPassword!!!", nil)
	if !errors.Is(err, backup.ErrDecryptionFailed) {
		t.Errorf("expected ErrDecryptionFailed on wrong password, got %v", err)
	}

	// 2. Empty password
	_, err = svc.RestoreBackup(ctx, archiveBytes, "", nil)
	if !errors.Is(err, backup.ErrPasswordRequired) {
		t.Errorf("expected ErrPasswordRequired, got %v", err)
	}

	// 3. Corrupted ciphertext (tampered file)
	tamperedBytes := make([]byte, len(archiveBytes))
	copy(tamperedBytes, archiveBytes)
	tamperedBytes[len(tamperedBytes)-5] ^= 0xFF // Flip bits in ciphertext
	_, err = svc.RestoreBackup(ctx, tamperedBytes, correctPassword, nil)
	if !errors.Is(err, backup.ErrDecryptionFailed) {
		t.Errorf("expected ErrDecryptionFailed on tampered ciphertext, got %v", err)
	}

	// 4. Invalid Magic header
	badHeaderBytes := make([]byte, len(archiveBytes))
	copy(badHeaderBytes, archiveBytes)
	badHeaderBytes[0] = 'B'
	badHeaderBytes[1] = 'A'
	badHeaderBytes[2] = 'D'
	badHeaderBytes[3] = '!'
	_, err = svc.RestoreBackup(ctx, badHeaderBytes, correctPassword, nil)
	if !errors.Is(err, backup.ErrInvalidBackupFormat) {
		t.Errorf("expected ErrInvalidBackupFormat, got %v", err)
	}

	// 5. Truncated file
	_, err = svc.RestoreBackup(ctx, archiveBytes[:10], correctPassword, nil)
	if !errors.Is(err, backup.ErrInvalidBackupFormat) {
		t.Errorf("expected ErrInvalidBackupFormat on truncated file, got %v", err)
	}
}
