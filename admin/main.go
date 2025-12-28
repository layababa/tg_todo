package main

import (
	"io/ioutil"
	"log"
	"os"
	"os/signal"

	_ "github.com/GoAdminGroup/go-admin/adapter/gin" // Gin 适配器
	_ "github.com/lib/pq"

	"github.com/GoAdminGroup/go-admin/engine"
	"github.com/GoAdminGroup/go-admin/modules/config"
	"github.com/GoAdminGroup/go-admin/modules/language"
	"github.com/GoAdminGroup/go-admin/plugins/admin"
	"github.com/GoAdminGroup/themes/adminlte"
	"github.com/gin-gonic/gin"

	"tg_todo_admin/tables"
)

func main() {
	gin.SetMode(gin.ReleaseMode)
	gin.DefaultWriter = ioutil.Discard

	r := gin.Default()

	eng := engine.Default()

	// 优先使用 DSN，否则拼接单独字段
	dsn := getEnv("DATABASE_DSN", "")
	if dsn == "" {
		dsn = "host=" + getEnv("DB_HOST", "localhost") +
			" port=" + getEnv("DB_PORT", "5432") +
			" user=" + getEnv("DB_USER", "tg_todo") +
			" password=" + getEnv("DB_PASSWORD", "change-me") +
			" dbname=" + getEnv("DB_NAME", "tg_todo") +
			" sslmode=disable"
	}

	cfg := config.Config{
		Databases: config.DatabaseList{
			"default": {
				Driver:       config.DriverPostgresql,
				Dsn:          dsn,
				MaxIdleConns: 5,
				MaxOpenConns: 10,
			},
		},
		UrlPrefix: "admin",
		Store: config.Store{
			Path:   "./uploads",
			Prefix: "uploads",
		},
		Language:     language.CN,
		IndexUrl:     "/",
		Debug:        getEnv("DEBUG", "false") == "true",
		AccessLogOff: true,
		ColorScheme:  adminlte.ColorschemeSkinBlack,
		Title:        "TG TODO 管理后台",
		LoginTitle:   "TG TODO Admin",
		LoginLogo:    "<b>TG</b> TODO",
		Logo:         "<b>TG</b> TODO",
		MiniLogo:     "<b>T</b>",
	}

	adminPlugin := admin.NewAdmin(tables.Generators)

	if err := eng.AddConfig(&cfg).
		AddGenerators(tables.Generators).
		AddPlugins(adminPlugin).
		Use(r); err != nil {
		panic(err)
	}

	r.Static("/uploads", "./uploads")

	eng.HTML("GET", "/admin", tables.GetDashboardContent)

	port := getEnv("PORT", "9033")
	log.Printf("TG TODO Admin starting on http://0.0.0.0:%s/admin", port)

	go func() {
		_ = r.Run(":" + port)
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt)
	<-quit
	log.Println("Shutting down server...")
}

func getEnv(key, defaultValue string) string {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}
	return value
}
