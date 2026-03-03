package models

import "time"

type Department struct {
	ID        int64        `gorm:"primaryKey" json:"id"`
	Name      string       `json:"name"`
	ParentID  *int64       `json:"parent_id"`
	CreatedAt time.Time    `json:"created_at"`
	Children  []Department `gorm:"foreignKey:ParentID" json:"-"`
	Employees []Employee   `json:"-"`
}

type Employee struct {
	ID           int64      `gorm:"primaryKey" json:"id"`
	DepartmentID int64      `json:"department_id"`
	FullName     string     `json:"full_name"`
	Position     string     `json:"position"`
	HiredAt      *time.Time `gorm:"type:date" json:"hired_at"`
	CreatedAt    time.Time  `json:"created_at"`
}
