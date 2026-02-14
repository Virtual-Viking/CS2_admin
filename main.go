package main

import (
	"embed"
	"fmt"
	"os"

	"cs2admin/internal/config"
	"cs2admin/internal/models"
	"cs2admin/internal/pkg/crypto"
	"cs2admin/internal/pkg/logger"

	"github.com/glebarez/sqlite"
	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
	wailsruntime "github.com/wailsapp/wails/v2/pkg/options/windows"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"
)

//go:embed all:frontend/dist
var assets embed.FS

func main() {
	// 1. Load application config
	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to load config: %v\n", err)
		os.Exit(1)
	}

	// 2. Initialize logging
	debug := cfg.LogLevel == "debug"
	logger.Init(cfg.LogDir, debug)
	logger.Log.Info().Str("version", appVersion).Msg("Starting CS2 Admin")

	// 3. Derive encryption key
	encKey, err := crypto.DeriveKey()
	if err != nil {
		logger.Log.Fatal().Err(err).Msg("Failed to derive encryption key")
	}

	// 4. Open SQLite database
	db, err := gorm.Open(sqlite.Open(cfg.GetDBPath()), &gorm.Config{
		Logger: gormlogger.Default.LogMode(gormlogger.Silent),
	})
	if err != nil {
		logger.Log.Fatal().Err(err).Msg("Failed to open database")
	}
	logger.Log.Info().Str("path", cfg.GetDBPath()).Msg("Database opened")

	// 5. Run auto-migrations
	if err := models.AutoMigrate(db); err != nil {
		logger.Log.Fatal().Err(err).Msg("Database migration failed")
	}
	logger.Log.Info().Msg("Database migrations complete")

	// 6. Create application
	app := NewApp(cfg, db, encKey)

	// 7. Run Wails
	err = wails.Run(&options.App{
		Title:     "CS2 Admin",
		Width:     1280,
		Height:    800,
		MinWidth:  960,
		MinHeight: 600,
		AssetServer: &assetserver.Options{
			Assets: assets,
		},
		BackgroundColour: &options.RGBA{R: 15, G: 15, B: 20, A: 1},
		OnStartup:        app.startup,
		OnDomReady:        app.domReady,
		OnBeforeClose:     app.beforeClose,
		OnShutdown:        app.shutdown,
		Windows: &wailsruntime.Options{
			WebviewIsTransparent: false,
			WindowIsTranslucent:  false,
			Theme:                wailsruntime.Dark,
		},
		Bind: []interface{}{
			app,
		},
	})

	if err != nil {
		logger.Log.Fatal().Err(err).Msg("Wails runtime error")
	}
}
