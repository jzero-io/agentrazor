package cmd

import (
	"fmt"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/jzero-io/jzero/core/stores/migrate"
	"github.com/zeromicro/go-zero/core/stores/sqlx"
)

var invalidMigrationTableCharacter = regexp.MustCompile(`[^a-z0-9_]`)

func runMigrations(sqlConf sqlx.SqlConf) error {
	if err := runMigrationSource(sqlConf, migrate.DefaultSource, migrate.DefaultXMigrationsTable); err != nil {
		return fmt.Errorf("run core migrations: %w", err)
	}

	pluginSources, err := filepath.Glob(filepath.Join("plugins", "*", "desc", "sql_migration"))
	if err != nil {
		return fmt.Errorf("discover plugin migrations: %w", err)
	}
	sort.Strings(pluginSources)
	for _, source := range pluginSources {
		pluginName := filepath.Base(filepath.Dir(filepath.Dir(source)))
		tableName := pluginMigrationTable(pluginName)
		if err := runMigrationSource(sqlConf, "file://"+filepath.ToSlash(source), tableName); err != nil {
			return fmt.Errorf("run plugin %q migrations: %w", pluginName, err)
		}
	}
	return nil
}

func runMigrationSource(sqlConf sqlx.SqlConf, source, tableName string) error {
	m, err := migrate.NewMigrate(
		sqlConf,
		migrate.WithSource(source),
		migrate.WithSourceAppendDriver(true),
		migrate.WithXMigrationsTable(tableName),
	)
	if err != nil {
		return err
	}
	defer m.Close()
	return m.Up()
}

func pluginMigrationTable(pluginName string) string {
	name := strings.ToLower(pluginName)
	name = strings.ReplaceAll(name, "-", "_")
	name = invalidMigrationTableCharacter.ReplaceAllString(name, "_")
	name = strings.Trim(name, "_")
	return name + "_schema_migrations"
}
