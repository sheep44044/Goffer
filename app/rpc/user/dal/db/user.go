package db

import (
	"context"
	"time"

	"gorm.io/gorm"
)

type DBManager struct {
	db *gorm.DB
}

// NewDBManager 构造函数
func NewDBManager(db *gorm.DB) *DBManager {
	return &DBManager{
		db: db,
	}
}

type User struct {
	ID       string `gorm:"primaryKey;type:varchar(64)" json:"user_id"`
	Username string `gorm:"type:varchar(255);uniqueIndex" json:"user_name"`
	Password string `gorm:"type:varchar(255)" json:"password"`

	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`
}

func (m *DBManager) CreateUser(ctx context.Context, users []*User) error {
	return m.db.WithContext(ctx).Create(users).Error
}

func (m *DBManager) QueryUser(ctx context.Context, userName string) ([]*User, error) {
	res := make([]*User, 0)
	if err := m.db.WithContext(ctx).Where("user_name = ?", userName).Find(&res).Error; err != nil {
		return nil, err
	}
	return res, nil
}

type Resume struct {
	ID          string `gorm:"primaryKey;type:varchar(64)" json:"resume_id"`
	UserID      string `gorm:"type:varchar(64);index" json:"user_id"` // 关联到具体的打工人
	FileName    string `gorm:"type:varchar(255)" json:"file_name"`    // 用户视角的原文件名
	FileURL     string `gorm:"type:varchar(512)" json:"file_url"`     // MinIO 的实际访问路径
	ParseStatus int    `gorm:"type:tinyint;default:0" json:"status"`  // 0:未解析, 1:向量化中, 2:解析完成

	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`
}

func (m *DBManager) CreateResume(ctx context.Context, resumes []*Resume) error {
	return m.db.WithContext(ctx).Create(resumes).Error
}

func (m *DBManager) UpdateResumeStatus(ctx context.Context, resumeID string, status int) error {
	return m.db.WithContext(ctx).Model(&Resume{}).Where("id = ?", resumeID).Update("status", status).Error
}

func (m *DBManager) GetResumeStatus(ctx context.Context, resumeID string, userID string) (int, error) {
	var resume Resume
	err := m.db.WithContext(ctx).Where("id = ? AND user_id = ?", resumeID, userID).First(resume).Error
	if err != nil {
		return 0, err // 记录不存在或其他错误
	}
	return resume.ParseStatus, nil
}
