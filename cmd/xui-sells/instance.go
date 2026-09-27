package main

import (
	"encoding/json"
	"fmt"
	"strconv"

	"github.com/spf13/cobra"

	"xui-sells-v2/internal/domain"
)

var instanceCmd = &cobra.Command{
	Use:   "instance",
	Short: "Manage bot instances and child bots (P01, P22)",
}

var (
	instName     string
	instType     string
	instToken    string
	instAdminTG  int64
	instPanelURL string
	instAPIKey   string
	instLang     string
	instParentID int64
	instActive   bool
)

var instanceListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all registered bot instances and child bots (P22)",
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		list, err := appStore.List(ctx)
		if err != nil {
			return err
		}

		if len(list) == 0 {
			fmt.Println("No instances found.")
			return nil
		}

		fmt.Printf("%-5s | %-20s | %-14s | %-12s | %-10s | %-8s\n", "ID", "Name", "Type", "Admin TG", "Child Bots", "Status")
		fmt.Println("--------------------------------------------------------------------------------")

		for _, inst := range list {
			childCount := 0
			if inst.IsReseller() {
				childCount, _ = appStore.CountChildBots(ctx, inst.ID)
			}

			status := "active"
			if !inst.IsActive {
				status = "disabled"
			}

			childStr := "-"
			if inst.IsReseller() {
				childStr = strconv.Itoa(childCount)
			}

			fmt.Printf("%-5d | %-20s | %-14s | %-12d | %-10s | %-8s\n",
				inst.ID, inst.Name, inst.Type, inst.AdminTelegramID, childStr, status)
		}
		return nil
	},
}

var instanceAddCmd = &cobra.Command{
	Use:   "add",
	Short: "Add a new customer, reseller, or child bot instance",
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()

		if instName == "" {
			return fmt.Errorf("--name is required")
		}
		if instToken == "" {
			return fmt.Errorf("--token is required")
		}
		if instAdminTG == 0 {
			return fmt.Errorf("--admin-tg is required")
		}

		typeEnum := domain.InstanceType(instType)
		if typeEnum != domain.InstanceTypeCustomer && typeEnum != domain.InstanceTypeReseller && typeEnum != domain.InstanceTypeChildCustomer {
			typeEnum = domain.InstanceTypeCustomer
		}

		langEnum := domain.Language(instLang)
		if langEnum != domain.LangFA && langEnum != domain.LangEN {
			langEnum = domain.LangFA
		}

		var parentID *int64
		if instParentID > 0 {
			parentID = &instParentID
		}

		newInst := domain.Instance{
			Name:             instName,
			Type:             typeEnum,
			BotToken:         instToken,
			AdminTelegramID:  instAdminTG,
			PanelURL:         instPanelURL,
			PanelAPIKey:      instAPIKey,
			DefaultLang:      langEnum,
			ParentInstanceID: parentID,
			IsActive:         true,
		}

		if err := appStore.Create(ctx, &newInst); err != nil {
			return fmt.Errorf("failed to create instance: %w", err)
		}

		fmt.Printf("✅ Instance created successfully! ID: %d, Name: %s, Type: %s\n", newInst.ID, newInst.Name, newInst.Type)
		return nil
	},
}

var instanceGetCmd = &cobra.Command{
	Use:   "get [id]",
	Short: "Get instance details by ID",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		id, err := strconv.ParseInt(args[0], 10, 64)
		if err != nil {
			return fmt.Errorf("invalid ID: %w", err)
		}

		inst, err := appStore.Get(ctx, id)
		if err != nil {
			return err
		}

		out, _ := json.MarshalIndent(inst, "", "  ")
		fmt.Println(string(out))

		if inst.IsReseller() {
			children, _ := appStore.ListChildBots(ctx, inst.ID)
			fmt.Printf("\nChild Bots (%d):\n", len(children))
			for _, ch := range children {
				fmt.Printf(" - [ID: %d] %s (Admin: %d, Active: %v)\n", ch.ID, ch.Name, ch.AdminTelegramID, ch.IsActive)
			}
		}
		return nil
	},
}

var instanceDeleteCmd = &cobra.Command{
	Use:   "delete [id]",
	Short: "Delete an instance by ID",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		id, err := strconv.ParseInt(args[0], 10, 64)
		if err != nil {
			return fmt.Errorf("invalid ID: %w", err)
		}

		if err := appStore.Delete(ctx, id); err != nil {
			return fmt.Errorf("failed to delete instance: %w", err)
		}

		fmt.Printf("✅ Instance %d deleted successfully.\n", id)
		return nil
	},
}

var instanceEditCmd = &cobra.Command{
	Use:   "edit [id]",
	Short: "Edit instance properties",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		id, err := strconv.ParseInt(args[0], 10, 64)
		if err != nil {
			return fmt.Errorf("invalid ID: %w", err)
		}

		inst, err := appStore.Get(ctx, id)
		if err != nil {
			return err
		}

		if cmd.Flags().Changed("name") {
			inst.Name = instName
		}
		if cmd.Flags().Changed("token") {
			inst.BotToken = instToken
		}
		if cmd.Flags().Changed("active") {
			inst.IsActive = instActive
		}

		if err := appStore.Update(ctx, inst); err != nil {
			return err
		}

		fmt.Printf("✅ Instance %d updated successfully.\n", inst.ID)
		return nil
	},
}

func init() {
	instanceCmd.AddCommand(instanceListCmd)
	instanceCmd.AddCommand(instanceAddCmd)
	instanceCmd.AddCommand(instanceGetCmd)
	instanceCmd.AddCommand(instanceDeleteCmd)
	instanceCmd.AddCommand(instanceEditCmd)

	instanceAddCmd.Flags().StringVar(&instName, "name", "", "Instance name (required)")
	instanceAddCmd.Flags().StringVar(&instType, "type", "customer", "Instance type: customer, reseller, or child_customer")
	instanceAddCmd.Flags().StringVar(&instToken, "token", "", "Telegram bot token (required)")
	instanceAddCmd.Flags().Int64Var(&instAdminTG, "admin-tg", 0, "Administrator Telegram user ID (required)")
	instanceAddCmd.Flags().StringVar(&instPanelURL, "panel-url", "", "3x-ui panel URL")
	instanceAddCmd.Flags().StringVar(&instAPIKey, "api-key", "", "3x-ui API key")
	instanceAddCmd.Flags().StringVar(&instLang, "lang", "fa", "Default bot language (fa or en)")
	instanceAddCmd.Flags().Int64Var(&instParentID, "parent-id", 0, "Parent reseller instance ID (if child bot)")

	instanceEditCmd.Flags().StringVar(&instName, "name", "", "New instance name")
	instanceEditCmd.Flags().StringVar(&instToken, "token", "", "New bot token")
	instanceEditCmd.Flags().BoolVar(&instActive, "active", true, "Instance active state")
}
