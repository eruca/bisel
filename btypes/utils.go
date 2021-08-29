package btypes

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"reflect"
	"strings"

	"gorm.io/gorm"
)

// ************************************ Pairs *********************************
type Pairs []Pair

func (ps *Pairs) Push(key string, value interface{}) {
	*ps = append(*ps, Pair{Key: key, Value: value})
}

type Pair struct {
	Key   string      `json:"key,omitempty"`
	Value interface{} `json:"value,omitempty"`
}

type PairStringer struct {
	Key   string       `json:"key,omitempty"`
	Value fmt.Stringer `json:"value,omitempty"`
}

// ValueString 直接将string实现fmt.Stringer
type ValueString string

func (vs ValueString) String() string {
	return string(vs)
}

func QueryAssist(db *gorm.DB, tabler Tabler, queryParam *QueryParam, total *int64, list interface{}, omits ...string) {
	tx := db.Begin()
	defer tx.Commit()

	tableName := tabler.TableName()

	tx = tx.Table(tableName)

	// 所有where合在一起的从句
	var conditions string
	if len(queryParam.Conds) > 0 {
		conditions = strings.Join(queryParam.Conds, " AND ")
		tx = tx.Where(conditions)
		if err := tx.Raw(fmt.Sprintf("SELECT COUNT(*) FROM %s WHERE deleted_at IS NULL AND %s", tableName, conditions)).Scan(total).Error; err != nil {
			tx.Rollback()
			panic(err)
		}
	} else {
		if err := tx.Raw(fmt.Sprintf("SELECT COUNT(*) FROM %s WHERE deleted_at IS NULL", tableName)).Scan(total).Error; err != nil {
			tx.Rollback()
			panic(err)
		}
	}

	if queryParam.Size <= 0 {
		if err := tx.Order(queryParam.Orderby).
			Omit(omits...).
			Find(list).Error; err != nil {
			tx.Rollback()
			panic(err)
		}
	} else {
		if err := tx.Order(queryParam.Orderby).
			Offset(int(queryParam.Offset)).
			Limit(int(queryParam.Size)).
			Omit(omits...).
			Find(list).Error; err != nil {
			tx.Rollback()
			panic(err)
		}
	}
}

func IsExist(filename string) bool {
	_, err := os.Stat(filename)
	return err == nil || os.IsExist(err)
}

func IsNotExist(filename string) bool {
	return !IsExist(filename)
}

func Struct2Map(obj interface{}) map[string]interface{} {
	t := reflect.TypeOf(obj)

	for t.Kind() == reflect.Ptr {
		t = t.Elem()
	}

	v := reflect.ValueOf(obj)
	v = reflect.Indirect(v)

	var data = make(map[string]interface{})

top:
	for i := 0; i < t.NumField(); i++ {
		name, ok := t.Field(i).Tag.Lookup("gorm")
		if ok {
		inner:
			for _, part := range strings.Split(name, ";") {
				kv := strings.Split(part, ":")
				if len(kv) != 2 {
					continue inner
				}
				if kv[0] == "column" {
					data[kv[1]] = v.Field(i).Interface()
					continue top
				}
			}
		}

		name, ok = t.Field(i).Tag.Lookup("json")
		if ok && name != "-" {
			name = strings.TrimSuffix(name, ",omitempty")
			data[name] = v.Field(i).Interface()
			continue
		}

		data[camelToSnake(t.Field(i).Name)] = v.Field(i).Interface()
	}
	return data
}

func camelToSnake(s string) string {
	out := make([]rune, 0, len(s)*2)
	for i, c := range s {
		if c >= 'A' && c <= 'Z' {
			if i == 0 {
				out = append(out, c-'A'+'a')
			} else {
				out = append(out, '_', c-'A'+'a')
			}
		} else {
			out = append(out, c)
		}
	}
	return string(out)
}

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
