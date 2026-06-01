package storage

import (
	"time"

	"gorm.io/gorm"
)

type Notifications struct{ db *gorm.DB }

func NewNotifications(db *gorm.DB) *Notifications { return &Notifications{db: db} }

func (r *Notifications) ListChannels() ([]NotificationChannel, error) {
	var list []NotificationChannel
	if err := r.db.Order("id ASC").Find(&list).Error; err != nil {
		return nil, err
	}
	return list, nil
}

func (r *Notifications) ListEnabledChannels() ([]NotificationChannel, error) {
	var list []NotificationChannel
	if err := r.db.Where("enabled = ?", true).Find(&list).Error; err != nil {
		return nil, err
	}
	return list, nil
}

func (r *Notifications) FindChannel(id uint) (*NotificationChannel, error) {
	var c NotificationChannel
	if err := r.db.First(&c, id).Error; err != nil {
		return nil, err
	}
	return &c, nil
}

func (r *Notifications) CreateChannel(c *NotificationChannel) error { return r.db.Create(c).Error }
func (r *Notifications) UpdateChannel(c *NotificationChannel) error { return r.db.Save(c).Error }
func (r *Notifications) DeleteChannel(id uint) error                { return r.db.Delete(&NotificationChannel{}, id).Error }

func (r *Notifications) AppendLog(l *NotificationLog) error {
	if l.SentAt.IsZero() {
		l.SentAt = time.Now()
	}
	return r.db.Create(l).Error
}

func (r *Notifications) ListLogs(limit int) ([]NotificationLog, error) {
	if limit <= 0 {
		limit = 100
	}
	var list []NotificationLog
	if err := r.db.Order("sent_at DESC").Limit(limit).Find(&list).Error; err != nil {
		return nil, err
	}
	return list, nil
}

// DeleteLogsBefore 删除 sent_at < cutoff 的通知日志，返回删除行数。
func (r *Notifications) DeleteLogsBefore(cutoff time.Time) (int64, error) {
	res := r.db.Where("sent_at < ?", cutoff).Delete(&NotificationLog{})
	return res.RowsAffected, res.Error
}
