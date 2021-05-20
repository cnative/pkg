package auth

import "context"

type contextKey string

// Anonymous user
const Anonymous = "anonymous"

func (c contextKey) String() string {
	return string(c)
}

var (
	contextKeyAuthenticated = contextKey("authn")
	contextKeyAuthorized    = contextKey("authz")

	userName   = "current-user"
	userClaims = "claims"
	userRoles  = "roles"
)

// CurrentUser for the request
func CurrentUser(ctx context.Context) string {

	if v, ok := ctx.Value(contextKeyAuthenticated).(map[string]interface{}); ok {
		if u, ok := v[userName]; ok {
			return u.(string)
		}
	}

	return Anonymous
}

// CurrentUserClaims for the request
func CurrentUserClaims(ctx context.Context) Claims {

	if v, ok := ctx.Value(contextKeyAuthenticated).(map[string]interface{}); ok {
		if u, ok := v[userClaims]; ok {
			return u.(Claims)
		}
	}

	return nil
}

// CurrentUserRoles for the request
func CurrentUserRoles(ctx context.Context) []string {

	if v, ok := ctx.Value(contextKeyAuthorized).(map[string]interface{}); ok {
		if u, ok := v[userRoles]; ok {
			return u.([]string)
		}
	}

	return nil
}

// returns a new context with the given user and the claims attached
func newAuthenticatedContext(parent context.Context, user string, cl Claims) context.Context {
	return context.WithValue(parent, contextKeyAuthenticated, map[string]interface{}{
		userName:   user,
		userClaims: cl,
	})
}

// returns a new context with the given authz result attached
func newAuthorizedContext(parent context.Context, roles []string) context.Context {
	return context.WithValue(parent, contextKeyAuthorized, map[string]interface{}{
		userRoles: roles,
	})
}
