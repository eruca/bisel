package btypes

import (
	"database/sql"
	"encoding/json"
	"strings"

	"gorm.io/gorm"
)

type NullString struct {
	sql.NullString
}

// MarshalJSON impl json.MarshalJSON
func (ns NullString) MarshalJSON() ([]byte, error) {
	if ns.Valid {
		return json.Marshal(ns.String)
	}
	return []byte("null"), nil
}

// UnmarshalJSON ...
func (ns *NullString) UnmarshalJSON(data []byte) error {
	var s *string
	if err := json.Unmarshal(data, &s); err != nil {
		return err
	}
	if s != nil {
		ns.Valid = true
		ns.String = *s
	} else {
		ns.Valid = false
	}
	return nil
}

// func QueryAssist(db *gorm.DB, tabler Tabler, pc *ParamsContext, total *int64, list interface{}, omits ...string) {
// 	tx := db.Begin()
// 	defer tx.Commit()

// 	tableName := tabler.TableName()
// 	tx = tx.Table(tableName)

// 	if len(pc.QueryParams.Conds) > 0 {
// 		// todo 还需对Conds重新设计
// 		tx = tx.Where(strings.Join(pc.QueryParams.Conds, " AND "))
// 	}
// 	tx.Raw(fmt.Sprintf("SELECT COUNT(*) FROM %s WHERE deleted_at IS NULL", tableName)).Scan(total)

// 	if err := tx.Where("1 = 1").Order(pc.QueryParams.Orderby).
// 		Offset(int(pc.QueryParams.Offset)).
// 		Limit(int(pc.QueryParams.Size)).
// 		Omit(omits...).
// 		Find(list).Error; err != nil {
// 		tx.Rollback()
// 		panic(err)
// 	}
// }

func QueryAssist(db *gorm.DB, tabler Tabler, pc *ParamsContext, total *int64, list interface{}, omits ...string) {
	tx := db.Begin()
	defer tx.Commit()

	tableName := tabler.TableName()
	tx = tx.Table(tableName)

	if len(pc.QueryParams.Conds) > 0 {
		// todo 还需对Conds重新设计
		tx = tx.Where(strings.Join(pc.QueryParams.Conds, " AND "))
	}

	if err := tx.Where("1 = 1").Order(pc.QueryParams.Orderby).
		Offset(int(pc.QueryParams.Offset)).
		Limit(int(pc.QueryParams.Size)).
		Omit(omits...).
		Count(total).
		Find(list).Error; err != nil {
		tx.Rollback()
		panic(err)
	}
}
