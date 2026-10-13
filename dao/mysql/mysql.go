package mysql

import (
	"fmt"
	"riff/conf"
	"riff/models"

	"go.uber.org/zap"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

var db *gorm.DB

func InitMySQL(cfg *conf.MySQLConfig) (err error) {
	dsn := fmt.Sprintf(
		"%s:%s@tcp(%s:%d)/%s?charset=utf8mb4&parseTime=True&loc=Local",
		cfg.Username,
		cfg.Password,
		cfg.Host,
		cfg.Port,
		cfg.DBname,
	)
	conn, err := gorm.Open(mysql.Open(dsn), &gorm.Config{})
	if err != nil {
		return err
	}

	db = conn

	err = db.AutoMigrate(&models.User{}, &models.Community{}, &models.Post{})
	if err != nil {
		zap.L().Error("db.AutoMigrate failed")
		return
	}

	return nil
}

func DB() *gorm.DB {
	return db
}

func Close() error {
	if db == nil {
		return nil
	}

	sqlDB, err := db.DB()
	if err != nil {
		return err
	}

	return sqlDB.Close()
}
