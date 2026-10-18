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
	Value     string   `json:"value" yaml:"value"`
	AccountID string   `json:"account_id" yaml:"account_id"`
	Scopes    []string `json:"scopes" yaml:"scopes"`
	ExpiresAt int64    `json:"expires_at" yaml:"expires_at"`
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
	Version           int       `json:"version" yaml:"version"`
	Clock             string    `json:"clock" yaml:"clock"`
	ProcessingSeconds int64     `json:"processing_seconds" yaml:"processing_seconds"`
	Accounts          []Account `json:"accounts" yaml:"accounts"`
	Tokens            []Token   `json:"tokens" yaml:"tokens"`
}
type Repository interface {
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
}
