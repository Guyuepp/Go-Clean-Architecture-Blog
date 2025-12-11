package repository

import (
	"encoding/base64"
	"time"
)

const (
	timeFormat  = "2006-01-02T15:04:05.999Z07:00" // reduce precision from RFC3339Nano as date format
	MaxPageSize = 100
	MinPageSize = 10
)

// DecodeCursor will decode cursor from user for mysql
func DecodeCursor(encodedTime string) (time.Time, error) {
	byt, err := base64.StdEncoding.DecodeString(encodedTime)
	if err != nil {
		return time.Time{}, err
	}

	timeString := string(byt)
	t, err := time.Parse(timeFormat, timeString)

	return t, err
}

// EncodeCursor will encode cursor from mysql to user
func EncodeCursor(t time.Time) string {
	timeString := t.Format(timeFormat)

	return base64.StdEncoding.EncodeToString([]byte(timeString))
}

// PageVerify 分页查询 过滤器
func PageVerify(pageSize *int64) {
	switch {
	case *pageSize > 100:
		*pageSize = MaxPageSize
	case *pageSize <= 0:
		*pageSize = MinPageSize
	}
}
