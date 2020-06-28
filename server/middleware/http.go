package middleware

import (
	"crypto/subtle"
	"net/http"
	"strings"

	"github.com/cnative/pkg/auth"
)

// HTTPBasicAuth wraps the HTTP handler function with Basic Auth
func HTTPBasicAuth(handler http.HandlerFunc, username, password string) http.HandlerFunc {

	return func(w http.ResponseWriter, r *http.Request) {

		user, pass, ok := r.BasicAuth()
		if !ok || subtle.ConstantTimeCompare([]byte(user), []byte(username)) != 1 || subtle.ConstantTimeCompare([]byte(pass), []byte(password)) != 1 {
			http.Error(w, "Unauthorized.\n", http.StatusUnauthorized)
			return
		}

		handler(w, r)
	}
}

// HTTPRuntimeIDAuth Wraps will return a new http.Handler that will enforce auth as configured
func HTTPRuntimeIDAuth(authRuntime auth.Runtime, wrapped http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		reqToken := r.Header.Get("Authorization")
		sp := strings.Split(reqToken, "Bearer")
		if len(sp) != 2 {
			http.Error(w, "Unauthorized.\n", http.StatusUnauthorized)
			return
		}
		reqToken = sp[1]

		ctx, c, err := authRuntime.Verify(r.Context(), reqToken)
		if err != nil {
			http.Error(w, "Unauthorized.\n", http.StatusUnauthorized)
			return
		}

		// TODO(vshiva): resolve req, resource and action.
		// Until this is fixed all request submitted via gateway is expected to fail
		ctx, authzResult, err := authRuntime.Authorize(ctx, c, "", "", nil)
		if err != nil || !authzResult.Allowed {
			http.Error(w, "Forbidden.\n", http.StatusForbidden)
			return
		}

		r = r.WithContext(ctx)

		wrapped.ServeHTTP(w, r)
	})
}
