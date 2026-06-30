package middleware

import (
	"context"

	"github.com/google/uuid"
	"github.com/knowledgeos/backend/internal/auth"
	"github.com/knowledgeos/backend/internal/domain"
)

type contextKey string

const (
	claimsKey     contextKey = "claims"
	companyIDKey  contextKey = "company_id"
	tenantKey     contextKey = "tenant"
	entityTypeKey contextKey = "entity_type"
	requestIDKey  contextKey = "request_id"
)

// TenantContext carries authenticated identity and active company for tenant routes.
type TenantContext struct {
	UserID    uuid.UUID
	Role      domain.Role
	CompanyID uuid.UUID
}

// SetClaims stores JWT claims on the context.
func SetClaims(ctx context.Context, claims *auth.Claims) context.Context {
	return context.WithValue(ctx, claimsKey, claims)
}

// GetClaims returns JWT claims from the context.
func GetClaims(ctx context.Context) *auth.Claims {
	claims, _ := ctx.Value(claimsKey).(*auth.Claims)
	return claims
}

// SetTenant stores resolved tenant context.
func SetTenant(ctx context.Context, tenant TenantContext) context.Context {
	ctx = context.WithValue(ctx, tenantKey, tenant)
	return context.WithValue(ctx, companyIDKey, tenant.CompanyID)
}

// TenantFromContext returns tenant context when present.
func TenantFromContext(ctx context.Context) (TenantContext, bool) {
	tenant, ok := ctx.Value(tenantKey).(TenantContext)
	return tenant, ok
}

// SetCompanyID sets the active company without full tenant context (webhooks/sync).
func SetCompanyID(ctx context.Context, id uuid.UUID) context.Context {
	return context.WithValue(ctx, companyIDKey, id)
}

// GetCompanyID returns the active company ID from tenant or legacy context value.
func GetCompanyID(ctx context.Context) uuid.UUID {
	if tenant, ok := TenantFromContext(ctx); ok {
		return tenant.CompanyID
	}
	id, _ := ctx.Value(companyIDKey).(uuid.UUID)
	return id
}

// GetUserID returns the authenticated user ID.
func GetUserID(ctx context.Context) uuid.UUID {
	if tenant, ok := TenantFromContext(ctx); ok && tenant.UserID != uuid.Nil {
		return tenant.UserID
	}
	claims := GetClaims(ctx)
	if claims == nil {
		return uuid.Nil
	}
	id, _ := uuid.Parse(claims.Subject)
	return id
}

// GetRole returns the authenticated user role.
func GetRole(ctx context.Context) domain.Role {
	if tenant, ok := TenantFromContext(ctx); ok && tenant.Role != "" {
		return tenant.Role
	}
	claims := GetClaims(ctx)
	if claims == nil {
		return ""
	}
	return claims.Role
}

// SetEntityType stores the polymorphic entity type route parameter.
func SetEntityType(ctx context.Context, et string) context.Context {
	return context.WithValue(ctx, entityTypeKey, et)
}

// GetEntityType returns the polymorphic entity type route parameter.
func GetEntityType(ctx context.Context) string {
	et, _ := ctx.Value(entityTypeKey).(string)
	return et
}

// SetRequestID stores the request correlation ID.
func SetRequestID(ctx context.Context, id string) context.Context {
	return context.WithValue(ctx, requestIDKey, id)
}

// GetRequestID returns the request correlation ID.
func GetRequestID(ctx context.Context) string {
	id, _ := ctx.Value(requestIDKey).(string)
	return id
}
