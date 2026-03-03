package repository

import (
	"context"
	"errors"

	"org-api/internal/models"

	"gorm.io/gorm"
)

type GormRepository struct {
	db *gorm.DB
}

func NewGormRepository(db *gorm.DB) *GormRepository {
	return &GormRepository{db: db}
}

func (r *GormRepository) CreateDepartment(ctx context.Context, dept *models.Department) error {
	return r.db.WithContext(ctx).Create(dept).Error
}

func (r *GormRepository) GetDepartmentByID(ctx context.Context, id int64) (*models.Department, error) {
	var dept models.Department
	if err := r.db.WithContext(ctx).First(&dept, id).Error; err != nil {
		return nil, err
	}

	return &dept, nil
}

func (r *GormRepository) UpdateDepartment(ctx context.Context, dept *models.Department) error {
	return r.db.WithContext(ctx).Save(dept).Error
}

func (r *GormRepository) DeleteDepartment(ctx context.Context, id int64, options DeleteOptions) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var dept models.Department
		if err := tx.First(&dept, id).Error; err != nil {
			return err
		}

		switch options.Mode {
		case DeleteModeCascade:
			return tx.Delete(&dept).Error
		case DeleteModeReassign:
			if options.ReassignToDepartment == nil {
				return ErrMissingReassignTarget
			}

			if *options.ReassignToDepartment == id {
				return ErrReassignToSameDepartment
			}

			var exists int64
			if err := tx.Model(&models.Department{}).
				Where("id = ?", *options.ReassignToDepartment).
				Count(&exists).Error; err != nil {
				return err
			}
			if exists == 0 {
				return gorm.ErrRecordNotFound
			}

			if err := tx.Model(&models.Department{}).
				Where("parent_id = ?", id).
				Update("parent_id", dept.ParentID).Error; err != nil {
				return err
			}

			if err := tx.Model(&models.Employee{}).
				Where("department_id = ?", id).
				Update("department_id", *options.ReassignToDepartment).Error; err != nil {
				return err
			}

			return tx.Delete(&dept).Error
		default:
			return ErrInvalidDeleteMode
		}
	})
}

func (r *GormRepository) ListChildDepartments(ctx context.Context, parentID int64) ([]models.Department, error) {
	var departments []models.Department
	if err := r.db.WithContext(ctx).
		Where("parent_id = ?", parentID).
		Order("name ASC").
		Find(&departments).Error; err != nil {
		return nil, err
	}

	return departments, nil
}

func (r *GormRepository) DepartmentExists(ctx context.Context, id int64) (bool, error) {
	var count int64
	if err := r.db.WithContext(ctx).Model(&models.Department{}).
		Where("id = ?", id).
		Count(&count).Error; err != nil {
		return false, err
	}

	return count > 0, nil
}

func (r *GormRepository) DepartmentNameExistsUnderParent(
	ctx context.Context,
	parentID *int64,
	name string,
	excludeID *int64,
) (bool, error) {
	query := r.db.WithContext(ctx).Model(&models.Department{}).Where("lower(name) = lower(?)", name)
	if parentID == nil {
		query = query.Where("parent_id IS NULL")
	} else {
		query = query.Where("parent_id = ?", *parentID)
	}

	if excludeID != nil {
		query = query.Where("id <> ?", *excludeID)
	}

	var count int64
	if err := query.Count(&count).Error; err != nil {
		return false, err
	}

	return count > 0, nil
}

func (r *GormRepository) GetParentID(ctx context.Context, id int64) (*int64, error) {
	var dept models.Department
	if err := r.db.WithContext(ctx).Select("id", "parent_id").First(&dept, id).Error; err != nil {
		return nil, err
	}

	return dept.ParentID, nil
}

func (r *GormRepository) CreateEmployee(ctx context.Context, employee *models.Employee) error {
	return r.db.WithContext(ctx).Create(employee).Error
}

func (r *GormRepository) ListEmployeesByDepartment(ctx context.Context, departmentID int64) ([]models.Employee, error) {
	var employees []models.Employee
	if err := r.db.WithContext(ctx).
		Where("department_id = ?", departmentID).
		Order("full_name ASC").
		Find(&employees).Error; err != nil {
		return nil, err
	}

	return employees, nil
}

func IsNotFound(err error) bool {
	return errors.Is(err, gorm.ErrRecordNotFound)
}
