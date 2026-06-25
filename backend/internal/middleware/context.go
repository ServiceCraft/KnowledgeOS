package middleware

import (
	"context"
	applog "github.com/knowledgeos/backend/internal/logger"

	"github.com/google/uuid"
	"github.com/knowledgeos/backend/internal/auth"
	"github.com/knowledgeos/backend/internal/domain"
)

type contextKey string

const (
	claimsKey     contextKey = "claims"
	companyIDKey  contextKey = "company_id"
	entityTypeKey contextKey = "entity_type"
	requestIDKey  contextKey = "request_id"
)

// SetClaims executes the middleware.SetClaims operation.
func SetClaims(ctx context.Context, claims *auth.Claims) context.Context {
	applog.TraceCall(ctx, "middleware.SetClaims")
	return context.WithValue(ctx, claimsKey, claims)
}

// GetClaims executes the middleware.GetClaims operation.
func GetClaims(ctx context.Context) *auth.Claims {
	applog.TraceCall(ctx, "middleware.GetClaims")
	claims, _ := ctx.Value(claimsKey).(*auth.Claims)
	return claims
}

// SetCompanyID executes the middleware.SetCompanyID operation.
func SetCompanyID(ctx context.Context, id uuid.UUID) context.Context {
	applog.TraceCall(ctx, "middleware.SetCompanyID")
	return context.WithValue(ctx, companyIDKey, id)
}

// GetCompanyID executes the middleware.GetCompanyID operation.
func GetCompanyID(ctx context.Context) uuid.UUID {
	applog.TraceCall(ctx, "middleware.GetCompanyID")
	id, _ := ctx.Value(companyIDKey).(uuid.UUID)
	return id
}

// GetUserID executes the middleware.GetUserID operation.
func GetUserID(ctx context.Context) uuid.UUID {
	applog.TraceCall(ctx, "middleware.GetUserID")
	claims := GetClaims(ctx)
	if claims == nil {
		return uuid.Nil
	}
	id, _ := uuid.Parse(claims.Subject)
	return id
}

// GetRole executes the middleware.GetRole operation.
func GetRole(ctx context.Context) domain.Role {
	applog.TraceCall(ctx, "middleware.GetRole")
	claims := GetClaims(ctx)
	if claims == nil {
		return ""
	}
	return claims.Role
}

// SetEntityType executes the middleware.SetEntityType operation.
func SetEntityType(ctx context.Context, et string) context.Context {
	applog.TraceCall(ctx, "middleware.SetEntityType")
	return context.WithValue(ctx, entityTypeKey, et)
}

// GetEntityType executes the middleware.GetEntityType operation.
func GetEntityType(ctx context.Context) string {
	applog.TraceCall(ctx, "middleware.GetEntityType")
	et, _ := ctx.Value(entityTypeKey).(string)
	return et
}

// SetRequestID executes the middleware.SetRequestID operation.
func SetRequestID(ctx context.Context, id string) context.Context {
	applog.TraceCall(ctx, "middleware.SetRequestID")
	return context.WithValue(ctx, requestIDKey, id)
}

// GetRequestID executes the middleware.GetRequestID operation.
func GetRequestID(ctx context.Context) string {
	applog.TraceCall(ctx, "middleware.GetRequestID")
	id, _ := ctx.Value(requestIDKey).(string)
	return id
}
