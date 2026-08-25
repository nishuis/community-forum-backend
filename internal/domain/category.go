package domain

import "gorm.io/gorm"

type Category struct {
	gorm.Model
}

func (Category) TableName() string {
	return "categorys"
}
