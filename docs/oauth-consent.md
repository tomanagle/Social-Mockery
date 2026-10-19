# Simulated OAuth consent

Social Mockery renders local provider-styled consent screens. They use seeded test identities, show the app and requested permissions, let users select an account and relevant Pages, and return an authorization code or denial to an exactly registered callback. They never collect provider passwords or contact a provider. Layout, fixtures, failure wording, and expiry choices marked below are synthetic, not live captures.

## Try it

```sh
make build
./bin/social-mockery --db /tmp/social-mockery.db --seed profiles/oauth-demo.yaml
```

Open one of these URLs on the local simulator (default port 3000):

- Instagram Login: `http://localhost:3000/https://www.instagram.com/oauth/authorize?client_id=mock-instagram-app&redirect_uri=https%3A%2F%2Fclient.example.test%2Fcallback&response_type=code&scope=instagram_business_basic,instagram_business_content_publish&state=demo`
- Instagram with Facebook Login: `http://localhost:3000/https://www.facebook.com/v25.0/dialog/oauth?client_id=mock-instagram-facebook-app&redirect_uri=https%3A%2F%2Fclient.example.test%2Fcallback&response_type=code&scope=instagram_basic,instagram_content_publish&state=demo`
- Facebook: `http://localhost:3000/https://www.facebook.com/v25.0/dialog/oauth?client_id=mock-facebook-app&redirect_uri=https%3A%2F%2Fclient.example.test%2Fcallback&response_type=code&scope=public_profile,pages_manage_posts&state=demo`
- TikTok: `http://localhost:3000/https://www.tiktok.com/v2/auth/authorize/?client_key=mock-tiktok-app&redirect_uri=https%3A%2F%2Fclient.example.test%2Fcallback&response_type=code&scope=user.info.basic,video.publish&state=demo`

The `.test` callback is deliberately not a working application. Change `redirect_uris` in the fixture to your callback before starting, and send the identical URI in authorization and exchange. TikTok web callbacks require HTTPS without query/fragment. TikTok desktop fixtures use `require_pkce: true` and an HTTP(S) `localhost` or `127.0.0.1` callback. Run behind HTTPS to use secure consent cookies outside local development.

Exchange the returned code with the app secret from the synthetic fixture:

```sh
curl http://localhost:3000/https://api.instagram.com/oauth/access_token \
  -F client_id=mock-instagram-app -F client_secret=mock-instagram-secret \
  -F grant_type=authorization_code -F code=CODE_FROM_CALLBACK \
  -F redirect_uri=https://client.example.test/callback
```

Facebook and Instagram with Facebook Login use GET `/<original HTTPS graph.facebook.com>/v24.0|v25.0|v26.0/oauth/access_token` with `client_id`, `client_secret`, `redirect_uri`, and `code`. TikTok uses form-encoded POST `/<original HTTPS open.tiktokapis.com>/v2/oauth/token/` with `client_key`, `client_secret`, `grant_type=authorization_code`, `redirect_uri`, and `code`. The exact native hosts and generated contract paths are retained; prepend the simulator origin and `/` to the original HTTPS URL.

## Configure test identities without deleting posts

A complete seed profile contains `accounts`, `apps`, `assets`, `grants`, and optional legacy `tokens`. Each app declares:

- `id`, `name`, and a fake `secret`
- `product`: `instagram`, `instagram_facebook`, `facebook`, or `tiktok`
- exact `redirect_uris`, `approved_scopes`, and allowed `account_ids`
- `require_pkce` (TikTok: desktop fixture; other products: optional simulator hardening)
- `graph_api_version` (Meta apps, optional): the app-dashboard version that unversioned Graph calls use, such as `v25.0`

`assets` assign Pages to actors, link professional Instagram accounts, and declare Page `tasks`. `grants` independently record user consent scopes and selected asset IDs. Requested permissions come from the authorize URL. Requesting a permission absent from app approval fails before consent. Consent cannot invent permissions or access another actor's Pages. A successful token exchange records only the chosen grants.

```sh
curl -X POST http://localhost:3000/admin/oauth/fixtures \
  -H 'Authorization: Bearer YOUR_SIMULATOR_ADMIN_TOKEN' \
  --data-binary @profiles/oauth-demo.yaml
```

This upserts accounts/apps/assets/grants while preserving existing posts and tokens; omitted records are not removed. The profile must include referenced accounts/apps/assets so it can be validated independently. Supplied `tokens`, clock and processing values are not imported by this upsert. Use startup seeding or `/admin/reset` for a complete replacement. Replace a grant with empty scopes/assets to simulate revoked consent. Reducing app approval or user grants invalidates affected OAuth-issued Instagram tokens on subsequent requests. Legacy fixture tokens intentionally keep their previous behavior.

Set `--admin-token` when exposing the simulator beyond a private loopback environment. Administrative requests with an Origin header are rejected. Profile secrets are synthetic only; do not put real credentials in fixtures.

## Supported slice and limits

- Browser authorize → consent → callback → single-use code → access token for all four products.
- Instagram Login follows Posty's production behaviour where it conflicts with Meta's Business Login guide. `https://www.instagram.com/oauth/authorize/` (trailing slash) is accepted alongside the documented form, and the code exchange returns a flat `{"access_token", "user_id", "permissions"}` object instead of the guide's `{"data": [...]}` wrapper. Evidence: the repository owner reported on 2026-10-19 that Posty's production Instagram connect works with exactly these requests and a flat-object parser. This is live-verified only for those two points; no sanitized capture is committed, the live type of `permissions` is unverified (the simulator keeps the guide's comma-separated string), and `api.instagram.com` trailing-slash requests remain unsupported.
- Facebook dialog/code exchange versions v24.0, v25.0, v26.0. Instagram Login and TikTok native OAuth paths are unversioned and v2 respectively. Generated contract adapters are separate for every Facebook version; unimplemented reference methods are not mounted.
- Instagram Login issued tokens, short- or long-lived, work with the existing Instagram v24.0/v25.0 resource slice and with `/me` (see [compatibility](compatibility.md)). Instagram Facebook Login requires one selected Page with a linked professional account; its token addresses that account. Existing posts survive reconnecting. This single-Page selection is a simulator limitation.
- Facebook/TikTok resource endpoints remain reference contracts, not working runtime APIs. Facebook Login for Business `config_id`, system-user tokens, app review, live account credentials, QR login, token debugging, permission endpoints, and provider revocation endpoints are not simulated.
- TikTok refresh is persistent and rotates access/refresh tokens atomically. Rotation always invalidates both previous values and retains the original refresh expiry, a synthetic policy within TikTok's documented ability to rotate refresh tokens.
- Instagram Login long-lived tokens: GET `/<original https://graph.instagram.com>/access_token` with `grant_type=ig_exchange_token`, `client_secret` and a valid short-lived `access_token`, and GET `/<original https://graph.instagram.com>/refresh_access_token` with `grant_type=ig_refresh_token` and a long-lived `access_token`. Both return `access_token`, `token_type: bearer` and `expires_in`. Documented: the unversioned paths, the request parameters, the response fields, 60-day validity, refresh only when the token is at least 24 hours old and unexpired, and the `instagram_business_basic` requirement. Synthetic: `expires_in` is exactly 5184000 (Meta's example shows 5183944); repeating an exchange for the same short-lived token returns the same long-lived token; the short-lived token stays valid until its own expiry; refresh keeps the token value and moves its expiry and 24-hour window forward; error messages and codes. Facebook Login long-lived exchange (`fb_exchange_token`) is not implemented.
- Consent lasts 10 simulated minutes. Instagram codes/short tokens last one hour per the Instagram guide; other codes last 5 minutes, Facebook access tokens one hour (synthetic). TikTok access and refresh lifetimes are 24 hours and 365 days. Advancing `/admin/clock/advance` drives expiry. Sessions/codes are bounded and persisted; reset removes them.
- Page task checks cover publishing, moderation, messaging, and insights. This is not a full provider business-role/app-review model. `approved_scopes` is a synthetic way to configure app approval, not a claim that every actor requires Advanced Access.
- Empty/partial user consent is supported to test insufficient-scope failures. Standard Facebook `public_profile`, when requested, is automatically retained. Facebook Login for Business's all-or-nothing permission mode is intentionally unsupported.
- Cookies bind each form to its browser session; hidden session IDs, exact redirect validation, no-store responses, no-referrer, frame denial, escaped templates, and atomic consumption protect the local flow. Multiple tabs have independent session cookies. Replaying Continue or going back after submission returns an expired/used-session error.

## Evidence

Documentation-based URL/parameter/response contracts come from the versioned OAuth reference specs and their provenance. No live account tests were run.

- [Facebook manual login flow](https://developers.facebook.com/docs/facebook-login/guides/advanced/manual-flow/)
- [Instagram business login](https://developers.facebook.com/docs/instagram-platform/instagram-api-with-instagram-login/business-login/)
- [Instagram platform and Page task requirements](https://developers.facebook.com/documentation/instagram-platform/overview)
- [TikTok web login](https://developers.tiktok.com/doc/login-kit-web/)
- [TikTok desktop login](https://developers.tiktok.com/doc/login-kit-desktop/)
- [TikTok token management](https://developers.tiktok.com/doc/oauth-user-access-token-management/)

TikTok desktop S256 uses lowercase hexadecimal SHA-256, corroborated by [the official pinned iOS SDK PKCE implementation](https://github.com/tiktok/tiktok-opensdk-ios/blob/cb0cdc785b653c3be116fd1302ec280fbcfac695/Sources/TikTokOpenAuthSDK/Public/PKCE.swift). This is applied only to TikTok; optional simulator PKCE for Meta uses base64url SHA-256. Web PKCE for TikTok is explicitly unsupported by this simulator rather than inferred from desktop behavior.
