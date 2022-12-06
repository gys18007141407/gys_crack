package mysql

import "github.com/jinzhu/gorm"

var (
	DB *gorm.DB
)

func InitDB() (err error) {
	dsn := "root:@(127.0.0.1:3306)/db1?charset=utf8mb4&parseTime=True&loc=Local"
	DB, err = gorm.Open("mysql", dsn)
	if err != nil {
		panic(err)
	}
	return DB.DB().Ping()
}

func Close() {
	err := DB.Close()
	if err != nil {
		panic(err)
	}
}