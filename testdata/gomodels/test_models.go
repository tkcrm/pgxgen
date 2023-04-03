package gomodels

import (
	"database/sql"
	"database/sql/driver"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgtype"
	"github.com/shopspring/decimal"
)

type WebhookStatus string

const (
	WebhookStatusCreated    WebhookStatus = "Created"
	WebhookStatusProcessing WebhookStatus = "Processing"
	WebhookStatusComplete   WebhookStatus = "Complete"
	WebhookStatusRefund     WebhookStatus = "Refund"
	WebhookStatusError      WebhookStatus = "Error"
)

func (e *WebhookStatus) Scan(src interface{}) error {
	switch s := src.(type) {
	case []byte:
		*e = WebhookStatus(s)
	case string:
		*e = WebhookStatus(s)
	default:
		return fmt.Errorf("unsupported scan type for WebhookStatus: %T", src)
	}
	return nil
}

type NullWebhookStatus struct {
	WebhookStatus WebhookStatus
	Valid         bool // Valid is true if WebhookStatus is not NULL
}

// Scan implements the Scanner interface.
func (ns *NullWebhookStatus) Scan(value interface{}) error {
	if value == nil {
		ns.WebhookStatus, ns.Valid = "", false
		return nil
	}
	ns.Valid = true
	return ns.WebhookStatus.Scan(value)
}

// Value implements the driver Valuer interface.
func (ns NullWebhookStatus) Value() (driver.Value, error) {
	if !ns.Valid {
		return nil, nil
	}
	return string(ns.WebhookStatus), nil
}

func AllWebhookStatusValues() []WebhookStatus {
	return []WebhookStatus{
		WebhookStatusCreated,
		WebhookStatusProcessing,
		WebhookStatusComplete,
		WebhookStatusRefund,
		WebhookStatusError,
	}
}

type WebhookType string

const (
	WebhookTypeOne   WebhookType = "One"
	WebhookTypeTwo   WebhookType = "Two"
	WebhookTypeThree WebhookType = "Three"
)

func (e *WebhookType) Scan(src interface{}) error {
	switch s := src.(type) {
	case []byte:
		*e = WebhookType(s)
	case string:
		*e = WebhookType(s)
	default:
		return fmt.Errorf("unsupported scan type for WebhookType: %T", src)
	}
	return nil
}

type NullWebhookType struct {
	WebhookType WebhookType
	Valid       bool // Valid is true if WebhookType is not NULL
}

// Scan implements the Scanner interface.
func (ns *NullWebhookType) Scan(value interface{}) error {
	if value == nil {
		ns.WebhookType, ns.Valid = "", false
		return nil
	}
	ns.Valid = true
	return ns.WebhookType.Scan(value)
}

// Value implements the driver Valuer interface.
func (ns NullWebhookType) Value() (driver.Value, error) {
	if !ns.Valid {
		return nil, nil
	}
	return string(ns.WebhookType), nil
}

func AllWebhookTypeValues() []WebhookType {
	return []WebhookType{
		WebhookTypeOne,
		WebhookTypeTwo,
		WebhookTypeThree,
	}
}

type Webhook struct {
	ID            uuid.UUID       `db:"id"`
	Type          WebhookType     `db:"type"`
	Status        WebhookStatus   `db:"status"`
	Error         string          `db:"error"`
	Amount        decimal.Decimal `db:"amount"`
	Currency      string          `db:"currency"`
	RawData       pgtype.JSONB    `db:"raw_data"`
	TransactionID uuid.UUID       `db:"transaction_id"`
	Commission    decimal.Decimal `db:"commission"`
	CreatedAt     sql.NullTime    `db:"created_at"`
	UpdatedAt     sql.NullTime    `db:"updated_at"`
}
