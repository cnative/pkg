package auth

import (
	"context"

	"github.com/pkg/errors"

	"github.com/coreos/go-oidc"

	"github.com/cnative/pkg/log"
)

// AuthorizationRequest describes information required (who and what) to perform authorization check
// for ex.
//  	{"app": "plant-app", "service": "trees", "name": "oak-resource", "action": "trim"}
//  	{"app": "plant-app", "service": "shurbs", "name": "oleander-resource", "action": "fertilize"}
//
//  are two valid resources that plant-app authorizes and manages performs authz
type AuthorizationRequest struct {
	App        string `json:"app,omitempty"`
	Service    string `json:"service,omitempty"`
	Subject    string `json:"subject,omitempty"`
	Resource   string `json:"resource,omitempty"`
	Action     string `json:"action,omitempty"`
	ResourceID string `json:"resource_id,omitempty"`
	Claims     Claims `json:"claims,omitempty"`

	// data is additional facts that are already available
	// to make the authorization decision. for example, current claims that subject has
	Data AuthorizationData `json:"data,omitempty"`
}

// AuthorizationData provides additional context data for auth policy evaluation engine
type AuthorizationData struct {
	RoleBindings []string          `json:"role_bindings,omitempty"`
	Resource     map[string]string `json:"resource,omitempty"`
}

// AuthorizationResult describes policy evaluation result
type AuthorizationResult struct {
	Allowed         bool `json:"allowed,omitempty"`
	ResourceMatched bool `json:"resource_matched,omitempty"`
}

// AuthorizerFn is a function that authorizes each grpc requests.
type AuthorizerFn func(context.Context, AuthorizationRequest) (AuthorizationResult, error)

// IDResolverFn resolves the Identity of the authenticated user which is available as the current user in the context
// by defaut it email is used as the identifier
type IDResolverFn func(Claims) string

// AddtionalClaimsProviderFn provides custom claims object are specified in the token
// use this if certain claims are returned as federated claims
type AddtionalClaimsProviderFn func() interface{}

// RoleBindingResolverFn returns the role bindings for a subject
type RoleBindingResolverFn func(ctx context.Context, subject string) ([]string, error)

// ResourceResolverFn resolves resource returns resource attributes that can be used for authz purpose
type ResourceResolverFn func(ctx context.Context, subject, resource, action, reqId string) (map[string]string, error)

// ResourceIdentifierFn looks at in coming request and picks out the resource id
type ResourceIdentifierFn func(ctx context.Context, req interface{}) (string, error)

// Runtime interface for authN/authZ
type Runtime interface {
	// Verifier authenticates & validates the token and establishes the Subject
	// Token is epected to be present in the context
	Verify(ctx context.Context, token string) (context.Context, Claims, error)
	// Authorizer authorizes resource use
	Authorize(ctx context.Context, claims Claims, resource string, action string, req interface{}) (context.Context, AuthorizationResult, error)
}

type runtime struct {
	logger log.Logger

	appName     string // app name passed as part of the authz request
	serviceName string // service name used as part of the authz request

	issuer                   string                    // oidc token issuer
	aud                      string                    // oidc audience
	caFile                   string                    // ca file
	requiredClaims           map[string]string         // oidc client ID
	signingAlgos             []string                  // JOSE asymmetric signing algorithms
	authorizer               AuthorizerFn              // Authorizes each rpc call
	verifier                 *oidc.IDTokenVerifier     // ID Token Verifier
	idResolver               IDResolverFn              // Current User ID resolver
	additionalClaimsProvider AddtionalClaimsProviderFn // Additional Claims resolver
	roleBindingResolver      RoleBindingResolverFn     // A RoleBinding resolver for a subject
	resourceResolver         ResourceResolverFn        // A Resource resolver for incoming resource
	resourceIdentifier       ResourceIdentifierFn      // Resource identifier resolver for incoming requests
	adminGroup               string                    // a group which needs to mapped to "admin" role in service. this group assignment and resolution happens outside of service
	adminRole                string                    // if the claim has an admin group, map the subject to this role
}

func (f optionFunc) apply(r *runtime) {
	f(r)
}

func emailAsIDResolver(c Claims) string {
	return c.GetEmail()
}

// NewRuntime returns a new Runtime
func NewRuntime(ctx context.Context, options ...Option) (Runtime, error) {
	// setup defaults
	r := &runtime{
		idResolver: emailAsIDResolver,
	}
	for _, opt := range options {
		opt.apply(r)
	}
	if r.logger == nil {
		logger, err := log.NewNop()
		if err != nil {
			return nil, err
		}
		r.logger = logger
	}

	verifier, err := newOIDCVerifier(ctx, r.issuer, r.aud)
	if err != nil {
		return nil, err
	}
	r.verifier = verifier

	r.logger.Infow("auth runtime initialized", "token-issuer", r.issuer, "audience", r.aud)

	return r, nil
}

func (r *runtime) hasExternalAdminGroupMapping(claims Claims) bool {
	for _, g := range claims.GetGroups() {
		if r.adminGroup != "" && r.adminGroup == g {
			return true
		}
	}

	return false
}

func (r *runtime) Authorize(ctx context.Context, claims Claims, resource string, action string, req interface{}) (cx context.Context, ar AuthorizationResult, err error) {

	if r.authorizer == nil {
		// default false
		return ctx, AuthorizationResult{}, nil
	}

	mroles := map[string]bool{}
	if r.hasExternalAdminGroupMapping(claims) {
		// if the presented claims has the external group then map the subject to the admin role
		mroles[r.adminRole] = true
	}

	var incomingResourceID string
	if r.resourceIdentifier != nil {
		rid, err := r.resourceIdentifier(ctx, req)
		if err != nil {
			return ctx, AuthorizationResult{}, err
		}
		incomingResourceID = rid
	}
	subject := CurrentUser(ctx)
	if r.roleBindingResolver != nil {
		boundRoles, err := r.roleBindingResolver(ctx, subject)
		if err != nil {
			return ctx, ar, err
		}
		for _, rb := range boundRoles {
			mroles[rb] = true
		}
	}
	roles := []string{}
	for k := range mroles {
		roles = append(roles, k)
	}

	var resourceInfo map[string]string
	if r.resourceResolver != nil {
		resourceInfo, err = r.resourceResolver(ctx, subject, resource, action, incomingResourceID)
		if err != nil {
			return ctx, ar, err
		}
	}

	authzReq := AuthorizationRequest{
		App:        r.appName,
		Service:    r.serviceName,
		Subject:    subject,
		Resource:   resource,
		ResourceID: incomingResourceID,
		Action:     action,
		Data: AuthorizationData{
			RoleBindings: roles,
			Resource:     resourceInfo,
		},
		Claims: claims,
	}

	ar, err = r.authorizer(ctx, authzReq)

	// TODO copy certain information into context and cache the authz result with a TTL
	return ctx, ar, err
}

func (r *runtime) Verify(ctx context.Context, token string) (context.Context, Claims, error) {

	idt, err := r.verifier.Verify(ctx, token)
	if err != nil {
		return nil, nil, errors.Wrap(err, "id token verification failed")
	}

	cl := &claims{} // parse the standard claims
	if err := idt.Claims(cl); err != nil {
		return nil, nil, errors.Wrap(err, "error resolving claims in identity token")
	}

	if r.additionalClaimsProvider != nil {
		additionalClaims := r.additionalClaimsProvider()
		if err := idt.Claims(additionalClaims); err != nil {
			return nil, nil, errors.Wrap(err, "error resolving additional claim")
		}
		cl.AdditionalClaims = additionalClaims
	}

	return newContext(ctx, r.idResolver(cl)), cl, nil
}

func newOIDCVerifier(ctx context.Context, issuer, audience string) (*oidc.IDTokenVerifier, error) {

	if issuer == "" {
		return nil, errors.New("token issuer url is empty")
	}

	provider, err := oidc.NewProvider(ctx, issuer)
	if err != nil {
		return nil, err
	}

	var cfg oidc.Config
	if audience != "" {
		cfg.ClientID = audience
	} else {
		cfg.SkipClientIDCheck = true
	}

	return provider.Verifier(&cfg), nil
}
