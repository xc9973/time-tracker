package utils

import (
	"fmt"
	"reflect"
	"strings"
)

// BuildUpdateQuery constructs a SQL UPDATE query and arguments from a struct with pointers.
// It iterates over the fields of the data struct, and for each non-nil pointer field,
// adds it to the update list.
// The data struct fields should have `db:"column_name"` tags or match column names lowercased.
// However, for simplicity here, we assume the data struct fields map directly to columns or we pass a map.
// Given the existing code uses structs like models.LogUpdate where fields map to columns:
// Category -> category, Task -> task, etc.
// We can use a simple mapping or just pass the map of column to value pointer.

// BuildWhereClause builds the WHERE clause and arguments for a query.
// conditions is a slice of strings (e.g., "category = ?", "status = ?").
// args is a slice of arguments corresponding to the conditions.
// It returns the combined WHERE clause (starting with " WHERE ") and the arguments.
func BuildWhereClause(conditions []string) string {
	if len(conditions) == 0 {
		return ""
	}
	return " WHERE " + strings.Join(conditions, " AND ")
}

// BuildUpdateQueryFromStruct builds update parts for a given struct.
// It assumes the struct fields are pointers.
// It takes a map of "FieldName" -> "column_name".
func BuildUpdateQueryFromStruct(data interface{}, fieldToCol map[string]string) ([]string, []interface{}) {
	val := reflect.ValueOf(data)
	if val.Kind() == reflect.Ptr {
		val = val.Elem()
	}

	var updates []string
	var args []interface{}

	for i := 0; i < val.NumField(); i++ {
		field := val.Type().Field(i)
		fieldName := field.Name

		colName, ok := fieldToCol[fieldName]
		if !ok {
			continue
		}

		value := val.Field(i)
		if !value.IsNil() {
			// value is a pointer, we need the value it points to
			updates = append(updates, fmt.Sprintf("%s = ?", colName))
			args = append(args, value.Elem().Interface())
		}
	}

	return updates, args
}
