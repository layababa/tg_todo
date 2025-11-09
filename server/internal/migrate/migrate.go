package migrate

import (
	"context"
	"database/sql"
	"io/fs"
	"log"
	"sort"

	"github.com/zz/tg_todo/server/migrations"
)

// Apply runs embedded SQL migrations sequentially (sorted by filename).
func Apply(ctx context.Context, db *sql.DB) error {
	files, err := fs.Glob(migrations.Files, "*.sql")
	if err != nil {
		return err
	}
	sort.Strings(files)

	for _, file := range files {
		sqlBytes, err := migrations.Files.ReadFile(file)
		if err != nil {
			return err
		}
		if _, err := db.ExecContext(ctx, string(sqlBytes)); err != nil {
			return err
		}
		log.Printf("migration applied: %s", file)
	}
	return nil
}
