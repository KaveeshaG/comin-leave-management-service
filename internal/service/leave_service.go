package service

import (
	"errors"
	"time"

	"github.com/Axontik/comin-leave-management-service/internal/domain"
	"github.com/Axontik/comin-leave-management-service/internal/repository"
	"github.com/google/uuid"
)

type LeaveService interface {
	CreateLeaveType(leaveType *domain.LeaveType) error
	GetLeaveType(orgID, id uuid.UUID) (*domain.LeaveType, error)
	UpdateLeaveType(leaveType *domain.LeaveType) error
	DeleteLeaveType(orgID, id uuid.UUID) error
	ListLeaveTypes(orgID uuid.UUID, params *domain.ListLeaveTypesParams) ([]domain.LeaveType, int64, error)
	CreateLeaveRequest(orgID uuid.UUID, req *domain.CreateLeaveRequestRequest) (*domain.LeaveRequest, error)

	CreateBalanceAdjustment(adjustment *domain.LeaveBalanceAdjustment) error
	UpdateLeaveBalance(balance *domain.LeaveBalance) error
	GetLeaveBalance(employeeID uuid.UUID, leaveTypeID uuid.UUID, year int) (*domain.LeaveBalance, error)
	ListLeaveBalances(employeeID uuid.UUID) ([]domain.LeaveBalance, error)
	UpdateLeaveRequest(request *domain.LeaveRequest) error
	GetLeaveRequest(id uuid.UUID) (*domain.LeaveRequest, error)
	DeleteLeaveRequest(id uuid.UUID) error
	ListLeaveRequests(orgID, employeeID uuid.UUID, status string) ([]domain.LeaveRequest, error)
	GetOverlappingRequests(employeeID uuid.UUID, startDate, endDate time.Time) ([]domain.LeaveRequest, error)
}

type leaveService struct {
	leaveRepo repository.LeaveRepository
}

func NewLeaveService(leaveRepo repository.LeaveRepository) LeaveService {
	return &leaveService{
		leaveRepo: leaveRepo,
	}
}

func (s *leaveService) CreateLeaveType(leaveType *domain.LeaveType) error {
	if err := validateLeaveType(leaveType); err != nil {
		return err
	}

	existingTypes, _, err := s.ListLeaveTypes(leaveType.OrganizationID, &domain.ListLeaveTypesParams{
		Name: leaveType.Name,
	})
	if err != nil {
		return err
	}
	if len(existingTypes) > 0 {
		return errors.New("leave type with this name already exists")
	}

	return s.leaveRepo.CreateLeaveType(leaveType)
}

func (s *leaveService) GetLeaveType(orgID, id uuid.UUID) (*domain.LeaveType, error) {
	leaveType, err := s.leaveRepo.GetLeaveType(id)
	if err != nil {
		return nil, err
	}

	if leaveType.OrganizationID != orgID {
		return nil, errors.New("leave type not found in organization")
	}

	return leaveType, nil
}

func (s *leaveService) UpdateLeaveType(leaveType *domain.LeaveType) error {
	if err := validateLeaveType(leaveType); err != nil {
		return err
	}

	existing, err := s.GetLeaveType(leaveType.OrganizationID, leaveType.ID)
	if err != nil {
		return err
	}

	if existing.Name != leaveType.Name {
		existingTypes, _, err := s.ListLeaveTypes(leaveType.OrganizationID, &domain.ListLeaveTypesParams{
			Name: leaveType.Name,
		})
		if err != nil {
			return err
		}
		if len(existingTypes) > 0 {
			return errors.New("leave type with this name already exists")
		}
	}

	return s.leaveRepo.UpdateLeaveType(leaveType)
}

func (s *leaveService) DeleteLeaveType(orgID, id uuid.UUID) error {
	existing, err := s.GetLeaveType(orgID, id)
	if err != nil {
		return err
	}

	hasActiveRequests, err := s.leaveRepo.HasActiveLeaveRequests(id)
	if err != nil {
		return err
	}
	if hasActiveRequests {
		return errors.New("cannot delete leave type with active leave requests")
	}

	return s.leaveRepo.DeleteLeaveType(existing.ID)
}

func (s *leaveService) ListLeaveTypes(orgID uuid.UUID, params *domain.ListLeaveTypesParams) ([]domain.LeaveType, int64, error) {
	if params != nil {
		if params.Page < 1 {
			params.Page = 1
		}
		if params.PageSize < 1 || params.PageSize > 100 {
			params.PageSize = 10
		}
	}

	return s.leaveRepo.ListLeaveTypesWithOptions(orgID, params)
}

func validateLeaveType(leaveType *domain.LeaveType) error {
	if leaveType.Name == "" {
		return errors.New("name is required")
	}
	if leaveType.DefaultDays < 0 {
		return errors.New("default days cannot be negative")
	}
	if leaveType.MaxDaysPerRequest < 1 {
		return errors.New("max days per request must be at least 1")
	}
	if leaveType.MinDaysNotice < 0 {
		return errors.New("minimum days notice cannot be negative")
	}
	return nil
}

func (s *leaveService) CreateLeaveRequest(orgID uuid.UUID, req *domain.CreateLeaveRequestRequest) (*domain.LeaveRequest, error) {
	if req.EmployeeID == uuid.Nil {
		return nil, errors.New("employee ID is required")
	}
	if req.LeaveTypeID == uuid.Nil {
		return nil, errors.New("leave type ID is required")
	}
	if req.StartDate.After(req.EndDate) {
		return nil, errors.New("start date cannot be after end date")
	}

	leaveType, err := s.GetLeaveType(orgID, req.LeaveTypeID)
	if err != nil {
		return nil, err
	}

	totalDays := int(req.EndDate.Sub(req.StartDate).Milliseconds() / 86400000)
	if totalDays > leaveType.MaxDaysPerRequest {
		return nil, errors.New("total days exceed maximum allowed")
	}

	leaveRequest := &domain.LeaveRequest{
		EmployeeID:  req.EmployeeID,
		LeaveTypeID: req.LeaveTypeID,
		StartDate:   req.StartDate,
		EndDate:     req.EndDate,
		Status:      domain.LeaveStatusPending,
		Reason:      req.Reason,
	}

	if err := s.leaveRepo.CreateLeaveRequest(leaveRequest); err != nil {
		return nil, err
	}

	return leaveRequest, nil
}

func (s *leaveService) CreateBalanceAdjustment(adjustment *domain.LeaveBalanceAdjustment) error {
	return s.leaveRepo.CreateBalanceAdjustment(adjustment)
}

func (s *leaveService) UpdateLeaveBalance(balance *domain.LeaveBalance) error {
	return s.leaveRepo.UpdateLeaveBalance(balance)
}

func (s *leaveService) GetLeaveBalance(employeeID uuid.UUID, leaveTypeID uuid.UUID, year int) (*domain.LeaveBalance, error) {
	// Validate input
	if employeeID == uuid.Nil {
		return nil, errors.New("employee ID is required")
	}
	if leaveTypeID == uuid.Nil {
		return nil, errors.New("leave type ID is required")
	}
	if year < 0 {
		return nil, errors.New("year cannot be negative")
	}

	// Get leave balance
	balance, err := s.leaveRepo.GetLeaveBalance(employeeID, leaveTypeID, year)
	if err != nil {
		return nil, err
	}

	return balance, nil
}

func (s *leaveService) DeleteLeaveRequest(id uuid.UUID) error {
	panic("unimplemented")
}

func (s *leaveService) GetLeaveRequest(id uuid.UUID) (*domain.LeaveRequest, error) {
	leaveRequest, err := s.leaveRepo.GetLeaveRequest(id)
	if err != nil {
		return nil, err
	}

	return leaveRequest, nil
}

func (s *leaveService) GetOverlappingRequests(employeeID uuid.UUID, startDate time.Time, endDate time.Time) ([]domain.LeaveRequest, error) {
	if employeeID == uuid.Nil {
		return nil, errors.New("employee ID is required")
	}
	if startDate.After(endDate) {
		return nil, errors.New("start date cannot be after end date")
	}
	leaveRequests, err := s.leaveRepo.GetOverlappingRequests(employeeID, startDate, endDate)
	if err != nil {
		return nil, err
	}

	return leaveRequests, nil
}

func (s *leaveService) ListLeaveBalances(employeeID uuid.UUID) ([]domain.LeaveBalance, error) {
	if employeeID == uuid.Nil {
		return nil, errors.New("employee ID is required")
	}
	leaveBalances, err := s.leaveRepo.ListLeaveBalances(employeeID)
	if err != nil {
		return nil, err
	}

	return leaveBalances, nil
}

func (s *leaveService) ListLeaveRequests(orgID uuid.UUID, employeeID uuid.UUID, status string) ([]domain.LeaveRequest, error) {
	if orgID == uuid.Nil {
		return nil, errors.New("organization ID is required")
	}
	if employeeID == uuid.Nil {
		return nil, errors.New("employee ID is required")
	}
	if status == "" {
		return nil, errors.New("status is required")
	}
	leaveRequests, err := s.leaveRepo.ListLeaveRequests(orgID, employeeID, status)
	if err != nil {
		return nil, err
	}

	return leaveRequests, nil
}

func (s *leaveService) UpdateLeaveRequest(request *domain.LeaveRequest) error {
	if request == nil {
		return errors.New("request is required")
	}
	err := s.leaveRepo.UpdateLeaveRequest(request)
	if err != nil {
		return err
	}
	return nil
}
