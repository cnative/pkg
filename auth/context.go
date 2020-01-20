package auth

import "context"

type contextKey string

// Anonymous user
const Anonymous = "anonymous"

func (c contextKey) String() string {
	return string(c)
}

var (
	// contextKeyAuthtoken   = contextKey("auth-token")
	contextKeyCurrentUser = contextKey("current-user")
)

// CurrentUser for the request
func CurrentUser(ctx context.Context) string {

	if v, ok := ctx.Value(contextKeyCurrentUser).(string); ok {
		return v
	}

	return Anonymous
}

// newContext returns a new context with the given user attached
func newContext(parent context.Context, user string) context.Context {
	return context.WithValue(parent, contextKeyCurrentUser, user)
}
