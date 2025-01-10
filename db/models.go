// Code generated by sqlc. DO NOT EDIT.
// versions:
//   sqlc v1.27.0

package db

import (
	"database/sql"
	"database/sql/driver"
	"fmt"
	"time"
)

type ThingsType string

const (
	ThingsTypeFile      ThingsType = "file"
	ThingsTypeDir       ThingsType = "dir"
	ThingsTypeIrods     ThingsType = "irods"
	ThingsTypeS3        ThingsType = "s3"
	ThingsTypeOpenstack ThingsType = "openstack"
)

func (e *ThingsType) Scan(src interface{}) error {
	switch s := src.(type) {
	case []byte:
		*e = ThingsType(s)
	case string:
		*e = ThingsType(s)
	default:
		return fmt.Errorf("unsupported scan type for ThingsType: %T", src)
	}
	return nil
}

type NullThingsType struct {
	ThingsType ThingsType
	Valid      bool // Valid is true if ThingsType is not NULL
}

// Scan implements the Scanner interface.
func (ns *NullThingsType) Scan(value interface{}) error {
	if value == nil {
		ns.ThingsType, ns.Valid = "", false
		return nil
	}
	ns.Valid = true
	return ns.ThingsType.Scan(value)
}

// Value implements the driver Valuer interface.
func (ns NullThingsType) Value() (driver.Value, error) {
	if !ns.Valid {
		return nil, nil
	}
	return string(ns.ThingsType), nil
}

type Subscriber struct {
	UserID  uint32
	ThingID uint32
}

type Thing struct {
	ID          uint32
	Address     string
	Type        ThingsType
	Created     time.Time
	Description sql.NullString
	Reason      string
	Remove      time.Time
	Warned1     sql.NullTime
	Warned2     sql.NullTime
	Removed     sql.NullBool
}

type User struct {
	ID    uint32
	Name  string
	Email string
}
