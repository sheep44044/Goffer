package db

import (
	"Goffer/app/rpc/user/config"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	gormopentracing "gorm.io/plugin/opentracing"
)

func Init(cfg *config.Config) (*gorm.DB, error) {
	db, err := gorm.Open(mysql.Open(cfg.DB.DSN()),
		&gorm.Config{
			PrepareStmt:            true,
			SkipDefaultTransaction: true,
		},
	)

	if err != nil {
		// 将错误返回，由上层决定是否 panic 或重试
		return nil, err
	}

	if err = db.Use(gormopentracing.New()); err != nil {
		return nil, err
	}

	return db, nil
}
