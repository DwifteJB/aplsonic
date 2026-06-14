package schema

import "gorm.io/gorm"


type AdminConfig struct {
	gorm.Model
	PasswordHash  string `gorm:"type:varchar(255)"` // SHA256(password + Salt)
	Salt          string `gorm:"type:varchar(255)"`
	SessionSecret string `gorm:"type:varchar(255)"` // HMAC key
}

func init() {
	AllModels = append(AllModels, &AdminConfig{})
}
