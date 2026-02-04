package common

import (
	"fmt"
	"log"
	"time"

	"github.com/KATOmemorial/cronyx/internal/model"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

var DB *gorm.DB

func InitDB() {
	dsn := "root:root@tcp(localhost:3306)/cronyx?charset=utf8mb4&parseTime=True&loc=Local"

	var err error
	DB, err = gorm.Open(mysql.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Fatalf("Failed to connect to MySQL: %v", err)
	}

	err = DB.AutoMigrate(&model.JobInfo{}, &model.JobLog{})
	if err != nil {
		log.Fatalf("Failed to migrate datebase: %v", err)
	}

	sqlDB, _ := DB.DB()
	sqlDB.SetMaxIdleConns(10)
	sqlDB.SetMaxOpenConns(100)
	sqlDB.SetConnMaxLifetime(time.Hour)

	fmt.Println("MySQL Connected & Migrated successfully!")
}
