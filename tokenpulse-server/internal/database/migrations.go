// 本文件封装 golang-migrate，并兼容早期 TokenPulse 数据库的迁移记录。
package database

import (
	"errors"
	"fmt"
	"net/url"
	"path/filepath"

	"github.com/golang-migrate/migrate/v4"
	migrateMySQL "github.com/golang-migrate/migrate/v4/database/mysql"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"gorm.io/gorm"
)

// migrationsTable 使用项目专属表名，避免与其他工具的 schema_migrations 冲突。
const migrationsTable = "tokenpulse_schema_migrations"

// ApplyMigrations 使用 golang-migrate 提供迁移锁、脏状态跟踪和顺序执行能力。
// 调用前会接管 TokenPulse 0.1 旧迁移器留下的版本记录。
func ApplyMigrations(db *gorm.DB, directory string) error {
	sqlDB, err := db.DB()
	if err != nil {
		return fmt.Errorf("get migration database: %w", err)
	}
	driver, err := migrateMySQL.WithInstance(sqlDB, &migrateMySQL.Config{MigrationsTable: migrationsTable})
	if err != nil {
		return fmt.Errorf("create migration driver: %w", err)
	}
	absolute, err := filepath.Abs(directory)
	if err != nil {
		return fmt.Errorf("resolve migrations directory: %w", err)
	}
	sourceURL := (&url.URL{Scheme: "file", Path: filepath.ToSlash(absolute)}).String()
	// 使用 file URL 而不是手工拼接路径，以兼容路径中的空格和不同平台分隔符。
	runner, err := migrate.NewWithDatabaseInstance(sourceURL, "mysql", driver)
	if err != nil {
		return fmt.Errorf("create migration runner: %w", err)
	}
	if err := bootstrapLegacyMigrations(db, runner); err != nil {
		return err
	}
	if err := runner.Up(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
		return fmt.Errorf("apply migrations: %w", err)
	}
	return nil
}

// bootstrapLegacyMigrations 将旧 schema_migrations 的最高版本写入新迁移器。
// 新迁移表已有有效版本时直接返回，保证该兼容逻辑只执行一次。
func bootstrapLegacyMigrations(db *gorm.DB, runner *migrate.Migrate) error {
	if db.Migrator().HasTable(migrationsTable) {
		if _, _, err := runner.Version(); !errors.Is(err, migrate.ErrNilVersion) {
			return nil
		}
	}
	if !db.Migrator().HasTable("schema_migrations") {
		return nil
	}
	var version int
	result := db.Raw(`SELECT COALESCE(MAX(CAST(SUBSTRING_INDEX(version, '_', 1) AS UNSIGNED)), 0) FROM schema_migrations`).Scan(&version)
	if result.Error != nil {
		return fmt.Errorf("read legacy migration version: %w", result.Error)
	}
	if version > 0 {
		if err := runner.Force(version); err != nil {
			return fmt.Errorf("bootstrap migration version: %w", err)
		}
	}
	return nil
}
