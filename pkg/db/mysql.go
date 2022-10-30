// Copyright 2020 Lingfei Kong <colin404@foxmail.com>. All rights reserved.
// Use of this source code is governed by a MIT style
// license that can be found in the LICENSE file.

package db

import (
	"fmt"
	"time"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// Options defines optsions for mysql database.
// Options 定义mysql数据库使用的选项
type Options struct {
	Host                  string           // msyql host地址
	Username              string           // 访问mysql的username
	Password              string           // 访问mysql的password
	Database              string           // 要访问的数据库
	MaxIdleConnections    int              // mysql的最大空闲连接数，推荐100
	MaxOpenConnections    int              // mysql的最大连接数，推荐100
	MaxConnectionLifeTime time.Duration    //mysql的空闲连接最大存活时间，推荐10s
	LogLevel              int              // 日志等级
	Logger                logger.Interface // 日志接口
}

// New create a new gorm db instance with the given options.
// New 使用指定的Options创建一个gorm.DB实例
func New(opts *Options) (*gorm.DB, error) {
	// 创建数据库dsn
	dsn := fmt.Sprintf(`%s:%s@tcp(%s)/%s?charset=utf8&parseTime=%t&loc=%s`,
		opts.Username,
		opts.Password,
		opts.Host,
		opts.Database,
		true,
		"Local")

	// 创建数据库连接池
	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{
		Logger: opts.Logger,
	})
	if err != nil {
		return nil, err
	}

	sqlDB, err := db.DB()
	if err != nil {
		return nil, err
	}

	// SetMaxOpenConns sets the maximum number of open connections to the database.
	// 设置MySQL最大连接数
	sqlDB.SetMaxOpenConns(opts.MaxOpenConnections)

	// SetConnMaxLifetime sets the maximum amount of time a connection may be reused.
	// 设置MySQL空闲连接最大存活时间
	sqlDB.SetConnMaxLifetime(opts.MaxConnectionLifeTime)

	// SetMaxIdleConns sets the maximum number of connections in the idle connection pool.
	// 设置最大空闲连接数
	sqlDB.SetMaxIdleConns(opts.MaxIdleConnections)

	return db, nil
}
