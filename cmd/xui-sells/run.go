package main

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/spf13/cobra"

	adapterHTTP "xui-sells-v2/internal/adapter/http"
	"xui-sells-v2/internal/adapter/telegram"
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

		fmt.Println("Starting xui-sells-v2 background runtime...")

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

		// 2. Initialize Telegram Bot Supervisor
		supervisor := telegram.NewSupervisor()
		if appStore != nil {
			instances, err := appStore.List(ctx)
			if err == nil {
				for _, inst := range instances {
					if inst.IsActive && inst.BotToken != "" {
						botInstance := telegram.NewBotInstance(inst, telegram.BotDependencies{
							Supervisor: supervisor,
						})
						_ = supervisor.RegisterBot(botInstance)
					}
				}
			}
		}

		_ = supervisor.StartAll(ctx)
		fmt.Printf("🤖 Telegram Bot Supervisor started with %d active bot(s).\n", supervisor.Count())
		fmt.Println("🚀 System ready. Press Ctrl+C to shut down.")

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
