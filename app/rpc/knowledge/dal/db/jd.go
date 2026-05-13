package db

import (
	"context"
	"time"

	"gorm.io/gorm"
)

type JD struct {
	JDID             string   `gorm:"column:jd_id;primaryKey"`
	Company          string   `gorm:"column:company"`
	Title            string   `gorm:"column:title"`
	Responsibilities string   `gorm:"column:responsibilities"`
	Requirements     string   `gorm:"column:requirements"`
	Tags             []string `gorm:"column:tags;serializer:json"`

	CSVID string `json:"csv_id"`

	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`
}

func (m *DBManager) CreateJD(ctx context.Context, jds []*JD) error {
	return m.db.WithContext(ctx).Create(jds).Error
}

type JDCSV struct {
	ID       string `gorm:"primaryKey;type:varchar(64)" json:"jdcsv_id"`
	UserID   string `gorm:"type:varchar(64);index" json:"user_id"`
	FileName string `gorm:"type:varchar(255)" json:"file_name"`
	FileURL  string `gorm:"type:varchar(512)" json:"file_url"`

	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`
}

func (m *DBManager) CreateJDCSV(ctx context.Context, jds []*JDCSV) error {
	return m.db.WithContext(ctx).Create(jds).Error
}
