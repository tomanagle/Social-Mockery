package oauth

import "github.com/tomanagle/Social-Mockery/internal/store"

type consentData struct {
	Session            string
	App                store.OAuthApp
	Accounts           []store.Account
	Assets             []store.OAuthAsset
	Scopes             []string
	Provider, Redirect string
}

const consentHTML = `<!doctype html>
<html lang="en"><head><meta charset="utf-8"><meta name="viewport" content="width=device-width,initial-scale=1"><title>{{.Provider}} simulated consent</title>
<style>
*{box-sizing:border-box}body{margin:0;background:#f3f5f8;color:#19202b;font:16px/1.5 system-ui,sans-serif}.banner{padding:12px;text-align:center;background:#fff0c3;color:#684600;font-size:14px;font-weight:650}.card{max-width:590px;margin:36px auto;background:white;border:1px solid #dee3eb;border-radius:18px;padding:32px;box-shadow:0 8px 32px #1020300a}.brand{font-weight:750;font-size:26px;color:#155cce}.brand[data-provider=Instagram]{color:#ba2573}.brand[data-provider=TikTok]{color:#111}h1{font-size:24px;line-height:1.3;margin:20px 0 10px}p{color:#566174}.subtle{font-size:13px;overflow-wrap:anywhere}fieldset{border:1px solid #e0e4eb;border-radius:10px;margin:22px 0;padding:14px}legend{font-weight:650;padding:0 6px}label{display:block;padding:8px 0;overflow-wrap:anywhere}input{accent-color:#155cce;margin-right:8px}select{padding:12px;width:100%;font:inherit;border:1px solid #bbc4d3;border-radius:8px}button{font:600 16px system-ui;border:0;border-radius:8px;padding:13px 18px;cursor:pointer;background:#155cce;color:white}.actions{display:flex;gap:12px;margin-top:24px}.deny{background:#e9edf3;color:#263548}.notice{font-size:14px;background:#f4f6f9;padding:12px;border-radius:8px}@media(max-width:620px){.card{margin:16px 10px;padding:22px}}
</style></head><body><div class="banner">SOCIAL MOCKERY · SIMULATION ONLY · No real provider login</div><main class="card"><div class="brand" data-provider="{{.Provider}}">{{.Provider}}</div><h1>Continue to {{.App.Name}}?</h1><p>This app is requesting access to the test account and permissions you choose below.</p><p class="notice">These are seeded test identities. Never enter a real password, token, or account secret. No request is sent to {{.Provider}}.</p>
<form method="post" action="/oauth/consent"><input type="hidden" name="session" value="{{.Session}}"><fieldset><legend>Choose a test account</legend><label for="account">Account</label><select id="account" name="account" required>{{range .Accounts}}<option value="{{.ID}}">{{.Username}} · {{.Type}} · {{.ID}}</option>{{end}}</select></fieldset>
<fieldset><legend>Requested permissions</legend>{{range .Scopes}}<label><input type="checkbox" name="scope" value="{{.}}" checked>{{permissionDescription .}}<br><span class="subtle">{{.}}</span></label>{{end}}<p class="subtle">Only checked permissions are granted. App approval and your consent are separate. Declining a permission may prevent a feature from working.</p></fieldset>
{{if .Assets}}<fieldset><legend>Choose Pages for the selected account</legend>{{range .Assets}}<label><input type="checkbox" name="asset" value="{{.ID}}">{{.Name}} · Page {{.ID}}<br><span class="subtle">Account {{.AccountID}}{{if .InstagramAccountID}} · Linked Instagram {{.InstagramAccountID}}{{end}}</span></label>{{end}}<p class="subtle">Select only Pages belonging to the account above. Instagram with Facebook Login requires one linked professional account.</p></fieldset>{{end}}
<p class="subtle">App ID: {{.App.ID}}<br>After your choice, return to: {{.Redirect}}</p><div class="actions"><button type="submit" name="decision" value="approve">Continue</button><button class="deny" type="submit" name="decision" value="deny">Cancel</button></div></form><p class="subtle">Simulated consent layout inspired by provider flows; not an exact reproduction. This screen expires after 10 minutes of simulator time.</p></main></body></html>`

// Descriptions are plain-language summaries, not provider UI quotations.
func permissionDescription(scope string) string {
	switch scope {
	case "public_profile", "user.info.basic":
		return "Read your basic profile information"
	case "email":
		return "Read your email address"
	case "instagram_basic", "instagram_business_basic":
		return "Read your Instagram profile and media"
	case "instagram_content_publish", "instagram_business_content_publish":
		return "Publish content to your Instagram account"
	case "instagram_manage_comments", "instagram_business_manage_comments":
		return "Manage comments on your Instagram content"
	case "instagram_manage_messages", "instagram_business_manage_messages":
		return "Manage Instagram messages"
	case "instagram_manage_insights", "instagram_business_manage_insights":
		return "Read Instagram insights"
	case "pages_show_list":
		return "List the Pages you manage"
	case "pages_read_engagement":
		return "Read Page content and engagement"
	case "pages_read_user_content":
		return "Read user content on your Pages"
	case "pages_manage_posts":
		return "Create and manage Page posts"
	case "pages_manage_engagement":
		return "Manage engagement on your Pages"
	case "pages_manage_metadata":
		return "Manage Page settings and metadata"
	case "read_insights":
		return "Read Page insights"
	case "user.info.profile":
		return "Read additional TikTok profile information"
	case "user.info.stats":
		return "Read TikTok profile statistics"
	case "video.list":
		return "Read your public TikTok videos"
	case "video.upload":
		return "Upload draft videos to TikTok"
	case "video.publish":
		return "Publish videos to TikTok"
	default:
		return scope
	}
}
