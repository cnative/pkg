package auth

import (
	"context"

	"github.com/pkg/errors"

	"github.com/coreos/go-oidc"

	"github.com/cnative/pkg/log"
)

// Resource is an object upon which an authorization check needs to be performed
// for ex.
//  	{"app": "plant-app", "service": "trees", "name": "oak-resource"}
//  	{"app": "plant-app", "service": "shurbs", "name": "oleander-resource"}
//
//  are two valid resources that plant-app authorizes and manages performs authz
type Resource struct {
	App     string `json:"app,omitempty"`
	Service string `json:"service,omitempty"`
	Name    string `json:"name,omitempty"`
}

// Action is an operation that can be performed on a Resource
// for ex.
// 		water, plant, trim and relocate are
//      the operation that are allowed on oak-resource resource
type Action string

// AuthorizerFn is a function that authorizes each grpc requests.
type AuthorizerFn func(context.Context, Claims, Resource, Action) bool

// IDResolverFn resolves the Identity of the authenticated user which is available as the current user in the context
// by defaut it email is used as the identifier
type IDResolverFn func(Claims) string

// AddtionalClaimsProviderFn provides custom claims object are specified in the token
// use this if certain claims are returned as federated claims
type AddtionalClaimsProviderFn func() interface{}

// Runtime interface for authN/authZ
type Runtime interface {
	// Verifier authenticates & validates the token and establishes the Subject
	// Token is epected to be present in the context
	Verify(ctx context.Context, token string) (context.Context, Claims, error)
	// Authorizer authorizes resource use
	Authorize(context.Context, Claims, Resource, Action) (context.Context, bool, error)
}

type runtime struct {
	logger *log.Logger

	issuer                   string                    // oidc token issuer
	aud                      string                    // oidc audience
	caFile                   string                    // ca file
	requiredClaims           map[string]string         // oidc client ID
	signingAlgos             []string                  // JOSE asymmetric signing algorithms
	authorizer               AuthorizerFn              // Authorizes each rpc call
	verifier                 *oidc.IDTokenVerifier     // ID Token Verifier
	idResolver               IDResolverFn              // Current User ID resolver
	additionalClaimsProvider AddtionalClaimsProviderFn // Additional Claims resolver
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

func (r *runtime) Authorize(ctx context.Context, claims Claims, resource Resource, action Action) (context.Context, bool, error) {
	if r.authorizer != nil {
		return ctx, r.authorizer(ctx, claims, resource, action), nil
	}

	return ctx, false, nil
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
