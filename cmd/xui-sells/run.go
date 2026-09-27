package main

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/spf13/cobra"

	adapterHTTP "xui-sells-v2/internal/adapter/http"
	"xui-sells-v2/internal/adapter/telegram"
	"xui-sells-v2/internal/app/provisioning"
	"xui-sells-v2/internal/domain"
	"xui-sells-v2/internal/infra/xui"
	"xui-sells-v2/web"
)

var (
	runHTTPPort int
	runBindAddr string
)

var runCmd = &cobra.Command{
	Use:   "run",
	Short: "Start the background services: HTTP web panel and Telegram bot supervisor",
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx, stop := signal.NotifyContext(cmd.Context(), os.Interrupt, syscall.SIGTERM)
		defer stop()

		if envPort := os.Getenv("PORT"); envPort != "" && runHTTPPort == 8080 {
			if p, err := strconv.Atoi(envPort); err == nil {
				runHTTPPort = p
			}
		}

		// 1. Initialize HTTP Web Panel Server
		httpCfg := adapterHTTP.Config{
			ChildBotStore: appStore,
			WebFS:         web.DistFS,
		}
		httpServerAdapter := adapterHTTP.NewServer(httpCfg)

		addr := fmt.Sprintf("%s:%d", runBindAddr, runHTTPPort)
		srv := &http.Server{
			Addr:         addr,
			Handler:      httpServerAdapter.Router(),
			ReadTimeout:  15 * time.Second,
			WriteTimeout: 15 * time.Second,
		}

		go func() {
			fmt.Printf("🌐 Reseller Web Panel listening at http://%s\n", addr)
			if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
				fmt.Fprintf(os.Stderr, "HTTP server error: %v\n", err)
			}
		}()

		// 2. Initialize Telegram Bot Supervisor with dynamic instance synchronization
		supervisor := telegram.NewSupervisor()
		_ = supervisor.StartAll(ctx)

		syncBots := func() {
			if appStore == nil {
				return
			}
			instances, err := appStore.List(ctx)
			if err != nil {
				return
			}

			activeIDs := make(map[int64]bool)
			for _, inst := range instances {
				if !inst.IsActive || inst.BotToken == "" {
					continue
				}
				activeIDs[inst.ID] = true
				existingBot, running := supervisor.GetBot(inst.ID)
				if !running {
					botInst := buildBotInstance(inst, supervisor)
					if err := supervisor.RegisterBot(botInst); err == nil {
						fmt.Printf("🤖 Bot instance '%s' (ID %d) started.\n", inst.Name, inst.ID)
					}
				} else if existingBot.Instance.BotToken != inst.BotToken || existingBot.Instance.PanelURL != inst.PanelURL || existingBot.Instance.GroupName != inst.GroupName {
					_ = supervisor.StopBot(inst.ID)
					botInst := buildBotInstance(inst, supervisor)
					_ = supervisor.RegisterBot(botInst)
					fmt.Printf("🔄 Bot instance '%s' (ID %d) reloaded with new configuration.\n", inst.Name, inst.ID)
				}
			}

			for _, runningID := range supervisor.ListRunning() {
				if !activeIDs[runningID] {
					_ = supervisor.StopBot(runningID)
					fmt.Printf("🛑 Bot instance ID %d stopped.\n", runningID)
				}
			}
		}

		syncBots()
		fmt.Printf("🤖 Telegram Bot Supervisor initialized with %d active bot(s).\n", supervisor.Count())
		fmt.Println("🚀 System ready. Press Ctrl+C to shut down.")

		// Background watcher to dynamically load instances added/modified via CLI
		go func() {
			ticker := time.NewTicker(3 * time.Second)
			defer ticker.Stop()
			for {
				select {
				case <-ctx.Done():
					return
				case <-ticker.C:
					syncBots()
				}
			}
		}()

		// Wait for termination signal
		<-ctx.Done()
		fmt.Println("\nReceived shutdown signal. Stopping services gracefully...")

		// Graceful shutdown with 5-second timeout
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		_ = srv.Shutdown(shutdownCtx)
		supervisor.StopAll()

		fmt.Println("👋 All services stopped cleanly.")
		return nil
	},
}

func init() {
	runCmd.Flags().IntVar(&runHTTPPort, "http-port", 8080, "Port for the HTTP web panel")
	runCmd.Flags().StringVar(&runBindAddr, "bind", "0.0.0.0", "Network interface address to bind to")
}

func buildBotInstance(inst domain.Instance, supervisor *telegram.Supervisor) *telegram.BotInstance {
	sender := telegram.NewHTTPSender(inst.BotToken)
	xuiClient := xui.NewClient(inst.PanelURL, inst.PanelAPIKey)
	provSvc := provisioning.NewService(xuiClient)

	deps := telegram.BotDependencies{
		Sender:     sender,
		ProvSvc:    provSvc,
		Supervisor: supervisor,
	}

	return telegram.NewBotInstance(inst, deps)
}
