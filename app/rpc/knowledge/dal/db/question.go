package db

import (
	"context"
	"time"

	"gorm.io/gorm"
)

type Question struct {
	QuestionID      string   `json:"question_id"`
	QuestionContent string   `json:"question_content"`
	StandardAnswer  string   `json:"standard_answer"`
	Tags            []string `json:"tags"`
	Difficulty      string   `json:"difficulty"`

	CSVID string `json:"csv_id"`

	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`
}

func (m *DBManager) CreateQuestion(ctx context.Context, questions []*Question) error {
	return m.db.WithContext(ctx).Create(questions).Error
}

type QuestionCSV struct {
	ID       string `gorm:"primaryKey;type:varchar(64)" json:"questioncsv_id"`
	UserID   string `gorm:"type:varchar(64);index" json:"user_id"`
	FileName string `gorm:"type:varchar(255)" json:"file_name"`
	FileURL  string `gorm:"type:varchar(512)" json:"file_url"`

	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`
}

func (m *DBManager) CreateQuestionCSV(ctx context.Context, questions []*QuestionCSV) error {
	return m.db.WithContext(ctx).Create(questions).Error
}
