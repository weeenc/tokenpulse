// Package repository 提供带请求上下文和事务边界的数据库访问入口。
package repository

import (
	"context"

	"gorm.io/gorm"
)

// Store 是业务服务访问数据库的统一边界，集中处理上下文传播和事务作用域，
// 从而避免 Handler 与 Service 直接管理底层数据库连接生命周期。
type Store struct {
	db *gorm.DB // db 已绑定当前请求上下文或事务。
}

// New 使用给定 GORM 连接创建数据仓储。
func New(db *gorm.DB) *Store { return &Store{db: db} }

// WithContext 返回绑定请求上下文的新 Store，不修改原 Store。
func (s *Store) WithContext(ctx context.Context) *Store {
	return &Store{db: s.db.WithContext(ctx)}
}

// Query 返回当前上下文及事务范围内的 GORM 查询对象。
func (s *Store) Query() *gorm.DB { return s.db }

// Transaction 在单个数据库事务中执行 operation，返回错误时自动回滚。
func (s *Store) Transaction(operation func(*Store) error) error {
	return s.db.Transaction(func(tx *gorm.DB) error { return operation(New(tx)) })
}
