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

// AppName for auhtz runtime
func AppName(appName string) Option {
	return optionFunc(func(r *runtime) {
		r.appName = appName
	})
}

// ServiceName for authz runtime
func ServiceName(serviceName string) Option {
	return optionFunc(func(r *runtime) {
		r.serviceName = serviceName
	})
}

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

// RoleBindingResolver uses this resolver to map a subject to a set of roles
func RoleBindingResolver(roleBindingResolver RoleBindingResolverFn) Option {
	return optionFunc(func(r *runtime) {
		r.roleBindingResolver = roleBindingResolver
	})
}

// ResourceResolver uses this resolver to lookup the resource attributes that can be used for authorization checks
func ResourceResolver(resourceResolver ResourceResolverFn) Option {
	return optionFunc(func(r *runtime) {
		r.resourceResolver = resourceResolver
	})
}

// ResourceIdentifier look at incoming request and identify the resource
func ResourceIdentifier(resourceIdentifier ResourceIdentifierFn) Option {
	return optionFunc(func(r *runtime) {
		r.resourceIdentifier = resourceIdentifier
	})
}

// AdminGroupRoleMapping maps the adminGroup to the specified adminRole for authorization request. the group assignment and resolution happens externally
func AdminGroupRoleMapping(adminGroup, adminRole string) Option {
	return optionFunc(func(r *runtime) {
		r.adminGroup = adminGroup
		r.adminRole = adminRole
	})
}
