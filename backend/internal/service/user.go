package service

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"strings"

	applog "github.com/knowledgeos/backend/internal/logger"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/knowledgeos/backend/internal/auth"
	"github.com/knowledgeos/backend/internal/domain"
)

const minPasswordLength = 8

var emailRegex = regexp.MustCompile(`^[^@\s]+@[^@\s]+\.[^@\s]+$`)

type UserService struct {
	users  domain.UserRepository
	tokens domain.TokenRepository
}

// NewUserService executes the service.NewUserService operation.
func NewUserService(users domain.UserRepository, tokens domain.TokenRepository) *UserService {
	return &UserService{users: users, tokens: tokens}
}

// Actor identifies the authenticated caller performing the operation.
type Actor struct {
	ID        uuid.UUID
	Role      domain.Role
	CompanyID uuid.UUID
}

type CreateUserRequest struct {
	Email      string      `json:"email"`
	Password   string      `json:"password"`
	Role       domain.Role `json:"role"`
	IsActive   *bool       `json:"is_active"`
	CompanyIDs []uuid.UUID `json:"company_ids"`
}

type UpdateUserRequest struct {
	Email      *string      `json:"email"`
	Password   *string      `json:"password"`
	Role       *domain.Role `json:"role"`
	IsActive   *bool        `json:"is_active"`
	CompanyIDs *[]uuid.UUID `json:"company_ids"`
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
	if req.Role == domain.RoleSuperadmin && actor.Role != domain.RoleSuperadmin {
		return nil, forbidden("only a superadmin can assign the superadmin role")
	}

	companyIDs, err := s.resolveCreateCompanyIDs(ctx, actor, req.Role, req.CompanyIDs)
	if err != nil {
		return nil, err
	}

	if _, err := s.users.GetByEmail(ctx, email); err == nil {
		return nil, conflict("a user with this email already exists")
	} else if err != nil && !errors.Is(err, domain.ErrNotFound) {
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

	user := &domain.User{
		Email:        email,
		PasswordHash: hash,
		Role:         req.Role,
		IsActive:     active,
	}

	if req.Role == domain.RoleSuperadmin {
		if err := s.users.Create(ctx, user); err != nil {
			if isUniqueViolation(err) {
				return nil, conflict("a user with this email already exists")
			}
			return nil, err
		}
		return user, nil
	}

	if err := s.users.CreateWithCompanies(ctx, user, companyIDs); err != nil {
		if isUniqueViolation(err) {
			return nil, conflict("a user with this email already exists")
		}
		return nil, err
	}
	user.CompanyIDs = companyIDs
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

	if !isSelf && actor.Role != domain.RoleSuperadmin && isPrivilegedRole(target.Role) {
		return nil, forbidden("only a superadmin can modify an administrator account")
	}

	revokeSessions := false

	if req.Role != nil && *req.Role != target.Role {
		if !domain.ValidRole(*req.Role) {
			return nil, badRequest("invalid role")
		}
		if isSelf {
			return nil, forbidden("you cannot change your own role")
		}
		if (*req.Role == domain.RoleSuperadmin || target.Role == domain.RoleSuperadmin) && actor.Role != domain.RoleSuperadmin {
			return nil, forbidden("only a superadmin can grant or revoke the superadmin role")
		}
		if *req.Role == domain.RoleSuperadmin {
			if err := s.users.SetCompanyIDs(ctx, target.ID, nil); err != nil {
				return nil, err
			}
			target.CompanyIDs = nil
		} else if target.Role == domain.RoleSuperadmin {
			if req.CompanyIDs == nil || len(*req.CompanyIDs) == 0 {
				return nil, badRequest("at least one company is required")
			}
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
			} else if err != nil && !errors.Is(err, domain.ErrNotFound) {
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

	if req.CompanyIDs != nil && target.Role != domain.RoleSuperadmin {
		merged, err := s.mergeCompanyIDs(ctx, actor, target.ID, *req.CompanyIDs)
		if err != nil {
			return nil, err
		}
		if err := s.users.SetCompanyIDs(ctx, target.ID, merged); err != nil {
			return nil, err
		}
		target.CompanyIDs = merged
		revokeSessions = true
	}

	if err := s.users.Update(ctx, target); err != nil {
		if isUniqueViolation(err) {
			return nil, conflict("a user with this email already exists")
		}
		return nil, err
	}

	if revokeSessions && s.tokens != nil {
		if err := s.tokens.RevokeAllUserTokens(ctx, target.ID); err != nil {
			applog.TraceErr(ctx, "revoke user sessions failed", err)
		}
	}
	return target, nil
}

// Delete executes the service.UserService.Delete operation.
func (s *UserService) Delete(ctx context.Context, actor Actor, id uuid.UUID) error {
	applog.TraceCall(ctx, "service.UserService.Delete")
	target, err := s.loadInCompany(ctx, actor.CompanyID, id)
	if err != nil {
		return applog.TraceErr(ctx, "delete user failed", err)
	}
	if target.ID == actor.ID {
		return forbidden("you cannot delete your own account")
	}
	if actor.Role != domain.RoleSuperadmin && isPrivilegedRole(target.Role) {
		return forbidden("only a superadmin can delete an administrator account")
	}
	return s.users.Delete(ctx, id)
}

func isPrivilegedRole(r domain.Role) bool {
	return r == domain.RoleAdmin || r == domain.RoleSuperadmin
}

func (s *UserService) loadInCompany(ctx context.Context, companyID, id uuid.UUID) (*domain.User, error) {
	applog.TraceCall(ctx, "service.UserService.loadInCompany")
	user, err := s.users.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return nil, notFound("user not found")
		}
		return nil, err
	}
	if user.Role == domain.RoleSuperadmin {
		return nil, notFound("user not found")
	}
	companyIDs, err := s.users.GetCompanyIDs(ctx, id)
	if err != nil {
		return nil, err
	}
	if !containsCompanyID(companyIDs, companyID) {
		return nil, notFound("user not found")
	}
	user.CompanyIDs = companyIDs
	return user, nil
}

func (s *UserService) resolveCreateCompanyIDs(ctx context.Context, actor Actor, role domain.Role, requested []uuid.UUID) ([]uuid.UUID, error) {
	if role == domain.RoleSuperadmin {
		return nil, nil
	}
	if len(requested) == 0 {
		return nil, badRequest("at least one company is required")
	}
	if err := s.validateActorCanAssign(ctx, actor, requested); err != nil {
		return nil, err
	}
	return dedupeUUIDs(requested), nil
}

func (s *UserService) validateActorCanAssign(ctx context.Context, actor Actor, requested []uuid.UUID) error {
	if actor.Role == domain.RoleSuperadmin {
		return nil
	}
	actorCompanies, err := s.users.GetCompanyIDs(ctx, actor.ID)
	if err != nil {
		return err
	}
	allowed := uuidSet(actorCompanies)
	for _, id := range requested {
		if !allowed[id] {
			return forbidden("you cannot assign one or more of the selected companies")
		}
	}
	return nil
}

func (s *UserService) mergeCompanyIDs(ctx context.Context, actor Actor, targetUserID uuid.UUID, requested []uuid.UUID) ([]uuid.UUID, error) {
	requested = dedupeUUIDs(requested)
	if len(requested) == 0 {
		return nil, badRequest("at least one company is required")
	}

	if actor.Role == domain.RoleSuperadmin {
		return requested, nil
	}

	existing, err := s.users.GetCompanyIDs(ctx, targetUserID)
	if err != nil {
		return nil, err
	}
	actorCompanies, err := s.users.GetCompanyIDs(ctx, actor.ID)
	if err != nil {
		return nil, err
	}
	allowed := uuidSet(actorCompanies)

	result := make([]uuid.UUID, 0, len(existing)+len(requested))
	for _, id := range existing {
		if !allowed[id] {
			result = append(result, id)
		}
	}
	for _, id := range requested {
		if allowed[id] {
			result = append(result, id)
		}
	}
	result = dedupeUUIDs(result)
	if len(result) == 0 {
		return nil, badRequest("at least one company is required")
	}
	return result, nil
}

func uuidSet(ids []uuid.UUID) map[uuid.UUID]bool {
	set := make(map[uuid.UUID]bool, len(ids))
	for _, id := range ids {
		set[id] = true
	}
	return set
}

func dedupeUUIDs(ids []uuid.UUID) []uuid.UUID {
	if len(ids) == 0 {
		return nil
	}
	seen := make(map[uuid.UUID]bool, len(ids))
	out := make([]uuid.UUID, 0, len(ids))
	for _, id := range ids {
		if seen[id] {
			continue
		}
		seen[id] = true
		out = append(out, id)
	}
	return out
}

func isUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == "23505" {
		return true
	}
	return err != nil && strings.Contains(strings.ToLower(err.Error()), "duplicate")
}
