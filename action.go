package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/alex-cos/sheriff/config"
	"github.com/alex-cos/sheriff/nodes"
	"github.com/urfave/cli/v3"
)

// nolint: contextcheck
func action(c context.Context, cmd *cli.Command) error {
	configfile := cmd.String("config")
	config := config.Config{}

	err := config.Load(configfile)
	if err != nil {
		return err
	}
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: logLevel(config.LogLevel)}))
	slog.SetDefault(logger)

	root, err := nodes.NewServiceNodes(logger, &config)
	if err != nil {
		return err
	}
	stopTimeout := config.StopTimeout
	if stopTimeout.Seconds() < 5 {
		stopTimeout = 5 * time.Second
	}

	err = root.Start(c)
	if err != nil {
		stopCtx, stopCancel := context.WithTimeout(context.Background(), stopTimeout)
		defer stopCancel()
		root.Stop(stopCtx)
		return err
	}

	period := config.Monitor.Period
	if period.Seconds() <= 10 {
		period = time.Minute
	}

	monitor := NewMonitor(logger, root, period, config.Monitor.Restart)

	monitor.Start()

	waitForSignal()

	monitor.Stop()

	stopCtx, stopCancel := context.WithTimeout(context.Background(), stopTimeout)
	defer stopCancel()
	logger.Info("shutting down")
	root.Stop(stopCtx)
	logger.Info("shutdown complete")

	return nil
}

// waitForSignal blocks until SIGINT or SIGTERM is received.
func waitForSignal() {
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
	<-sig
}

// logLevel converts a string level ("debug", "info", "warn", "error") to the
// corresponding slog.Level.
func logLevel(lvl string) slog.Level {
	switch lvl {
	case "debug":
		return slog.LevelDebug
	case "warn":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}
