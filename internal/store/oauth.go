package store

type OAuthApp struct {
	ID             string   `json:"id" yaml:"id"`
	Name           string   `json:"name" yaml:"name"`
	Secret         string   `json:"secret" yaml:"secret"`
	Product        string   `json:"product" yaml:"product"`
	RedirectURIs   []string `json:"redirect_uris" yaml:"redirect_uris"`
	ApprovedScopes []string `json:"approved_scopes" yaml:"approved_scopes"`
	AccountIDs     []string `json:"account_ids" yaml:"account_ids"`
	RequirePKCE    bool     `json:"require_pkce" yaml:"require_pkce"`
	// GraphAPIVersion is the app dashboard version used by unversioned Graph calls.
	GraphAPIVersion string `json:"graph_api_version,omitempty" yaml:"graph_api_version,omitempty"`
}
type OAuthGrant struct {
	AppID     string   `json:"app_id" yaml:"app_id"`
	AccountID string   `json:"account_id" yaml:"account_id"`
	Scopes    []string `json:"scopes" yaml:"scopes"`
	AssetIDs  []string `json:"asset_ids,omitempty" yaml:"asset_ids,omitempty"`
}
type OAuthAsset struct {
	ID                 string   `json:"id" yaml:"id"`
	Name               string   `json:"name" yaml:"name"`
	AccountID          string   `json:"account_id" yaml:"account_id"`
	InstagramAccountID string   `json:"instagram_account_id,omitempty" yaml:"instagram_account_id,omitempty"`
	Tasks              []string `json:"tasks" yaml:"tasks"`
}
type OAuthSession struct {
	ID          string   `json:"id"`
	AppID       string   `json:"app_id"`
	RedirectURI string   `json:"redirect_uri"`
	State       string   `json:"state"`
	Scopes      []string `json:"scopes"`
	Challenge   string   `json:"challenge"`
	ExpiresAt   int64    `json:"expires_at"`
}
type OAuthCode struct {
	Value       string   `json:"value"`
	AppID       string   `json:"app_id"`
	AccountID   string   `json:"account_id"`
	RedirectURI string   `json:"redirect_uri"`
	Scopes      []string `json:"scopes"`
	AssetIDs    []string `json:"asset_ids"`
	Challenge   string   `json:"challenge"`
	ExpiresAt   int64    `json:"expires_at"`
}
