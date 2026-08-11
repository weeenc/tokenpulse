// Package database 负责创建 MySQL/GORM 连接并管理数据库迁移。
package database

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"time"

	mysqlDriver "github.com/go-sql-driver/mysql"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// Open 解析 DSN、配置安全的 SQL 日志并返回可复用的 GORM 连接。
func Open(dsn string, production bool) (*gorm.DB, error) {
	dsnConfig, err := mysqlDriver.ParseDSN(dsn)
	if err != nil {
		return nil, fmt.Errorf("parse mysql DSN: %w", err)
	}
	dsnConfig.MultiStatements = true
	// 迁移文件可能包含多条语句，因此仅在受控服务端 DSN 上启用多语句。
	dsn = dsnConfig.FormatDSN()
	level := logger.Warn
	if production {
		// 生产环境只记录 SQL 错误，避免高频慢查询日志造成额外开销。
		level = logger.Error
	}
	gormLogger := logger.New(log.New(os.Stdout, "", log.LstdFlags), logger.Config{
		SlowThreshold:             time.Second,
		LogLevel:                  level,
		IgnoreRecordNotFoundError: true,
		ParameterizedQueries:      true,
		Colorful:                  false,
	})
	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{Logger: gormLogger, NowFunc: func() time.Time { return time.Now().UTC() }})
	if err != nil {
		return nil, fmt.Errorf("open mysql: %w", err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("get database pool: %w", err)
	}
	configurePool(sqlDB)
	return db, nil
}

// configurePool 设置连接池上限、空闲连接数以及连接回收周期。
func configurePool(db *sql.DB) {
	db.SetMaxOpenConns(50)
	db.SetMaxIdleConns(10)
	db.SetConnMaxLifetime(30 * time.Minute)
	db.SetConnMaxIdleTime(5 * time.Minute)
}
