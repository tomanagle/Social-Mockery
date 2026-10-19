package store

import (
	"context"
	"errors"
)

var ErrNotFound = errors.New("not found")
var ErrConflict = errors.New("conflict")
var ErrCapacity = errors.New("state capacity reached; reset or increase max-records")

type Account struct {
	ID       string `json:"id" yaml:"id"`
	Username string `json:"username" yaml:"username"`
	Type     string `json:"type" yaml:"type"`
}
type Token struct {
	RefreshToken     string   `json:"refresh_token,omitempty" yaml:"refresh_token,omitempty"`
	RefreshExpiresAt int64    `json:"refresh_expires_at,omitempty" yaml:"refresh_expires_at,omitempty"`
	GrantAccountID   string   `json:"grant_account_id,omitempty" yaml:"grant_account_id,omitempty"`
	AppID            string   `json:"app_id,omitempty" yaml:"app_id,omitempty"`
	Product          string   `json:"product,omitempty" yaml:"product,omitempty"`
	AssetIDs         []string `json:"asset_ids,omitempty" yaml:"asset_ids,omitempty"`
	Value            string   `json:"value" yaml:"value"`
	AccountID        string   `json:"account_id" yaml:"account_id"`
	Scopes           []string `json:"scopes" yaml:"scopes"`
	ExpiresAt        int64    `json:"expires_at" yaml:"expires_at"`
	// Kind distinguishes Instagram Login long-lived tokens ("long_lived") from
	// short-lived or legacy fixture tokens (""). IssuedAt is virtual Unix time.
	Kind     string `json:"kind,omitempty" yaml:"kind,omitempty"`
	IssuedAt int64  `json:"issued_at,omitempty" yaml:"issued_at,omitempty"`
}
type Container struct {
	ID          string `json:"id"`
	AccountID   string `json:"account_id"`
	MediaType   string `json:"media_type"`
	URL         string `json:"media_url"`
	Caption     string `json:"caption"`
	CreatedAt   int64  `json:"created_at"`
	ReadyAt     int64  `json:"ready_at"`
	PublishedID string `json:"published_id,omitempty"`
}
type Post struct {
	ID        string `json:"id"`
	AccountID string `json:"account_id"`
	MediaType string `json:"media_type"`
	URL       string `json:"media_url"`
	Caption   string `json:"caption"`
	CreatedAt int64  `json:"created_at"`
}
type Seed struct {
	Apps              []OAuthApp   `json:"apps,omitempty" yaml:"apps,omitempty"`
	Grants            []OAuthGrant `json:"grants,omitempty" yaml:"grants,omitempty"`
	Assets            []OAuthAsset `json:"assets,omitempty" yaml:"assets,omitempty"`
	Version           int          `json:"version" yaml:"version"`
	Clock             string       `json:"clock" yaml:"clock"`
	ProcessingSeconds int64        `json:"processing_seconds" yaml:"processing_seconds"`
	Accounts          []Account    `json:"accounts" yaml:"accounts"`
	Tokens            []Token      `json:"tokens" yaml:"tokens"`
}
type Repository interface {
	OAuthRefreshToken(context.Context, string) (Token, error)
	RefreshOAuthToken(context.Context, string, Token) error
	ExchangeInstagramToken(context.Context, string, Token) error
	ExtendInstagramToken(context.Context, string, int64, int64) error
	OAuthApp(context.Context, string) (OAuthApp, error)
	OAuthAccounts(context.Context, string) ([]Account, error)
	OAuthAssets(context.Context, string) ([]OAuthAsset, error)
	OAuthGrant(context.Context, string, string) (OAuthGrant, error)
	PutOAuthSession(context.Context, OAuthSession) error
	CompleteOAuthSession(context.Context, string, OAuthCode) error
	GetOAuthSession(context.Context, string) (OAuthSession, error)
	DeleteOAuthSession(context.Context, string) error
	PutOAuthCode(context.Context, OAuthCode) error
	GetOAuthCode(context.Context, string) (OAuthCode, error)
	RedeemOAuthCode(context.Context, string, Token, OAuthGrant) error
	UpsertOAuthFixtures(context.Context, Seed) error

	Reset(context.Context, Seed, int64) error
	Clock(context.Context) (int64, int64, error)
	Advance(context.Context, int64) (int64, error)
	Account(context.Context, string) (Account, error)
	Token(context.Context, string) (Token, error)
	CreateContainer(context.Context, Container, int) (Container, error)
	Container(context.Context, string) (Container, error)
	Publish(context.Context, string, string, int) (Post, error)
	Post(context.Context, string) (Post, error)
	Posts(context.Context, string, int, int) ([]Post, error)
	PostCount(context.Context, string) (int, error)
}
