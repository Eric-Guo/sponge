// todo generate model code to here
// delete the templates code start

package model

import (
	"github.com/Eric-Guo/sponge/pkg/sgorm"
)

// UserExample object fields mapping table
type UserExample struct {
	sgorm.Model `gorm:"embedded"`

	Name     string `gorm:"column:name;NOT NULL" json:"name"`         // username
	Password string `gorm:"column:password;NOT NULL" json:"password"` // password
	Email    string `gorm:"column:email;NOT NULL" json:"email"`       // email
	Phone    string `gorm:"column:phone;NOT NULL" json:"phone"`       // phone number
	Avatar   string `gorm:"column:avatar;NOT NULL" json:"avatar"`     // avatar
	Age      int    `gorm:"column:age;NOT NULL" json:"age"`           // age
	Gender   int    `gorm:"column:gender;NOT NULL" json:"gender"`     // gender, 1:Male, 2:Female, other values:unknown
	Status   int    `gorm:"column:status;NOT NULL" json:"status"`     // account status, 1:inactive, 2:activated, 3:blocked
	LoginAt  int64  `gorm:"column:login_at;NOT NULL" json:"loginAt"`  // login timestamp
}

// TableName get table name
func (table *UserExample) TableName() string {
	return "user_example"
}

// UserExampleColumnNames Whitelist for custom query fields to prevent sql injection attacks
var UserExampleColumnNames = map[string]bool{
	"id":         true,
	"created_at": true,
	"updated_at": true,
	"deleted_at": true,
	"name":       true,
	"password":   true,
	"email":      true,
	"phone":      true,
	"avatar":     true,
	"age":        true,
	"gender":     true,
	"status":     true,
	"login_at":   true,
}

// delete the templates code end
