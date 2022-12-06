package common

type User struct {
	UserId 					uint 		`gorm:"column:user_id;primary_key;auto_increment" json:"user_id"`			// 用户id
	UserName 				string		`gorm:"column:user_name;not null;unique" json:"user_name"`         			// 用户名
	UserEmail 				string 		`gorm:"column:user_email;not null;unique" json:"user_email"`				// 用户邮箱
	UserPassword 			string		`gorm:"column:user_password;not null" json:"user_password"`					// 用户密码
}
