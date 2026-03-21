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
	entityTypeKey contextKey = "entity_type"
)

func SetClaims(ctx context.Context, claims *auth.Claims) context.Context {
	return context.WithValue(ctx, claimsKey, claims)
}

func GetClaims(ctx context.Context) *auth.Claims {
	claims, _ := ctx.Value(claimsKey).(*auth.Claims)
	return claims
}

func SetCompanyID(ctx context.Context, id uuid.UUID) context.Context {
	return context.WithValue(ctx, companyIDKey, id)
}

func GetCompanyID(ctx context.Context) uuid.UUID {
	id, _ := ctx.Value(companyIDKey).(uuid.UUID)
	return id
}

func GetUserID(ctx context.Context) uuid.UUID {
	claims := GetClaims(ctx)
	if claims == nil {
		return uuid.Nil
	}
	id, _ := uuid.Parse(claims.Subject)
	return id
}

func GetRole(ctx context.Context) domain.Role {
	claims := GetClaims(ctx)
	if claims == nil {
		return ""
	}
	return claims.Role
}

func SetEntityType(ctx context.Context, et string) context.Context {
	return context.WithValue(ctx, entityTypeKey, et)
}

func GetEntityType(ctx context.Context) string {
	et, _ := ctx.Value(entityTypeKey).(string)
	return et
}
