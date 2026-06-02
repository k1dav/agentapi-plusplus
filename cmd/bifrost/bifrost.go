package bifrost

import (
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/coder/agentapi/internal/routing"
	"github.com/coder/agentapi/internal/server"
	"github.com/spf13/cobra"
	"golang.org/x/xerrors"
)

const (
	FlagPort     = "port"
	FlagCliproxy = "cliproxy"
)

// CreateBifrostCmd builds the `agentapi bifrost` subcommand, which runs the
// Bifrost agent-routing server (the cliproxy+bifrost extension) on top of the
// agentapi HTTP API. It supersedes the standalone stdlib-flag entrypoint that
// previously lived in cmd/agentapi/main.go, folding that capability into the
// single cobra command tree so the project ships one binary with one flag
// convention.
//
// Flag values are read directly from the command's flag set (not the shared
// global viper instance) to avoid colliding with the `server` subcommand,
// which binds a "port" key to viper with a different default.
func CreateBifrostCmd() *cobra.Command {
	bifrostCmd := &cobra.Command{
		Use:   "bifrost",
		Short: "Run the Bifrost agent-routing server",
		Long: "Run the Bifrost agent-routing server, connecting the agentapi HTTP " +
			"API to a cliproxy+bifrost backend for multi-model routing.",
		RunE: func(cmd *cobra.Command, args []string) error {
			logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

			port, err := cmd.Flags().GetInt(FlagPort)
			if err != nil {
				return xerrors.Errorf("failed to read --%s: %w", FlagPort, err)
			}
			cliproxyURL, err := cmd.Flags().GetString(FlagCliproxy)
			if err != nil {
				return xerrors.Errorf("failed to read --%s: %w", FlagCliproxy, err)
			}

			router, err := routing.NewAgentBifrost(cliproxyURL)
			if err != nil {
				return xerrors.Errorf("failed to initialize agent bifrost: %w", err)
			}

			srv := server.New(port, router)

			quit := make(chan os.Signal, 1)
			signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
			go func() {
				<-quit
				logger.Info("Shutting down agentapi bifrost server")
				srv.Shutdown()
			}()

			logger.Info("AgentAPI bifrost server starting", "port", port, "cliproxy", cliproxyURL)
			if err := srv.Start(); err != nil {
				return xerrors.Errorf("server error: %w", err)
			}
			return nil
		},
	}

	bifrostCmd.Flags().IntP(FlagPort, "p", 8318, "Port to run the bifrost routing server on")
	bifrostCmd.Flags().String(FlagCliproxy, "http://127.0.0.1:8317", "cliproxy+bifrost backend URL")

	return bifrostCmd
}
