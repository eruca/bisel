package utils

import (
	"os"
	"reflect"
	"strings"
)

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
