package auth

// Claims represents a standard profile info returned as result of an OpenID Authentication Event. See https://openid.net/specs/openid-connect-core-1_0.html#StandardClaims
type Claims interface {
	GetSubject() string
	GetName() string
	GetGivenName() string
	GetFamilyName() string
	GetMiddleName() string
	GetNickName() string
	GetPreferredUserName() string
	GetProfileURL() string
	GetPictureURL() string
	GetEmail() string
	IsEmailVerified() bool
	GetLocale() string
	GetGroups() []string

	GetAdditionalClaims() interface{}
}

type claims struct {
	Subject           string   `json:"sub,omitempty"`
	Name              string   `json:"name,omitempty"`
	GivenName         string   `json:"given_name,omitempty"`
	FamilyName        string   `json:"family_name,omitempty"`
	MiddleName        string   `json:"middle_name,omitempty"`
	NickName          string   `json:"nickname,omitempty"`
	PreferredUserName string   `json:"preferred_username,omitempty"`
	ProfileURL        string   `json:"profile,omitempty"`
	PictureURL        string   `json:"picture,omitempty"`
	Email             string   `json:"email,omitempty"`
	EmailVerified     bool     `json:"email_verified,omitempty"`
	Locale            string   `json:"locale,omitempty"`
	Groups            []string `json:"groups,omitempty"`

	AdditionalClaims interface{} `json:"additional_claims,omitempty"` // these are custom claims that are presented in the token.
}

// GetSubject returns the sub field of this token
func (c *claims) GetSubject() string {

	return c.Subject
}

func (c *claims) GetName() string {

	return c.Name
}

func (c *claims) GetGivenName() string {

	return c.GivenName
}

func (c *claims) GetFamilyName() string {

	return c.FamilyName
}

func (c *claims) GetMiddleName() string {

	return c.MiddleName
}

func (c *claims) GetNickName() string {

	return c.NickName
}

func (c *claims) GetPreferredUserName() string {

	return c.PreferredUserName
}

func (c *claims) GetProfileURL() string {

	return c.ProfileURL
}

func (c *claims) GetPictureURL() string {

	return c.PictureURL
}

func (c *claims) GetEmail() string {

	return c.Email
}

func (c *claims) IsEmailVerified() bool {

	return c.EmailVerified
}

func (c *claims) GetLocale() string {

	return c.Locale
}

func (c *claims) GetGroups() []string {
	if c.Groups == nil {
		return []string{}
	}

	return c.Groups
}

// GetConnectorUserID returns the connector-local unique identifier. This can
// be useful for logging a more friendly field
func (c *claims) GetAdditionalClaims() interface{} {
	return c.AdditionalClaims
}
