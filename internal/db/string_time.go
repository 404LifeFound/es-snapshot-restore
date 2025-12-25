package db

import (
	"database/sql/driver"
	"fmt"
	"time"
)

type TimeString struct {
	time.Time
}

func (t *TimeString) Scan(value any) error {
	if value == nil {
		return nil
	}

	switch v := value.(type) {
	case time.Time:
		t.Time = v
		return nil
	case string:
		return t.parse(v)
	case []byte:
		return t.parse(string(v))
	default:
		return fmt.Errorf("cannot scan %T into TimeString", value)
	}
}

func (t TimeString) Value() (driver.Value, error) {
	if t.Time.IsZero() {
		return nil, nil
	}
	return t.Time, nil
}

func (t *TimeString) parse(s string) error {
	layouts := []string{
		time.RFC3339,
		time.RFC3339Nano,
		"2006-01-02 15:04:05",
		"2006-01-02",
	}

	for _, layout := range layouts {
		if tt, err := time.ParseInLocation(layout, s, time.Local); err == nil {
			t.Time = tt
			return nil
		}
	}

	return fmt.Errorf("invalid time format: %s", s)
}

func NewTimeString(s string) (TimeString, error) {
	layouts := []string{
		time.RFC3339,
		time.RFC3339Nano,
		"2006-01-02 15:04:05",
		"2006-01-02",
	}

	for _, layout := range layouts {
		if t, err := time.ParseInLocation(layout, s, time.Local); err != nil {
			return TimeString{}, err
		} else {
			return TimeString{Time: t}, nil
		}
	}
	return TimeString{}, fmt.Errorf("not support time format: %s", s)
}
