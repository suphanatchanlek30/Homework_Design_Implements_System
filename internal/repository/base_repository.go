package repository

import (
	"context"

	"gorm.io/gorm"
)

type BaseRepository[T any] struct {
	db *gorm.DB
}

// NewBaseRepository creates a reusable generic repository around one GORM database handle.
// สร้าง generic repository พื้นฐานที่ใช้ซ้ำได้บน GORM database handle ตัวเดียว
func NewBaseRepository[T any](db *gorm.DB) *BaseRepository[T] {
	return &BaseRepository[T]{db: db}
}

// Create inserts one entity row using the caller's context.
// บันทึก entity หนึ่งรายการลงฐานข้อมูลโดยใช้ context ของผู้เรียก
func (r *BaseRepository[T]) Create(ctx context.Context, entity *T) error {
	return r.db.WithContext(ctx).Create(entity).Error
}

// FindByID loads one entity by primary key.
// โหลด entity หนึ่งรายการจาก primary key
func (r *BaseRepository[T]) FindByID(ctx context.Context, id uint64) (*T, error) {
	var entity T
	if err := r.db.WithContext(ctx).First(&entity, id).Error; err != nil {
		return nil, err
	}
	return &entity, nil
}

// Update saves the current state of one entity back to the database.
// บันทึกสถานะล่าสุดของ entity กลับลงฐานข้อมูล
func (r *BaseRepository[T]) Update(ctx context.Context, entity *T) error {
	return r.db.WithContext(ctx).Save(entity).Error
}

// Delete removes one entity row by primary key.
// ลบ entity หนึ่งรายการตาม primary key
func (r *BaseRepository[T]) Delete(ctx context.Context, id uint64) error {
	var entity T
	return r.db.WithContext(ctx).Delete(&entity, id).Error
}

// List loads every row for the repository entity type.
// โหลดข้อมูลทุกแถวของ entity ชนิดนี้จากฐานข้อมูล
func (r *BaseRepository[T]) List(ctx context.Context) ([]T, error) {
	var entities []T
	if err := r.db.WithContext(ctx).Find(&entities).Error; err != nil {
		return nil, err
	}
	return entities, nil
}
