package dbstore_test

import (
	"net/http/httptest"
	"testing"

	"github.com/tomanagle/Social-Mockery/internal/handlers"
	"github.com/tomanagle/Social-Mockery/internal/simulation"
	"github.com/tomanagle/Social-Mockery/internal/store"
	"github.com/tomanagle/Social-Mockery/internal/store/db"
	"github.com/tomanagle/Social-Mockery/internal/store/dbstore"
)

func TestSeededOAuthTokenInfersActorForHTTPAuthorization(t *testing.T) {
	for _, product := range []string{"instagram", "instagram_facebook"} {
		t.Run(product, func(t *testing.T) {
			conn, err := db.Open(":memory:")
			if err != nil {
				t.Fatal(err)
			}
			defer conn.Close()
			seed := oauthSeed()
			seed.Apps[0].Product = product
			actor := seed.Accounts[0].ID
			target := actor
			host := "graph.instagram.com"
			scopes := []string{"instagram_business_basic"}
			var assets []string
			if product == "instagram_facebook" {
				target = "42"
				host = "graph.facebook.com"
				scopes = []string{"instagram_basic"}
				assets = []string{"1001"}
				seed.Accounts = append(seed.Accounts, store.Account{ID: target, Username: "linked", Type: "BUSINESS"})
				seed.Assets = []store.OAuthAsset{{ID: "1001", AccountID: actor, InstagramAccountID: target, Tasks: []string{"CREATE_CONTENT"}}}
			}
			seed.Apps[0].ApprovedScopes = scopes
			seed.Grants = []store.OAuthGrant{{AppID: "app", AccountID: actor, Scopes: scopes, AssetIDs: assets}}
			seed.Tokens = append(seed.Tokens, store.Token{Value: "seeded-oauth", AppID: "app", Product: product, AccountID: target, Scopes: scopes, AssetIDs: assets})
			repo := dbstore.New(conn)
			engine := simulation.New(repo, seed, 100)
			if err = engine.Reset(ctx, seed); err != nil {
				t.Fatal(err)
			}
			token, err := repo.Token(ctx, "seeded-oauth")
			if err != nil || token.GrantAccountID != actor {
				t.Fatalf("actor=%s err=%v", token.GrantAccountID, err)
			}
			request := httptest.NewRequest("GET", "/https://"+host+"/v25.0/"+target+"?fields=id", nil)
			request.Header.Set("Authorization", "Bearer seeded-oauth")
			response := httptest.NewRecorder()
			handlers.New(engine, "").ServeHTTP(response, request)
			if response.Code != 200 {
				t.Fatalf("seeded token resource: %d %s", response.Code, response.Body.String())
			}
		})
	}
}
