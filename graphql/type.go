package graphql

import (
	"database/sql"
	"gopkg.in/guregu/null.v3"
	"strings"
	"time"
)

type YyyyMmDdHhMmSs struct {
	sql.NullTime
}

func (date *YyyyMmDdHhMmSs) UnmarshalJSON(data []byte) error {
	str := strings.TrimRight(strings.TrimLeft(string(data), "\""), "\"")
	if str == "null" || str == "" {
		return nil
	}

	layout := "2006-01-02 15:04:05"
	if strings.Contains(str, "T") {
		layout = "2006-01-02T15:04:05"
	}

	t, err := time.Parse(layout, str)
	if err != nil {
		return err
	}

	date.Time = t
	return nil
}

type ArrayToString struct {
	null.String
}

func (arr *ArrayToString) UnmarshalJSON(data []byte) error {
	str := string(data)
	if str == "null" || str == "" {
		arr.Valid = false
		return nil
	}

	arr.String = null.String{
		NullString: sql.NullString{
			String: strings.TrimRight(strings.TrimLeft(str, "\""), "\""),
		},
	}
	arr.Valid = true
	return nil
}
