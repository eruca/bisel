package types

import "gorm.io/gorm"

// DB ...
type DB struct {
	Gorm *gorm.DB
}
