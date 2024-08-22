package repo

import (
	"database/sql"
	"errors"
	"fmt"
	"github.com/jmoiron/sqlx"
	"github.com/lib/pq"
	"reflect"
	"strings"
)

const (
	update = "update"
	insert = "insert"
)

func AddRow(tx *sqlx.Tx, object any, tableName string) error {
	query := generateInsertQuery(object, tableName)
	valuesMap := getValuesMap(object)
	_, err := tx.NamedExec(query, valuesMap)
	return err
}

func UpdateRow(tx *sqlx.Tx, object any, tableName string, keyField string) error {
	query := generateUpdateQuery(object, tableName, keyField)
	valuesMap := getValuesMap(object)
	_, err := tx.NamedExec(query, valuesMap)
	return err
}

func Exists(tx *sqlx.Tx, tableName string, fieldName string, val interface{}) (bool, error) {
	query := fmt.Sprintf(`SELECT 1 FROM %s WHERE %s = $1 LIMIT 1;`, tableName, fieldName)

	var exists int
	err := tx.QueryRowx(query, val).Scan(&exists)
	if err != nil && errors.Is(err, sql.ErrNoRows) {
		return false, nil
	}

	return true, err
}

func UpsertRow(tx *sqlx.Tx, object any, tableName string, keyField string) (opType string, err error) {
	keyValue, err := getFieldValueByTag(object, keyField)
	if err != nil {
		return
	}

	exists, err := Exists(tx, tableName, keyField, keyValue)
	if err != nil {
		return
	}

	if exists {
		opType = update
		err = UpdateRow(tx, object, tableName, keyField)
		return
	}

	opType = insert
	err = AddRow(tx, object, tableName)
	return
}

func generateInsertQuery(obj interface{}, tableName string) string {
	t := reflect.TypeOf(obj)
	if t.Kind() == reflect.Ptr {
		t = t.Elem()
	}

	var columns []string
	var placeholders []string

	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		dbTag := field.Tag.Get("db")
		if dbTag != "" {
			columns = append(columns, dbTag)
			placeholders = append(placeholders, ":"+dbTag)
		}
	}

	columnsStr := strings.Join(columns, ", ")
	placeholdersStr := strings.Join(placeholders, ", ")

	query := fmt.Sprintf(`INSERT INTO %s (%s) VALUES (%s);`, tableName, columnsStr, placeholdersStr)
	return query
}

func generateUpdateQuery(obj interface{}, tableName string, keyField string) string {
	t := reflect.TypeOf(obj)
	if t.Kind() == reflect.Ptr {
		t = t.Elem()
	}

	var updateFields []string

	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		dbTag := field.Tag.Get("db")
		if dbTag != "" && dbTag != keyField {
			updateFields = append(updateFields, fmt.Sprintf("%s = :%s", dbTag, dbTag))
		}
	}

	updateFieldsStr := strings.Join(updateFields, ", ")
	query := fmt.Sprintf(`UPDATE %s SET %s WHERE %s = :%s;`, tableName, updateFieldsStr, keyField, keyField)
	return query
}

func getFieldValueByTag(obj interface{}, tagName string) (interface{}, error) {
	v := reflect.ValueOf(obj)

	if v.Kind() == reflect.Ptr {
		v = v.Elem()
	}
	if v.Kind() != reflect.Struct {
		return nil, fmt.Errorf("expected a struct or a pointer to a struct, got %T", obj)
	}

	t := v.Type()
	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		if dbTag := field.Tag.Get("db"); dbTag == tagName {
			return v.Field(i).Interface(), nil
		}
	}

	return nil, fmt.Errorf("no field with db tag %q found", tagName)
}

func getValuesMap(obj interface{}) map[string]interface{} {
	m := make(map[string]interface{})
	t := reflect.TypeOf(obj)

	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		tagName := field.Tag.Get("db")
		if tagName == "" {
			continue
		}
		val, _ := getFieldValueByTag(obj, tagName)

		// cast slices to db array
		if field.Type.Kind() == reflect.Slice {
			val = pq.Array(val)
		}

		m[tagName] = val
	}

	return m
}
