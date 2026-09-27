package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"

	"xui-sells-v2/internal/app/backup"
	"xui-sells-v2/internal/app/instance"
)

var (
	dataDir       string
	instancesFile string

	appStore    instance.Store
	backupSvc   *backup.Service

	rootCmd = &cobra.Command{
		Use:   "xui-sells",
		Short: "xui-sells-v2: Multi-tenant Telegram VPN sales & reseller platform",
		Long: `xui-sells-v2 is an integrated high-performance Telegram sales bot and reseller
management platform for 3x-ui panels, featuring unified instance management,
reseller child bots, embedded web panel, and encrypted backups.`,
	}
)

func init() {
	cobra.OnInitialize(initConfig)

	rootCmd.PersistentFlags().StringVar(&dataDir, "data-dir", "data", "Directory for persistent database and configuration files")
	rootCmd.PersistentFlags().StringVar(&instancesFile, "instances-file", "", "Custom path to instances.json file")

	rootCmd.AddCommand(menuCmd)
	rootCmd.AddCommand(instanceCmd)
	rootCmd.AddCommand(backupCmd)
	rootCmd.AddCommand(updateCmd)
	rootCmd.AddCommand(runCmd)
	rootCmd.AddCommand(versionCmd)
}

func initConfig() {
	if appStore != nil {
		if backupSvc == nil {
			provider := backup.NewInstanceStoreDataProvider(appStore)
			backupSvc = backup.NewService(provider)
		}
		return
	}

	if instancesFile == "" {
		instancesFile = filepath.Join(dataDir, "instances.json")
	}

	store, err := instance.NewFileStore(instancesFile)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Warning: failed to initialize instance file store: %v\n", err)
		appStore = instance.NewMemoryStore()
	} else {
		appStore = store
	}

	provider := backup.NewInstanceStoreDataProvider(appStore)
	backupSvc = backup.NewService(provider)
}

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print software version and build info",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("xui-sells-v2 version 2.0.0 (Go 1.26)")
	},
}

func main() {
	if err := rootCmd.ExecuteContext(context.Background()); err != nil {
		os.Exit(1)
	}
}
