package main

import (
	"context"
	"log/slog"
	"os"
	"time"

	"cally/internal/config"
	"cally/internal/database"
	"cally/internal/repository"
	"cally/internal/room"
	"cally/internal/server"
	"cally/internal/service"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))
	slog.SetDefault(logger)

	cfg, err := config.Load()
	if err != nil {
		slog.Error("failed to load configuration", "error", err)
		os.Exit(1)
	}

	// 1. Initialize PostgreSQL database connection pool
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	var repos *repository.Repositories

	db, err := database.Connect(ctx, cfg)
	if err != nil {
		slog.Warn("PostgreSQL connection skipped/unavailable, falling back to in-memory store", "reason", err.Error())
		repos = &repository.Repositories{
			Rooms:        repository.NewMemoryRoomRepository(),
			Users:        repository.NewMemoryUserRepository(),
			Participants: repository.NewMemoryParticipantRepository(),
			CallLogs:     repository.NewMemoryCallLogRepository(),
		}
	} else if db != nil {
		defer func() {
			_ = db.Close()
		}()
		slog.Info("PostgreSQL repository persistence initialized")
		repos = &repository.Repositories{
			Rooms:        repository.NewPostgresRoomRepository(db.SQLDB),
			Users:        repository.NewPostgresUserRepository(db.SQLDB),
			Participants: repository.NewPostgresParticipantRepository(db.SQLDB),
			CallLogs:     repository.NewPostgresCallLogRepository(db.SQLDB),
		}
	} else {
		repos = &repository.Repositories{
			Rooms:        repository.NewMemoryRoomRepository(),
			Users:        repository.NewMemoryUserRepository(),
			Participants: repository.NewMemoryParticipantRepository(),
			CallLogs:     repository.NewMemoryCallLogRepository(),
		}
	}

	manager := room.NewManager()
	roomService := service.NewRoomService(manager, repos, cfg)

	// Clean up idle rooms periodically
	go func() {
		ticker := time.NewTicker(15 * time.Minute)
		defer ticker.Stop()
		for range ticker.C {
			manager.CleanIdleRooms(time.Duration(cfg.RoomIdleTimeoutMin) * time.Minute)
		}
	}()

	srv := server.NewServer(cfg, manager, roomService)
	if err := srv.Start(); err != nil {
		slog.Error("server startup error", "error", err)
		os.Exit(1)
	}
}
