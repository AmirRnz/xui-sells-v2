package main

import (
	"fmt"
	"time"

	"github.com/spf13/cobra"

	"xui-sells-v2/internal/app/backup"
)

var backupCmd = &cobra.Command{
	Use:   "backup",
	Short: "Create and restore encrypted system or instance backups (P23)",
}

var (
	backupScope    string
	backupInstID   int64
	backupPassword string
	backupOutPath  string
	backupFilePath string
	targetInstID   int64
)

var backupCreateCmd = &cobra.Command{
	Use:   "create",
	Short: "Create an AES-GCM encrypted backup archive",
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()

		if backupPassword == "" {
			return fmt.Errorf("--password is required")
		}

		scope := backup.BackupScope(backupScope)
		if scope != backup.ScopeGlobal && scope != backup.ScopeInstance {
			scope = backup.ScopeGlobal
		}

		var instIDPtr *int64
		if scope == backup.ScopeInstance {
			if backupInstID <= 0 {
				return fmt.Errorf("--instance-id is required when scope is instance")
			}
			instIDPtr = &backupInstID
		}

		if backupOutPath == "" {
			if scope == backup.ScopeGlobal {
				backupOutPath = fmt.Sprintf("backup-global-%d.tar.gz.enc", time.Now().Unix())
			} else {
				backupOutPath = fmt.Sprintf("backup-instance-%d-%d.tar.gz.enc", backupInstID, time.Now().Unix())
			}
		}

		if backupSvc == nil {
			return fmt.Errorf("backup service not initialized")
		}

		err := backupSvc.CreateBackupToFile(ctx, scope, instIDPtr, backupPassword, backupOutPath)
		if err != nil {
			return fmt.Errorf("backup creation failed: %w", err)
		}

		fmt.Printf("✅ Encrypted backup archive successfully created: %s (Scope: %s)\n", backupOutPath, scope)
		return nil
	},
}

var backupRestoreCmd = &cobra.Command{
	Use:   "restore",
	Short: "Restore data from an AES-GCM encrypted backup archive",
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()

		if backupFilePath == "" {
			return fmt.Errorf("--file is required")
		}
		if backupPassword == "" {
			return fmt.Errorf("--password is required")
		}

		if backupSvc == nil {
			return fmt.Errorf("backup service not initialized")
		}

		var targetPtr *int64
		if targetInstID > 0 {
			targetPtr = &targetInstID
		}

		manifest, err := backupSvc.RestoreBackupFromFile(ctx, backupFilePath, backupPassword, targetPtr)
		if err != nil {
			return fmt.Errorf("backup restoration failed: %w", err)
		}

		fmt.Printf("✅ Backup successfully verified and restored!\n")
		fmt.Printf("   Scope: %s\n", manifest.Scope)
		fmt.Printf("   Created: %s\n", manifest.CreatedAt.Format(time.RFC3339))
		fmt.Printf("   Checksum SHA-256: %s\n", manifest.ChecksumSHA)
		return nil
	},
}

func init() {
	backupCmd.AddCommand(backupCreateCmd)
	backupCmd.AddCommand(backupRestoreCmd)

	backupCreateCmd.Flags().StringVar(&backupScope, "scope", "global", "Backup scope: 'global' or 'instance'")
	backupCreateCmd.Flags().Int64Var(&backupInstID, "instance-id", 0, "Instance ID (required if scope is instance)")
	backupCreateCmd.Flags().StringVar(&backupPassword, "password", "", "Encryption password (required)")
	backupCreateCmd.Flags().StringVarP(&backupOutPath, "out", "o", "", "Output file path")

	backupRestoreCmd.Flags().StringVarP(&backupFilePath, "file", "f", "", "Encrypted backup file path (required)")
	backupRestoreCmd.Flags().StringVar(&backupPassword, "password", "", "Decryption password (required)")
	backupRestoreCmd.Flags().Int64Var(&targetInstID, "target-instance-id", 0, "Target instance ID (optional, for instance restore)")
}
