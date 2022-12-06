package user

import (
	"crack_front/src/common"
	"crack_front/src/config"
	"errors"
	"github.com/jinzhu/gorm"
	_ "github.com/jinzhu/gorm/dialects/mysql"
	"regexp"
)

var (
	DB *gorm.DB
)

func InitMySQL() (err error) {
	DB, err = gorm.Open("mysql", config.Cfg.MySQL_DataSourceName)
	if err != nil {
		panic(err)
	}
	if !DB.HasTable(&common.User{}) {
		DB = DB.CreateTable(&common.User{})
	}
	return DB.DB().Ping()
}

func CloseMySQL() {
	err := DB.Close()
	if err != nil {
		panic(err)
	}
}

func VerifyUserName(UserName string) (ok bool){
	ok, _ = regexp.MatchString("^[a-zA-Z0-9_]{8,16}$", UserName);
	return ok
}

func VerifyUserEmail(UserEmail string) (ok bool) {
	ok, _ = regexp.MatchString("^([a-zA-Z0-9_]+\\.)*([a-zA-Z0-9_]+)@([a-zA-Z0-9]+\\.)+([a-zA-Z0-9]+)$", UserEmail)
	return ok
}

func VerifyUserPassword(UserPassword string) (ok bool) {
	ok, _ = regexp.MatchString("^[a-zA-Z0-9]{8,16}$", UserPassword)
	return
}

func SelectUserByName(UserName string) (curUser *common.User, err error) {
	curUser = &common.User{}
	if err = DB.Where("user_name=?", UserName).First(curUser).Error; err != nil {
		return nil, err
	}
	return
}

func SelectUserByEmail(UserEmail string) (curUser *common.User, err error) {
	curUser = &common.User{}
	if err = DB.Where("user_email=?", UserEmail).First(curUser).Error; err != nil {
		return nil, err
	}
	return
}

func InsertUser(newUser *common.User) (err error) {
	return DB.Create(&newUser).Error
}

func EmailExisted(UserEmail string) (existed bool, err error) {
	var (
		count		int
	)
	if err = DB.Model(&common.User{}).Where("user_email=?", UserEmail).Count(&count).Error; err != nil {
		return true, err
	}
	return count != 0, nil
}

func NameExisted(UserName string) (existed bool, err error) {
	if _, err = SelectUserByName(UserName); err != nil {
		return false, nil
	}
	return true, errors.New("用户名已经被注册")
}
