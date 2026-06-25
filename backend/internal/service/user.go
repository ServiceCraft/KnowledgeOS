package service

import (
	"context"
	"errors"
	"fmt"
	applog "github.com/knowledgeos/backend/internal/logger"
	"net/http"
	"regexp"
	"strings"

	"github.com/google/uuid"
	"github.com/knowledgeos/backend/internal/auth"
	"github.com/knowledgeos/backend/internal/domain"
	"gorm.io/gorm"
)

const minPasswordLength = 8

var emailRegex = regexp.MustCompile(`^[^@\s]+@[^@\s]+\.[^@\s]+$`)

// UserError carries an HTTP status so the handler can map service failures to
// the right response code without leaking implementation details.
type UserError struct {
	Status int
	Msg    string
}

// Error executes the service.UserError.Error operation.
func (e *UserError) Error() string { return e.Msg }

func badRequest(msg string) error { return &UserError{Status: http.StatusBadRequest, Msg: msg} }
func forbidden(msg string) error  { return &UserError{Status: http.StatusForbidden, Msg: msg} }
func notFound(msg string) error   { return &UserError{Status: http.StatusNotFound, Msg: msg} }
func conflict(msg string) error   { return &UserError{Status: http.StatusConflict, Msg: msg} }

// HTTPStatus extracts the status code from an error returned by UserService,
// defaulting to 500 for unexpected errors.
func HTTPStatus(err error) int {
	var ue *UserError
	if errors.As(err, &ue) {
		return ue.Status
	}
	return http.StatusInternalServerError
}

type UserService struct {
	users  domain.UserRepository
	tokens domain.SyncRepository
}

// NewUserService executes the service.NewUserService operation.
func NewUserService(users domain.UserRepository, tokens domain.SyncRepository) *UserService {
	return &UserService{users: users, tokens: tokens}
}

// Actor identifies the authenticated caller performing the operation.
type Actor struct {
	ID        uuid.UUID
	Role      domain.Role
	CompanyID uuid.UUID
}

type CreateUserRequest struct {
	Email    string      `json:"email"`
	Password string      `json:"password"`
	Role     domain.Role `json:"role"`
	IsActive *bool       `json:"is_active"`
}

type UpdateUserRequest struct {
	Email    *string      `json:"email"`
	Password *string      `json:"password"`
	Role     *domain.Role `json:"role"`
	IsActive *bool        `json:"is_active"`
}

// List executes the service.UserService.List operation.
func (s *UserService) List(ctx context.Context, companyID uuid.UUID, filter domain.UserFilter) ([]domain.User, int64, error) {
	applog.TraceCall(ctx, "service.UserService.List")
	return s.users.List(ctx, companyID, filter)
}

// ValidateNewCredentials normalizes the email and enforces the email-format and
// minimum-password-length rules shared by all account-creation paths.
func ValidateNewCredentials(email, password string) (string, error) {
	normalized := strings.TrimSpace(strings.ToLower(email))
	if !emailRegex.MatchString(normalized) {
		return "", badRequest("invalid email")
	}
	if len(password) < minPasswordLength {
		return "", badRequest(fmt.Sprintf("password must be at least %d characters", minPasswordLength))
	}
	return normalized, nil
}

// Create executes the service.UserService.Create operation.
func (s *UserService) Create(ctx context.Context, actor Actor, req CreateUserRequest) (*domain.User, error) {
	applog.TraceCall(ctx, "service.UserService.Create")
	email, err := ValidateNewCredentials(req.Email, req.Password)
	if err != nil {
		return nil, err
	}
	if !domain.ValidRole(req.Role) {
		return nil, badRequest("invalid role")
	}
	// Only a superadmin may mint another superadmin.
	if req.Role == domain.RoleSuperadmin && actor.Role != domain.RoleSuperadmin {
		return nil, forbidden("only a superadmin can assign the superadmin role")
	}

	if _, err := s.users.GetByEmail(ctx, email); err == nil {
		return nil, conflict("a user with this email already exists")
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}

	hash, err := auth.HashPassword(req.Password)
	if err != nil {
		return nil, err
	}

	active := true
	if req.IsActive != nil {
		active = *req.IsActive
	}

	companyID := actor.CompanyID
	user := &domain.User{
		CompanyID:    &companyID,
		Email:        email,
		PasswordHash: hash,
		Role:         req.Role,
		IsActive:     active,
	}
	if err := s.users.Create(ctx, user); err != nil {
		if isUniqueViolation(err) {
			return nil, conflict("a user with this email already exists")
		}
		return nil, err
	}
	return user, nil
}

// Update executes the service.UserService.Update operation.
func (s *UserService) Update(ctx context.Context, actor Actor, id uuid.UUID, req UpdateUserRequest) (*domain.User, error) {
	applog.TraceCall(ctx, "service.UserService.Update")
	target, err := s.loadInCompany(ctx, actor.CompanyID, id)
	if err != nil {
		return nil, err
	}

	isSelf := target.ID == actor.ID

	// Administrator and superadmin accounts can only be managed by a superadmin.
	// An administrator may still edit their OWN account, but not other admins.
	if !isSelf && actor.Role != domain.RoleSuperadmin && isPrivilegedRole(target.Role) {
		return nil, forbidden("only a superadmin can modify an administrator account")
	}

	// Whether this change should terminate the target's existing sessions.
	revokeSessions := false

	if req.Role != nil && *req.Role != target.Role {
		if !domain.ValidRole(*req.Role) {
			return nil, badRequest("invalid role")
		}
		if isSelf {
			return nil, forbidden("you cannot change your own role")
		}
		// Granting or revoking superadmin is reserved for superadmins.
		if (*req.Role == domain.RoleSuperadmin || target.Role == domain.RoleSuperadmin) && actor.Role != domain.RoleSuperadmin {
			return nil, forbidden("only a superadmin can grant or revoke the superadmin role")
		}
		target.Role = *req.Role
		revokeSessions = true
	}

	if req.Email != nil {
		email := strings.TrimSpace(strings.ToLower(*req.Email))
		if !emailRegex.MatchString(email) {
			return nil, badRequest("invalid email")
		}
		if email != target.Email {
			if existing, err := s.users.GetByEmail(ctx, email); err == nil && existing.ID != target.ID {
				return nil, conflict("a user with this email already exists")
			} else if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
				return nil, err
			}
			target.Email = email
		}
	}

	if req.Password != nil {
		if len(*req.Password) < minPasswordLength {
			return nil, badRequest(fmt.Sprintf("password must be at least %d characters", minPasswordLength))
		}
		hash, err := auth.HashPassword(*req.Password)
		if err != nil {
			return nil, err
		}
		target.PasswordHash = hash
		revokeSessions = true
	}

	if req.IsActive != nil && *req.IsActive != target.IsActive {
		if isSelf && !*req.IsActive {
			return nil, forbidden("you cannot deactivate your own account")
		}
		target.IsActive = *req.IsActive
		if !target.IsActive {
			revokeSessions = true
		}
	}

	if err := s.users.Update(ctx, target); err != nil {
		if isUniqueViolation(err) {
			return nil, conflict("a user with this email already exists")
		}
		return nil, err
	}

	// Terminate outstanding sessions after a sensitive change (role, password,
	// deactivation). Best-effort: the update is already committed.
	if revokeSessions && s.tokens != nil {
		_ = s.tokens.RevokeAllUserTokens(ctx, target.ID)
	}
	return target, nil
}

// Delete executes the service.UserService.Delete operation.
func (s *UserService) Delete(ctx context.Context, actor Actor, id uuid.UUID) error {
	applog.TraceCall(ctx, "service.UserService.Delete")
	target, err := s.loadInCompany(ctx, actor.CompanyID, id)
	if err != nil {
		return err
	}
	if target.ID == actor.ID {
		return forbidden("you cannot delete your own account")
	}
	// Deleting an administrator or superadmin is reserved for superadmins.
	if actor.Role != domain.RoleSuperadmin && isPrivilegedRole(target.Role) {
		return forbidden("only a superadmin can delete an administrator account")
	}
	return s.users.Delete(ctx, id)
}

// isPrivilegedRole reports whether a role (admin or superadmin) may only be
// managed by a superadmin.
func isPrivilegedRole(r domain.Role) bool {
	return r == domain.RoleAdmin || r == domain.RoleSuperadmin
}

// loadInCompany fetches a user and enforces multi-tenant isolation: a user from
// another company is reported as not found.
func (s *UserService) loadInCompany(ctx context.Context, companyID, id uuid.UUID) (*domain.User, error) {
	applog.TraceCall(ctx, "service.UserService.loadInCompany")
	user, err := s.users.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, notFound("user not found")
		}
		return nil, err
	}
	if user.CompanyID == nil || *user.CompanyID != companyID {
		return nil, notFound("user not found")
	}
	return user, nil
}

func isUniqueViolation(err error) bool {
	return err != nil && strings.Contains(strings.ToLower(err.Error()), "duplicate")
}
