package auth

import (
	"github.com/cnative/pkg/log"
)

type (
	// Option configures choices
	Option interface {
		apply(*runtime)
	}
	optionFunc func(*runtime)
)

// Logger for runtime
func Logger(l *log.Logger) Option {
	return optionFunc(func(r *runtime) {
		r.logger = l.NamedLogger("auth")
	})
}

// OIDCIssuer OIDC token issuer
func OIDCIssuer(iss string) Option {
	return optionFunc(func(r *runtime) {
		r.issuer = iss
	})
}

// OIDCAudience OIDC Audience which is the OIDC Client ID
func OIDCAudience(aud string) Option {
	return optionFunc(func(r *runtime) {
		r.aud = aud
	})
}

// OIDCCAFile CA file
func OIDCCAFile(caFile string) Option {
	return optionFunc(func(r *runtime) {
		r.caFile = caFile
	})
}

// OIDCSigningAlgos OIDC Signing Algos
func OIDCSigningAlgos(signingAlgos []string) Option {
	return optionFunc(func(r *runtime) {
		r.signingAlgos = signingAlgos
	})
}

// OIDCRequiredClaims OIDC Required Claims
func OIDCRequiredClaims(requiredClaims map[string]string) Option {
	return optionFunc(func(r *runtime) {
		r.requiredClaims = requiredClaims
	})
}

// Authorizer performs authz for every request
func Authorizer(authorizer AuthorizerFn) Option {
	return optionFunc(func(r *runtime) {
		r.authorizer = authorizer
	})
}

// IDResolver to resolve the ID for authenticated user
func IDResolver(idResolver IDResolverFn) Option {
	return optionFunc(func(r *runtime) {
		r.idResolver = idResolver
	})
}

// AdditionalClaimsProvider creates and returns an additional claims object which is will be filled during identity token verification
func AdditionalClaimsProvider(additionalClaimsProvider AddtionalClaimsProviderFn) Option {
	return optionFunc(func(r *runtime) {
		r.additionalClaimsProvider = additionalClaimsProvider
	})
}
