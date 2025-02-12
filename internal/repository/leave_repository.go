// internal/repository/leave_repository.go
package repository

import (
	"fmt"
	"time"

	"github.com/Axontik/comin-leave-management-service/internal/domain"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type LeaveRepository interface {
	// LeaveType methods
	CreateLeaveType(leaveType *domain.LeaveType) error
	GetLeaveType(id uuid.UUID) (*domain.LeaveType, error)
	UpdateLeaveType(leaveType *domain.LeaveType) error
	DeleteLeaveType(id uuid.UUID) error
	ListLeaveTypes(orgID uuid.UUID) ([]domain.LeaveType, error)

	// LeaveRequest methods
	CreateLeaveRequest(request *domain.LeaveRequest) error
	GetLeaveRequest(id uuid.UUID) (*domain.LeaveRequest, error)
	UpdateLeaveRequest(request *domain.LeaveRequest) error
	ListLeaveRequests(orgID, employeeID uuid.UUID, status string) ([]domain.LeaveRequest, error)
	GetOverlappingRequests(employeeID uuid.UUID, startDate, endDate time.Time) ([]domain.LeaveRequest, error)

	// LeaveBalance methods
	GetLeaveBalance(employeeID, leaveTypeID uuid.UUID, year int) (*domain.LeaveBalance, error)
	UpdateLeaveBalance(balance *domain.LeaveBalance) error
	ListLeaveBalances(employeeID uuid.UUID) ([]domain.LeaveBalance, error)

	// Balance Adjustment methods
	CreateBalanceAdjustment(adjustment *domain.LeaveBalanceAdjustment) error
	GetBalanceAdjustment(id uuid.UUID) (*domain.LeaveBalanceAdjustment, error)
	UpdateBalanceAdjustment(adjustment *domain.LeaveBalanceAdjustment) error
	ListBalanceAdjustments(balanceID uuid.UUID) ([]domain.LeaveBalanceAdjustment, error)

	HasActiveLeaveRequests(leaveTypeID uuid.UUID) (bool, error)
	ListLeaveTypesWithOptions(orgID uuid.UUID, params *domain.ListLeaveTypesParams) ([]domain.LeaveType, int64, error)
}

type leaveRepository struct {
	db *gorm.DB
}

func NewLeaveRepository(db *gorm.DB) LeaveRepository {
	return &leaveRepository{db: db}
}

// LeaveType implementation
func (r *leaveRepository) CreateLeaveType(leaveType *domain.LeaveType) error {
	if leaveType.ID == uuid.Nil {
		leaveType.ID = uuid.New()
	}
	return r.db.Create(leaveType).Error
}

func (r *leaveRepository) GetLeaveType(id uuid.UUID) (*domain.LeaveType, error) {
	var leaveType domain.LeaveType
	err := r.db.First(&leaveType, "id = ?", id).Error
	return &leaveType, err
}

func (r *leaveRepository) UpdateLeaveType(leaveType *domain.LeaveType) error {
	return r.db.Model(&domain.LeaveType{}).
		Where("id = ? AND organization_id = ?", leaveType.ID, leaveType.OrganizationID).
		Updates(leaveType).Error
}

func (r *leaveRepository) DeleteLeaveType(id uuid.UUID) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		// Check if there are any active leave requests
		var count int64
		if err := tx.Model(&domain.LeaveRequest{}).
			Where("leave_type_id = ? AND status IN ?", id, []string{"pending", "approved"}).
			Count(&count).Error; err != nil {
			return err
		}

		if count > 0 {
			return fmt.Errorf("cannot delete leave type with active requests")
		}

		return tx.Delete(&domain.LeaveType{}, "id = ?", id).Error
	})
}

func (r *leaveRepository) ListLeaveTypes(orgID uuid.UUID) ([]domain.LeaveType, error) {
	var leaveTypes []domain.LeaveType

	// Query with organization filter and active status
	query := r.db.Where("organization_id = ?", orgID)

	// Execute query with ordering
	err := query.
		Order("name ASC").
		Find(&leaveTypes).
		Error

	if err != nil {
		return nil, fmt.Errorf("failed to list leave types: %w", err)
	}

	return leaveTypes, nil
}

// With pagination and filtering options
func (r *leaveRepository) ListLeaveTypesWithOptions(orgID uuid.UUID, params *domain.ListLeaveTypesParams) ([]domain.LeaveType, int64, error) {
	var leaveTypes []domain.LeaveType
	var total int64

	// Base query
	query := r.db.Model(&domain.LeaveType{}).
		Where("organization_id = ?", orgID)

	// Apply filters if provided
	if params != nil {
		if params.IsPaid != nil {
			query = query.Where("is_paid = ?", *params.IsPaid)
		}
		if params.RequiresApproval != nil {
			query = query.Where("requires_approval = ?", *params.RequiresApproval)
		}
		if params.Name != "" {
			query = query.Where("name ILIKE ?", "%"+params.Name+"%")
		}
	}

	// Get total count before pagination
	err := query.Count(&total).Error
	if err != nil {
		return nil, 0, fmt.Errorf("failed to count leave types: %w", err)
	}

	// Apply pagination
	if params != nil && params.Page > 0 && params.PageSize > 0 {
		offset := (params.Page - 1) * params.PageSize
		query = query.Offset(offset).Limit(params.PageSize)
	}

	// Execute final query with ordering
	err = query.
		Order("name ASC").
		Find(&leaveTypes).
		Error

	if err != nil {
		return nil, 0, fmt.Errorf("failed to list leave types: %w", err)
	}

	return leaveTypes, total, nil
}

func (r *leaveRepository) CreateLeaveRequest(request *domain.LeaveRequest) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		// Create the leave request
		if err := tx.Create(request).Error; err != nil {
			return err
		}

		// Update leave balance
		result := tx.Model(&domain.LeaveBalance{}).
			Where("employee_id = ? AND leave_type_id = ? AND year = ?",
				request.EmployeeID, request.LeaveTypeID, request.StartDate.Year()).
			Update("pending_days", gorm.Expr("pending_days + ?", request.Days))

		if result.Error != nil {
			return result.Error
		}

		if result.RowsAffected == 0 {
			return fmt.Errorf("no leave balance found for employee")
		}

		return nil
	})
}

func (r *leaveRepository) GetLeaveRequest(id uuid.UUID) (*domain.LeaveRequest, error) {
	var request domain.LeaveRequest
	err := r.db.Preload("LeaveType").First(&request, "id = ?", id).Error
	return &request, err
}

func (r *leaveRepository) UpdateLeaveRequest(request *domain.LeaveRequest) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		oldRequest := &domain.LeaveRequest{}
		if err := tx.First(oldRequest, request.ID).Error; err != nil {
			return err
		}

		// Update leave balances based on status change
		if oldRequest.Status != request.Status {
			balance := &domain.LeaveBalance{}
			err := tx.Where("employee_id = ? AND leave_type_id = ? AND year = ?",
				request.EmployeeID, request.LeaveTypeID, request.StartDate.Year()).
				First(balance).Error
			if err != nil {
				return err
			}

			switch request.Status {
			case "approved":
				balance.PendingDays -= request.Days
				balance.UsedDays += request.Days
			case "rejected", "cancelled":
				balance.PendingDays -= request.Days
			}

			if err := tx.Save(balance).Error; err != nil {
				return err
			}
		}

		return tx.Save(request).Error
	})
}

func (r *leaveRepository) ListLeaveRequests(orgID, employeeID uuid.UUID, status string) ([]domain.LeaveRequest, error) {
	var requests []domain.LeaveRequest
	query := r.db.Preload("LeaveType").Where("organization_id = ?", orgID)

	if employeeID != uuid.Nil {
		query = query.Where("employee_id = ?", employeeID)
	}
	if status != "" {
		query = query.Where("status = ?", status)
	}

	err := query.Order("created_at DESC").Find(&requests).Error
	return requests, err
}

func (r *leaveRepository) GetOverlappingRequests(employeeID uuid.UUID, startDate, endDate time.Time) ([]domain.LeaveRequest, error) {
	var requests []domain.LeaveRequest
	err := r.db.Where("employee_id = ? AND status IN (?) AND "+
		"((start_date BETWEEN ? AND ?) OR (end_date BETWEEN ? AND ?) OR (start_date <= ? AND end_date >= ?))",
		employeeID, []string{"pending", "approved"},
		startDate, endDate, startDate, endDate, startDate, endDate).
		Find(&requests).Error
	return requests, err
}

// LeaveBalance methods
func (r *leaveRepository) GetLeaveBalance(employeeID, leaveTypeID uuid.UUID, year int) (*domain.LeaveBalance, error) {
	var balance domain.LeaveBalance
	err := r.db.Preload("LeaveType").
		Where("employee_id = ? AND leave_type_id = ? AND year = ?",
			employeeID, leaveTypeID, year).
		First(&balance).Error
	return &balance, err
}

func (r *leaveRepository) UpdateLeaveBalance(balance *domain.LeaveBalance) error {
	result := r.db.Model(&domain.LeaveBalance{}).
		Where("id = ?", balance.ID).
		Updates(map[string]interface{}{
			"total_days": balance.TotalDays,
			"updated_at": time.Now(),
		})

	if result.Error != nil {
		return fmt.Errorf("failed to update leave balance: %w", result.Error)
	}

	if result.RowsAffected == 0 {
		return fmt.Errorf("leave balance not found")
	}

	return nil
}

func (r *leaveRepository) ListLeaveBalances(employeeID uuid.UUID) ([]domain.LeaveBalance, error) {
	var balances []domain.LeaveBalance
	err := r.db.Preload("LeaveType").
		Where("employee_id = ? AND year = ?", employeeID, time.Now().Year()).
		Find(&balances).Error
	return balances, err
}

// Holiday methods
func (r *leaveRepository) CreateHoliday(holiday *domain.Holiday) error {
	return r.db.Create(holiday).Error
}

func (r *leaveRepository) GetHoliday(id uuid.UUID) (*domain.Holiday, error) {
	var holiday domain.Holiday
	err := r.db.First(&holiday, "id = ?", id).Error
	return &holiday, err
}

func (r *leaveRepository) UpdateHoliday(holiday *domain.Holiday) error {
	return r.db.Save(holiday).Error
}

func (r *leaveRepository) DeleteHoliday(id uuid.UUID) error {
	return r.db.Delete(&domain.Holiday{}, "id = ?", id).Error
}

func (r *leaveRepository) ListHolidays(orgID uuid.UUID, startDate, endDate time.Time) ([]domain.Holiday, error) {
	var holidays []domain.Holiday
	query := r.db.Where("organization_id = ?", orgID)

	if !startDate.IsZero() && !endDate.IsZero() {
		query = query.Where("date BETWEEN ? AND ?", startDate, endDate)
	}

	err := query.Order("date ASC").Find(&holidays).Error
	return holidays, err
}

// Leave Request History methods
func (r *leaveRepository) CreateLeaveRequestHistory(history *domain.LeaveRequestHistory) error {
	return r.db.Create(history).Error
}

func (r *leaveRepository) ListLeaveRequestHistory(leaveRequestID uuid.UUID) ([]domain.LeaveRequestHistory, error) {
	var history []domain.LeaveRequestHistory
	err := r.db.Where("leave_request_id = ?", leaveRequestID).
		Order("created_at DESC").
		Find(&history).Error
	return history, err
}

// Leave Balance operations
func (r *leaveRepository) InitializeYearlyBalance(orgID uuid.UUID, year int) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		// Get all active employees and leave types
		var leaveTypes []domain.LeaveType
		if err := tx.Where("organization_id = ?", orgID).Find(&leaveTypes).Error; err != nil {
			return err
		}

		// Query to get all active employees from the employee service
		// This would typically be done through a service client
		var employeeIDs []uuid.UUID
		// TODO: Get employee IDs from employee service

		// Create balances for each employee and leave type
		for _, empID := range employeeIDs {
			for _, leaveType := range leaveTypes {
				balance := &domain.LeaveBalance{
					OrganizationID: orgID,
					EmployeeID:     empID,
					LeaveTypeID:    leaveType.ID,
					Year:           year,
					TotalDays:      float64(leaveType.DefaultDays),
					UsedDays:       0,
					PendingDays:    0,
				}
				if err := tx.Create(balance).Error; err != nil {
					return err
				}
			}
		}
		return nil
	})
}

func (r *leaveRepository) AdjustLeaveBalance(balance *domain.LeaveBalance, adjustment float64, reason string) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		balance.TotalDays += adjustment

		// Create balance adjustment history
		history := &domain.LeaveBalanceAdjustment{
			LeaveBalanceID: balance.ID,
			Adjustment:     adjustment,
			Reason:         reason,
			PerformedBy:    uuid.UUID{}, // TODO: Get from context
		}

		if err := tx.Create(history).Error; err != nil {
			return err
		}

		return tx.Save(balance).Error
	})
}

// Reporting methods
func (r *leaveRepository) GetLeaveStats(orgID uuid.UUID, startDate, endDate time.Time) (*domain.LeaveStats, error) {
	var stats domain.LeaveStats

	// Total leave requests
	err := r.db.Model(&domain.LeaveRequest{}).
		Where("organization_id = ? AND start_date BETWEEN ? AND ?",
			orgID, startDate, endDate).
		Select("COUNT(*) as total_requests, " +
			"SUM(CASE WHEN status = 'approved' THEN days ELSE 0 END) as total_days_taken").
		Scan(&stats).Error

	if err != nil {
		return nil, err
	}

	// Leave by type
	err = r.db.Model(&domain.LeaveRequest{}).
		Joins("JOIN leave_types ON leave_requests.leave_type_id = leave_types.id").
		Where("leave_requests.organization_id = ? AND leave_requests.start_date BETWEEN ? AND ?",
			orgID, startDate, endDate).
		Group("leave_types.name").
		Select("leave_types.name, COUNT(*) as count, SUM(days) as total_days").
		Scan(&stats.LeaveByType).Error

	return &stats, err
}

func (r *leaveRepository) CreateBalanceAdjustment(adjustment *domain.LeaveBalanceAdjustment) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(adjustment).Error; err != nil {
			return err
		}

		if adjustment.Status == domain.AdjustmentStatusApproved {
			balance := &domain.LeaveBalance{}
			if err := tx.First(balance, adjustment.LeaveBalanceID).Error; err != nil {
				return err
			}

			balance.TotalDays += adjustment.Adjustment
			return tx.Save(balance).Error
		}

		return nil
	})
}

func (r *leaveRepository) GetBalanceAdjustment(id uuid.UUID) (*domain.LeaveBalanceAdjustment, error) {
	var adjustment domain.LeaveBalanceAdjustment
	err := r.db.Preload("LeaveBalance").First(&adjustment, "id = ?", id).Error
	return &adjustment, err
}

func (r *leaveRepository) UpdateBalanceAdjustment(adjustment *domain.LeaveBalanceAdjustment) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		oldAdjustment := &domain.LeaveBalanceAdjustment{}
		if err := tx.First(oldAdjustment, adjustment.ID).Error; err != nil {
			return err
		}

		if oldAdjustment.Status != adjustment.Status && adjustment.Status == domain.AdjustmentStatusApproved {
			balance := &domain.LeaveBalance{}
			if err := tx.First(balance, adjustment.LeaveBalanceID).Error; err != nil {
				return err
			}

			balance.TotalDays += adjustment.Adjustment
			if err := tx.Save(balance).Error; err != nil {
				return err
			}
		}

		return tx.Save(adjustment).Error
	})
}

func (r *leaveRepository) ListBalanceAdjustments(balanceID uuid.UUID) ([]domain.LeaveBalanceAdjustment, error) {
	var adjustments []domain.LeaveBalanceAdjustment
	err := r.db.Where("leave_balance_id = ?", balanceID).
		Order("created_at DESC").
		Find(&adjustments).Error
	return adjustments, err
}

// HasActiveLeaveRequests checks if there are any active leave requests for a leave type
func (r *leaveRepository) HasActiveLeaveRequests(leaveTypeID uuid.UUID) (bool, error) {
	var count int64
	err := r.db.Model(&domain.LeaveRequest{}).
		Where("leave_type_id = ? AND status IN (?)",
			leaveTypeID,
			[]string{domain.LeaveStatusPending, domain.LeaveStatusApproved}).
		Count(&count).Error

	if err != nil {
		return false, fmt.Errorf("failed to check active leave requests: %w", err)
	}

	return count > 0, nil
}
