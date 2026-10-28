// Command migrate applies or reverts the database migrations.
//
// Use: migrate up|down
// The settings come from environment variables: the POSTGRES_ variables, MIGRATIONS_DIR and LOG_LEVEL.
// After "up", the command makes the first Admin from ADMIN_EMAIL and ADMIN_PASSWORD. It does nothing
// when an Admin with this email address exists.
package main

import (
	"context"
	"errors"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/rudradcruze/go-react-template/backend/internal/bootstrap"
	"github.com/rudradcruze/go-react-template/backend/internal/platform/config"
	"github.com/rudradcruze/go-react-template/backend/internal/platform/database"
	"github.com/rudradcruze/go-react-template/backend/internal/platform/logger"
)

func main() {
	if err := run(os.Args[1:]); err != nil {
		os.Exit(1)
	}
}

func run(args []string) error {
	cfg, err := config.LoadMigrate()
	if err != nil {
		logger.New(slog.LevelInfo, os.Stderr).Error("The settings are not valid.", slog.Any("error", err))
		return err
	}
	log := logger.New(cfg.LogLevel, os.Stdout)

	if len(args) != 1 || (args[0] != "up" && args[0] != "down") {
		log.Error("The command is not valid. Use: migrate up|down.", slog.Any("arguments", args))
		return errors.New("usage")
	}
	command := args[0]

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	db, err := database.Open(ctx, cfg.Postgres)
	if err != nil {
		log.Error("The migration cannot start.", slog.Any("error", err))
		return err
	}
	sqlDB, err := db.DB()
	if err != nil {
		log.Error("The migration cannot start.", slog.Any("error", err))
		return err
	}
	defer func() { _ = sqlDB.Close() }()

	if command == "up" {
		err = database.MigrateUp(ctx, sqlDB, cfg.Dir)
	} else {
		err = database.MigrateDown(ctx, sqlDB, cfg.Dir)
	}
	if err != nil {
		log.Error("The migration failed.", slog.String("command", command), slog.Any("error", err))
		return err
	}
	log.Info("The migration command is complete.", slog.String("command", command), slog.String("folder", cfg.Dir))
	if command != "up" {
		return nil
	}

	created, err := bootstrap.NewSeedAdmin(db).Execute(ctx, cfg.AdminEmail, cfg.AdminPassword)
	if err != nil {
		log.Error("The first Admin cannot be made. Set ADMIN_EMAIL to a valid email address, "+
			"and ADMIN_PASSWORD to a password of 12 to 72 bytes.", slog.Any("error", err))
		return err
	}
	log.Info("The Admin check is complete.", slog.Bool("created", created))
	return nil
}
