# Instagram API contract inventory

Research date: 2026-10-18. All entries are documentation-based, not live-verified.

**This is a broad, incomplete research contract, not a claim of full Instagram compatibility.** OAuth, upload transport and release-specific validation remain blocked (see below). The simulator still implements only its original four operations in v24.0/v25.0. None of these contract-only documents is registered by the application.

## Scope and versions

Facebook Login and Instagram Login use separate documents and generated packages. Facebook Login has a linked-Page model; Instagram Login has its own permissions and messaging endpoints. SDK-adjacent advertising, commerce and Threads-related edges are inventoried because they appear on Instagram objects, but are not implied to work under Instagram Login.

- v24.0: Business SDK v24.0.1, pinned revision f1722382526f18b5ab33de007641c4b89f5d86fc.
- v25.0: Business SDK v25.0.3, pinned revision 542e10e31c40f7925f9ac03c000922a2fd0a2365.
- v26.0: Business SDK v26.0.2, pinned revision efd8423a2e595ea8d4c04eb824ce113f2f1d68cd; research only, no runtime router.

The official release table was inaccessible/HTTP 429. These are SDK-evidenced version candidates, not verification that v26.0 is the latest Graph release or that all three remain available. The Meta Postman collection is version-neutral and contains older v22/v23 examples. Repeating that collection baseline across documents explicitly does not verify it independently against each release.

## Endpoint inventory

Each row is one wire method/path. Generic Graph object paths merge account, media, container, comment, conversation, message and scoped-person reads where appropriate; use fields and the documented node type. The machine-readable sdk-inventory.json retains the individual SDK methods, parameter types, source revision and source line.

### v24.0

| Product | Method | Path | Evidence / shape |
|---|---|---|---|
| facebook-login | GET | `/v24.0/{objectID}` | SDK fields; mutation/error details partial |
| facebook-login | DELETE | `/v24.0/{objectID}` | SDK fields; mutation/error details partial |
| facebook-login | POST | `/v24.0/{objectID}` | SDK fields; mutation/error details partial |
| facebook-login | GET | `/v24.0/{objectID}/agencies` | SDK fields; mutation/error details partial |
| facebook-login | GET | `/v24.0/{objectID}/authorized_adaccounts` | SDK fields; mutation/error details partial |
| facebook-login | POST | `/v24.0/{objectID}/authorized_adaccounts` | SDK fields; mutation/error details partial |
| facebook-login | GET | `/v24.0/{objectID}/available_catalogs` | SDK fields; mutation/error details partial |
| facebook-login | GET | `/v24.0/{objectID}/branded_content_ad_permissions` | SDK fields; mutation/error details partial |
| facebook-login | POST | `/v24.0/{objectID}/branded_content_ad_permissions` | SDK fields; mutation/error details partial |
| facebook-login | GET | `/v24.0/{objectID}/branded_content_advertisable_medias` | SDK fields; mutation/error details partial |
| facebook-login | DELETE | `/v24.0/{objectID}/branded_content_tag_approval` | SDK fields; mutation/error details partial |
| facebook-login | GET | `/v24.0/{objectID}/branded_content_tag_approval` | SDK fields; mutation/error details partial |
| facebook-login | POST | `/v24.0/{objectID}/branded_content_tag_approval` | SDK fields; mutation/error details partial |
| facebook-login | GET | `/v24.0/{objectID}/catalog_product_search` | SDK fields; mutation/error details partial |
| facebook-login | GET | `/v24.0/{objectID}/collaboration_invites` | SDK fields; mutation/error details partial |
| facebook-login | POST | `/v24.0/{objectID}/collaboration_invites` | SDK fields; mutation/error details partial |
| facebook-login | GET | `/v24.0/{objectID}/connected_threads_user` | SDK fields; mutation/error details partial |
| facebook-login | GET | `/v24.0/{objectID}/content_publishing_limit` | SDK fields; mutation/error details partial |
| facebook-login | GET | `/v24.0/{objectID}/creator_marketplace_creators` | SDK fields; mutation/error details partial |
| facebook-login | GET | `/v24.0/{objectID}/dataset` | SDK fields; mutation/error details partial |
| facebook-login | POST | `/v24.0/{objectID}/dataset` | SDK fields; mutation/error details partial |
| facebook-login | GET | `/v24.0/{objectID}/insights` | SDK fields; mutation/error details partial |
| facebook-login | GET | `/v24.0/{objectID}/instagram_backed_threads_user` | SDK fields; mutation/error details partial |
| facebook-login | POST | `/v24.0/{objectID}/instagram_backed_threads_user` | SDK fields; mutation/error details partial |
| facebook-login | GET | `/v24.0/{objectID}/live_media` | SDK fields; mutation/error details partial |
| facebook-login | GET | `/v24.0/{objectID}/media` | SDK fields; mutation/error details partial |
| facebook-login | POST | `/v24.0/{objectID}/media` | SDK fields; mutation/error details partial |
| facebook-login | POST | `/v24.0/{objectID}/media_publish` | SDK fields; mutation/error details partial |
| facebook-login | POST | `/v24.0/{objectID}/mentions` | SDK fields; mutation/error details partial |
| facebook-login | GET | `/v24.0/{objectID}/notification_message_tokens` | SDK fields; mutation/error details partial |
| facebook-login | GET | `/v24.0/{objectID}/product_appeal` | SDK fields; mutation/error details partial |
| facebook-login | POST | `/v24.0/{objectID}/product_appeal` | SDK fields; mutation/error details partial |
| facebook-login | GET | `/v24.0/{objectID}/recently_searched_hashtags` | SDK fields; mutation/error details partial |
| facebook-login | GET | `/v24.0/{objectID}/scheduled_media` | SDK fields; mutation/error details partial |
| facebook-login | GET | `/v24.0/{objectID}/stories` | SDK fields; mutation/error details partial |
| facebook-login | GET | `/v24.0/{objectID}/tags` | SDK fields; mutation/error details partial |
| facebook-login | GET | `/v24.0/{objectID}/upcoming_events` | SDK fields; mutation/error details partial |
| facebook-login | POST | `/v24.0/{objectID}/upcoming_events` | SDK fields; mutation/error details partial |
| facebook-login | GET | `/v24.0/{objectID}/welcome_message_flows` | SDK fields; mutation/error details partial |
| facebook-login | GET | `/v24.0/{objectID}/boost_ads_list` | SDK fields; mutation/error details partial |
| facebook-login | GET | `/v24.0/{objectID}/branded_content_partner_promote` | SDK fields; mutation/error details partial |
| facebook-login | POST | `/v24.0/{objectID}/branded_content_partner_promote` | SDK fields; mutation/error details partial |
| facebook-login | GET | `/v24.0/{objectID}/children` | SDK fields; mutation/error details partial |
| facebook-login | GET | `/v24.0/{objectID}/collaborators` | SDK fields; mutation/error details partial |
| facebook-login | GET | `/v24.0/{objectID}/comments` | SDK fields; mutation/error details partial |
| facebook-login | POST | `/v24.0/{objectID}/comments` | SDK fields; mutation/error details partial |
| facebook-login | DELETE | `/v24.0/{objectID}/partnership_ad_code` | SDK fields; mutation/error details partial |
| facebook-login | POST | `/v24.0/{objectID}/partnership_ad_code` | SDK fields; mutation/error details partial |
| facebook-login | GET | `/v24.0/{objectID}/product_tags` | SDK fields; mutation/error details partial |
| facebook-login | POST | `/v24.0/{objectID}/product_tags` | SDK fields; mutation/error details partial |
| facebook-login | GET | `/v24.0/{objectID}/replies` | SDK fields; mutation/error details partial |
| facebook-login | POST | `/v24.0/{objectID}/replies` | SDK fields; mutation/error details partial |
| facebook-login | GET | `/v24.0/{objectID}/recent_media` | SDK fields; mutation/error details partial |
| facebook-login | GET | `/v24.0/{objectID}/top_media` | SDK fields; mutation/error details partial |
| facebook-login | GET | `/v24.0/me/accounts` | SDK fields; mutation/error details partial |
| facebook-login | GET | `/v24.0/{objectID}/conversations` | SDK fields; mutation/error details partial |
| facebook-login | GET | `/v24.0/{objectID}/subscribed_apps` | SDK fields; mutation/error details partial |
| facebook-login | POST | `/v24.0/{objectID}/subscribed_apps` | SDK fields; mutation/error details partial |
| facebook-login | DELETE | `/v24.0/{objectID}/subscribed_apps` | SDK fields; mutation/error details partial |
| instagram-login | POST | `/v24.0/{objectID}/messages` | Collection examples; version-neutral, partial |
| instagram-login | POST | `/v24.0/{objectID}/message_attachments` | Collection examples; version-neutral, partial |
| instagram-login | GET | `/v24.0/{objectID}/conversations` | Collection examples; version-neutral, partial |
| instagram-login | GET | `/v24.0/{objectID}` | Collection examples; version-neutral, partial |
| instagram-login | GET | `/v24.0/{objectID}/messenger_profile` | Collection examples; version-neutral, partial |
| instagram-login | POST | `/v24.0/{objectID}/messenger_profile` | Collection examples; version-neutral, partial |
| instagram-login | DELETE | `/v24.0/{objectID}/messenger_profile` | Collection examples; version-neutral, partial |
| instagram-login | GET | `/v24.0/{objectID}/welcome_message_flows` | Collection examples; version-neutral, partial |
| instagram-login | POST | `/v24.0/{objectID}/welcome_message_flows` | Collection examples; version-neutral, partial |
| instagram-login | DELETE | `/v24.0/{objectID}/welcome_message_flows` | Collection examples; version-neutral, partial |
| instagram-login | GET | `/v24.0/{objectID}/subscribed_apps` | Collection examples; version-neutral, partial |
| instagram-login | POST | `/v24.0/{objectID}/subscribed_apps` | Collection examples; version-neutral, partial |
| instagram-login | DELETE | `/v24.0/{objectID}/subscribed_apps` | Collection examples; version-neutral, partial |
| instagram-login | GET | `/v24.0/me/conversations` | Collection examples; version-neutral, partial |
| instagram-login | POST | `/v24.0/{objectID}/media` | Collection examples; version-neutral, partial |
| instagram-login | POST | `/v24.0/{objectID}/media_publish` | Collection examples; version-neutral, partial |
| instagram-login | GET | `/v24.0/{objectID}/insights` | Collection examples; version-neutral, partial |
| instagram-login | GET | `/v24.0/{objectID}/comments` | Collection examples; version-neutral, partial |
| instagram-login | POST | `/v24.0/{objectID}/replies` | Collection examples; version-neutral, partial |
| webhooks | POST | `/webhooks/instagram` | Collection examples; version-neutral, partial |
### v25.0

| Product | Method | Path | Evidence / shape |
|---|---|---|---|
| facebook-login | GET | `/v25.0/{objectID}` | SDK fields; mutation/error details partial |
| facebook-login | DELETE | `/v25.0/{objectID}` | SDK fields; mutation/error details partial |
| facebook-login | POST | `/v25.0/{objectID}` | SDK fields; mutation/error details partial |
| facebook-login | GET | `/v25.0/{objectID}/agencies` | SDK fields; mutation/error details partial |
| facebook-login | GET | `/v25.0/{objectID}/authorized_adaccounts` | SDK fields; mutation/error details partial |
| facebook-login | POST | `/v25.0/{objectID}/authorized_adaccounts` | SDK fields; mutation/error details partial |
| facebook-login | GET | `/v25.0/{objectID}/available_catalogs` | SDK fields; mutation/error details partial |
| facebook-login | GET | `/v25.0/{objectID}/branded_content_ad_permissions` | SDK fields; mutation/error details partial |
| facebook-login | POST | `/v25.0/{objectID}/branded_content_ad_permissions` | SDK fields; mutation/error details partial |
| facebook-login | GET | `/v25.0/{objectID}/branded_content_advertisable_medias` | SDK fields; mutation/error details partial |
| facebook-login | DELETE | `/v25.0/{objectID}/branded_content_tag_approval` | SDK fields; mutation/error details partial |
| facebook-login | GET | `/v25.0/{objectID}/branded_content_tag_approval` | SDK fields; mutation/error details partial |
| facebook-login | POST | `/v25.0/{objectID}/branded_content_tag_approval` | SDK fields; mutation/error details partial |
| facebook-login | POST | `/v25.0/{objectID}/business_messaging_feature_status` | SDK fields; mutation/error details partial |
| facebook-login | GET | `/v25.0/{objectID}/catalog_product_search` | SDK fields; mutation/error details partial |
| facebook-login | GET | `/v25.0/{objectID}/collaboration_invites` | SDK fields; mutation/error details partial |
| facebook-login | POST | `/v25.0/{objectID}/collaboration_invites` | SDK fields; mutation/error details partial |
| facebook-login | GET | `/v25.0/{objectID}/collaborative_media` | SDK fields; mutation/error details partial |
| facebook-login | GET | `/v25.0/{objectID}/connected_threads_user` | SDK fields; mutation/error details partial |
| facebook-login | GET | `/v25.0/{objectID}/content_publishing_limit` | SDK fields; mutation/error details partial |
| facebook-login | GET | `/v25.0/{objectID}/creator_marketplace_creators` | SDK fields; mutation/error details partial |
| facebook-login | GET | `/v25.0/{objectID}/dataset` | SDK fields; mutation/error details partial |
| facebook-login | POST | `/v25.0/{objectID}/dataset` | SDK fields; mutation/error details partial |
| facebook-login | GET | `/v25.0/{objectID}/insights` | SDK fields; mutation/error details partial |
| facebook-login | GET | `/v25.0/{objectID}/instagram_backed_threads_user` | SDK fields; mutation/error details partial |
| facebook-login | POST | `/v25.0/{objectID}/instagram_backed_threads_user` | SDK fields; mutation/error details partial |
| facebook-login | GET | `/v25.0/{objectID}/live_media` | SDK fields; mutation/error details partial |
| facebook-login | GET | `/v25.0/{objectID}/media` | SDK fields; mutation/error details partial |
| facebook-login | POST | `/v25.0/{objectID}/media` | SDK fields; mutation/error details partial |
| facebook-login | POST | `/v25.0/{objectID}/media_publish` | SDK fields; mutation/error details partial |
| facebook-login | POST | `/v25.0/{objectID}/mentions` | SDK fields; mutation/error details partial |
| facebook-login | POST | `/v25.0/{objectID}/moderate_conversations` | SDK fields; mutation/error details partial |
| facebook-login | GET | `/v25.0/{objectID}/notification_message_tokens` | SDK fields; mutation/error details partial |
| facebook-login | GET | `/v25.0/{objectID}/product_appeal` | SDK fields; mutation/error details partial |
| facebook-login | POST | `/v25.0/{objectID}/product_appeal` | SDK fields; mutation/error details partial |
| facebook-login | GET | `/v25.0/{objectID}/recently_searched_hashtags` | SDK fields; mutation/error details partial |
| facebook-login | GET | `/v25.0/{objectID}/scheduled_media` | SDK fields; mutation/error details partial |
| facebook-login | GET | `/v25.0/{objectID}/stories` | SDK fields; mutation/error details partial |
| facebook-login | GET | `/v25.0/{objectID}/tags` | SDK fields; mutation/error details partial |
| facebook-login | GET | `/v25.0/{objectID}/upcoming_events` | SDK fields; mutation/error details partial |
| facebook-login | POST | `/v25.0/{objectID}/upcoming_events` | SDK fields; mutation/error details partial |
| facebook-login | GET | `/v25.0/{objectID}/welcome_message_flows` | SDK fields; mutation/error details partial |
| facebook-login | GET | `/v25.0/{objectID}/boost_ads_list` | SDK fields; mutation/error details partial |
| facebook-login | GET | `/v25.0/{objectID}/branded_content_partner_promote` | SDK fields; mutation/error details partial |
| facebook-login | POST | `/v25.0/{objectID}/branded_content_partner_promote` | SDK fields; mutation/error details partial |
| facebook-login | GET | `/v25.0/{objectID}/children` | SDK fields; mutation/error details partial |
| facebook-login | GET | `/v25.0/{objectID}/collaborators` | SDK fields; mutation/error details partial |
| facebook-login | GET | `/v25.0/{objectID}/comments` | SDK fields; mutation/error details partial |
| facebook-login | POST | `/v25.0/{objectID}/comments` | SDK fields; mutation/error details partial |
| facebook-login | DELETE | `/v25.0/{objectID}/partnership_ad_code` | SDK fields; mutation/error details partial |
| facebook-login | POST | `/v25.0/{objectID}/partnership_ad_code` | SDK fields; mutation/error details partial |
| facebook-login | GET | `/v25.0/{objectID}/product_tags` | SDK fields; mutation/error details partial |
| facebook-login | POST | `/v25.0/{objectID}/product_tags` | SDK fields; mutation/error details partial |
| facebook-login | GET | `/v25.0/{objectID}/replies` | SDK fields; mutation/error details partial |
| facebook-login | POST | `/v25.0/{objectID}/replies` | SDK fields; mutation/error details partial |
| facebook-login | GET | `/v25.0/{objectID}/recent_media` | SDK fields; mutation/error details partial |
| facebook-login | GET | `/v25.0/{objectID}/top_media` | SDK fields; mutation/error details partial |
| facebook-login | GET | `/v25.0/me/accounts` | SDK fields; mutation/error details partial |
| facebook-login | GET | `/v25.0/{objectID}/conversations` | SDK fields; mutation/error details partial |
| facebook-login | GET | `/v25.0/{objectID}/subscribed_apps` | SDK fields; mutation/error details partial |
| facebook-login | POST | `/v25.0/{objectID}/subscribed_apps` | SDK fields; mutation/error details partial |
| facebook-login | DELETE | `/v25.0/{objectID}/subscribed_apps` | SDK fields; mutation/error details partial |
| instagram-login | POST | `/v25.0/{objectID}/messages` | Collection examples; version-neutral, partial |
| instagram-login | POST | `/v25.0/{objectID}/message_attachments` | Collection examples; version-neutral, partial |
| instagram-login | GET | `/v25.0/{objectID}/conversations` | Collection examples; version-neutral, partial |
| instagram-login | GET | `/v25.0/{objectID}` | Collection examples; version-neutral, partial |
| instagram-login | GET | `/v25.0/{objectID}/messenger_profile` | Collection examples; version-neutral, partial |
| instagram-login | POST | `/v25.0/{objectID}/messenger_profile` | Collection examples; version-neutral, partial |
| instagram-login | DELETE | `/v25.0/{objectID}/messenger_profile` | Collection examples; version-neutral, partial |
| instagram-login | GET | `/v25.0/{objectID}/welcome_message_flows` | Collection examples; version-neutral, partial |
| instagram-login | POST | `/v25.0/{objectID}/welcome_message_flows` | Collection examples; version-neutral, partial |
| instagram-login | DELETE | `/v25.0/{objectID}/welcome_message_flows` | Collection examples; version-neutral, partial |
| instagram-login | GET | `/v25.0/{objectID}/subscribed_apps` | Collection examples; version-neutral, partial |
| instagram-login | POST | `/v25.0/{objectID}/subscribed_apps` | Collection examples; version-neutral, partial |
| instagram-login | DELETE | `/v25.0/{objectID}/subscribed_apps` | Collection examples; version-neutral, partial |
| instagram-login | GET | `/v25.0/me/conversations` | Collection examples; version-neutral, partial |
| instagram-login | POST | `/v25.0/{objectID}/media` | Collection examples; version-neutral, partial |
| instagram-login | POST | `/v25.0/{objectID}/media_publish` | Collection examples; version-neutral, partial |
| instagram-login | GET | `/v25.0/{objectID}/insights` | Collection examples; version-neutral, partial |
| instagram-login | GET | `/v25.0/{objectID}/comments` | Collection examples; version-neutral, partial |
| instagram-login | POST | `/v25.0/{objectID}/replies` | Collection examples; version-neutral, partial |
| webhooks | POST | `/webhooks/instagram` | Collection examples; version-neutral, partial |
### v26.0

| Product | Method | Path | Evidence / shape |
|---|---|---|---|
| facebook-login | GET | `/v26.0/{objectID}` | SDK fields; mutation/error details partial |
| facebook-login | DELETE | `/v26.0/{objectID}` | SDK fields; mutation/error details partial |
| facebook-login | POST | `/v26.0/{objectID}` | SDK fields; mutation/error details partial |
| facebook-login | GET | `/v26.0/{objectID}/agencies` | SDK fields; mutation/error details partial |
| facebook-login | GET | `/v26.0/{objectID}/authorized_adaccounts` | SDK fields; mutation/error details partial |
| facebook-login | POST | `/v26.0/{objectID}/authorized_adaccounts` | SDK fields; mutation/error details partial |
| facebook-login | GET | `/v26.0/{objectID}/available_catalogs` | SDK fields; mutation/error details partial |
| facebook-login | GET | `/v26.0/{objectID}/branded_content_ad_permissions` | SDK fields; mutation/error details partial |
| facebook-login | POST | `/v26.0/{objectID}/branded_content_ad_permissions` | SDK fields; mutation/error details partial |
| facebook-login | GET | `/v26.0/{objectID}/branded_content_advertisable_medias` | SDK fields; mutation/error details partial |
| facebook-login | DELETE | `/v26.0/{objectID}/branded_content_tag_approval` | SDK fields; mutation/error details partial |
| facebook-login | GET | `/v26.0/{objectID}/branded_content_tag_approval` | SDK fields; mutation/error details partial |
| facebook-login | POST | `/v26.0/{objectID}/branded_content_tag_approval` | SDK fields; mutation/error details partial |
| facebook-login | POST | `/v26.0/{objectID}/business_messaging_feature_status` | SDK fields; mutation/error details partial |
| facebook-login | GET | `/v26.0/{objectID}/catalog_product_search` | SDK fields; mutation/error details partial |
| facebook-login | GET | `/v26.0/{objectID}/collaboration_invites` | SDK fields; mutation/error details partial |
| facebook-login | POST | `/v26.0/{objectID}/collaboration_invites` | SDK fields; mutation/error details partial |
| facebook-login | GET | `/v26.0/{objectID}/collaborative_media` | SDK fields; mutation/error details partial |
| facebook-login | GET | `/v26.0/{objectID}/connected_threads_user` | SDK fields; mutation/error details partial |
| facebook-login | GET | `/v26.0/{objectID}/content_publishing_limit` | SDK fields; mutation/error details partial |
| facebook-login | GET | `/v26.0/{objectID}/creator_marketplace_creators` | SDK fields; mutation/error details partial |
| facebook-login | GET | `/v26.0/{objectID}/dataset` | SDK fields; mutation/error details partial |
| facebook-login | POST | `/v26.0/{objectID}/dataset` | SDK fields; mutation/error details partial |
| facebook-login | GET | `/v26.0/{objectID}/insights` | SDK fields; mutation/error details partial |
| facebook-login | GET | `/v26.0/{objectID}/instagram_backed_threads_user` | SDK fields; mutation/error details partial |
| facebook-login | POST | `/v26.0/{objectID}/instagram_backed_threads_user` | SDK fields; mutation/error details partial |
| facebook-login | GET | `/v26.0/{objectID}/live_media` | SDK fields; mutation/error details partial |
| facebook-login | GET | `/v26.0/{objectID}/media` | SDK fields; mutation/error details partial |
| facebook-login | POST | `/v26.0/{objectID}/media` | SDK fields; mutation/error details partial |
| facebook-login | POST | `/v26.0/{objectID}/media_publish` | SDK fields; mutation/error details partial |
| facebook-login | POST | `/v26.0/{objectID}/mentions` | SDK fields; mutation/error details partial |
| facebook-login | POST | `/v26.0/{objectID}/moderate_conversations` | SDK fields; mutation/error details partial |
| facebook-login | GET | `/v26.0/{objectID}/notification_message_tokens` | SDK fields; mutation/error details partial |
| facebook-login | GET | `/v26.0/{objectID}/product_appeal` | SDK fields; mutation/error details partial |
| facebook-login | POST | `/v26.0/{objectID}/product_appeal` | SDK fields; mutation/error details partial |
| facebook-login | GET | `/v26.0/{objectID}/recently_searched_hashtags` | SDK fields; mutation/error details partial |
| facebook-login | GET | `/v26.0/{objectID}/scheduled_media` | SDK fields; mutation/error details partial |
| facebook-login | GET | `/v26.0/{objectID}/stories` | SDK fields; mutation/error details partial |
| facebook-login | GET | `/v26.0/{objectID}/tags` | SDK fields; mutation/error details partial |
| facebook-login | GET | `/v26.0/{objectID}/upcoming_events` | SDK fields; mutation/error details partial |
| facebook-login | POST | `/v26.0/{objectID}/upcoming_events` | SDK fields; mutation/error details partial |
| facebook-login | GET | `/v26.0/{objectID}/welcome_message_flows` | SDK fields; mutation/error details partial |
| facebook-login | GET | `/v26.0/{objectID}/boost_ads_list` | SDK fields; mutation/error details partial |
| facebook-login | GET | `/v26.0/{objectID}/branded_content_partner_promote` | SDK fields; mutation/error details partial |
| facebook-login | POST | `/v26.0/{objectID}/branded_content_partner_promote` | SDK fields; mutation/error details partial |
| facebook-login | GET | `/v26.0/{objectID}/children` | SDK fields; mutation/error details partial |
| facebook-login | GET | `/v26.0/{objectID}/collaborators` | SDK fields; mutation/error details partial |
| facebook-login | GET | `/v26.0/{objectID}/comments` | SDK fields; mutation/error details partial |
| facebook-login | POST | `/v26.0/{objectID}/comments` | SDK fields; mutation/error details partial |
| facebook-login | DELETE | `/v26.0/{objectID}/partnership_ad_code` | SDK fields; mutation/error details partial |
| facebook-login | POST | `/v26.0/{objectID}/partnership_ad_code` | SDK fields; mutation/error details partial |
| facebook-login | GET | `/v26.0/{objectID}/product_tags` | SDK fields; mutation/error details partial |
| facebook-login | POST | `/v26.0/{objectID}/product_tags` | SDK fields; mutation/error details partial |
| facebook-login | GET | `/v26.0/{objectID}/replies` | SDK fields; mutation/error details partial |
| facebook-login | POST | `/v26.0/{objectID}/replies` | SDK fields; mutation/error details partial |
| facebook-login | GET | `/v26.0/{objectID}/recent_media` | SDK fields; mutation/error details partial |
| facebook-login | GET | `/v26.0/{objectID}/top_media` | SDK fields; mutation/error details partial |
| facebook-login | GET | `/v26.0/me/accounts` | SDK fields; mutation/error details partial |
| facebook-login | GET | `/v26.0/{objectID}/conversations` | SDK fields; mutation/error details partial |
| facebook-login | GET | `/v26.0/{objectID}/subscribed_apps` | SDK fields; mutation/error details partial |
| facebook-login | POST | `/v26.0/{objectID}/subscribed_apps` | SDK fields; mutation/error details partial |
| facebook-login | DELETE | `/v26.0/{objectID}/subscribed_apps` | SDK fields; mutation/error details partial |
| instagram-login | POST | `/v26.0/{objectID}/messages` | Collection examples; version-neutral, partial |
| instagram-login | POST | `/v26.0/{objectID}/message_attachments` | Collection examples; version-neutral, partial |
| instagram-login | GET | `/v26.0/{objectID}/conversations` | Collection examples; version-neutral, partial |
| instagram-login | GET | `/v26.0/{objectID}` | Collection examples; version-neutral, partial |
| instagram-login | GET | `/v26.0/{objectID}/messenger_profile` | Collection examples; version-neutral, partial |
| instagram-login | POST | `/v26.0/{objectID}/messenger_profile` | Collection examples; version-neutral, partial |
| instagram-login | DELETE | `/v26.0/{objectID}/messenger_profile` | Collection examples; version-neutral, partial |
| instagram-login | GET | `/v26.0/{objectID}/welcome_message_flows` | Collection examples; version-neutral, partial |
| instagram-login | POST | `/v26.0/{objectID}/welcome_message_flows` | Collection examples; version-neutral, partial |
| instagram-login | DELETE | `/v26.0/{objectID}/welcome_message_flows` | Collection examples; version-neutral, partial |
| instagram-login | GET | `/v26.0/{objectID}/subscribed_apps` | Collection examples; version-neutral, partial |
| instagram-login | POST | `/v26.0/{objectID}/subscribed_apps` | Collection examples; version-neutral, partial |
| instagram-login | DELETE | `/v26.0/{objectID}/subscribed_apps` | Collection examples; version-neutral, partial |
| instagram-login | GET | `/v26.0/me/conversations` | Collection examples; version-neutral, partial |
| instagram-login | POST | `/v26.0/{objectID}/media` | Collection examples; version-neutral, partial |
| instagram-login | POST | `/v26.0/{objectID}/media_publish` | Collection examples; version-neutral, partial |
| instagram-login | GET | `/v26.0/{objectID}/insights` | Collection examples; version-neutral, partial |
| instagram-login | GET | `/v26.0/{objectID}/comments` | Collection examples; version-neutral, partial |
| instagram-login | POST | `/v26.0/{objectID}/replies` | Collection examples; version-neutral, partial |
| webhooks | POST | `/webhooks/instagram` | Collection examples; version-neutral, partial |

## Coverage gaps and completion gates

These are deliberately absent or permissive, not silently marked complete:

1. OAuth authorization, code exchange, long-lived exchange, refresh, debug/revoke and permission management: official reference pages could not be retrieved. The official collection has only Page/account discovery, not these flows. No guessed OAuth server routes were added.
2. Resumable/binary media upload transport: SDK has an upload_type parameter, but that does not prove upload host, headers, body, chunking or retry contract. URL-based container creation is described; upload transport is pending.
3. Root hashtag search and complete business_discovery/mentioned_media/mentioned_comment field expansion: SDK confirms node fields and hashtag recent/top-media edges, not all root query syntax or restrictions.
4. Instagram Login reads/moderation/discovery beyond the exact official collection entries: Facebook Login SDK endpoints are not automatically copied across hosts. Story availability, account types, media restrictions, carousel limits, trial Reels and collaborator behavior need endpoint-specific release evidence.
5. Permissions: known product scope families are recorded below, but every operation’s complete app-review, granted-user-scope, role and asset requirements still need official per-version verification. Authentication security schemes are descriptive and do not validate permissions.
6. Exact errors, HTTP statuses, rate-limit headers, pagination boundaries, token lifetimes, retry safety and publication timing remain incomplete. Default responses are intentionally permissive; they are not retry advice.
7. Webhook verification/signatures/retries and all event variants: payload examples are modeled, but current cryptographic signature requirements have not been established. Callback /webhooks/instagram is an application-chosen template, not a Meta endpoint. Do not deploy a webhook receiver from these schemas alone.
8. Ads/commerce/Threads SDK target types outside the retrieved Instagram classes remain explicitly unmodeled objects. They are adjacent surface inventory, not ready-to-enforce validators.
9. No authorized live API tests were run. Documentation-based models must not be labeled live verified. Contract test recording/sanitization, SDK clients, and broad simulator behavior remain future implementation work.

## Permission and account distinctions

- Facebook Login collection names pages_show_list, instagram_basic, instagram_content_publish, pages_read_engagement and instagram_manage_comments. It requires a linked Facebook Page for Instagram professional accounts. These are a collection-wide list, not an assertion every operation requires all of them.
- Instagram Login collection names instagram_business_basic, instagram_business_content_publish, instagram_business_manage_messages and instagram_business_manage_comments. It does not require a linked Facebook Page. Older business_* names are not reused as current permissions.
- Messaging requires the correct recipient relationship, user initiation and messaging-specific permissions; private comment replies and HUMAN_AGENT are distinct variants, not blanket requirements for every send.
- Business discovery, tags and mentions are different operations. In the SDK, business_discovery and mentioned_* are fields on IGUser; they are not invented standalone paths.

## Source quality issues

The official collection includes unrelated welcome-flow responses under Text Message and Sticker. Those examples were excluded. Timestamp/edit-count placeholders and conflicting webhook comment nesting are marked permissively rather than guessed. Collection required fields are limited to documented request structure, not inferred from every example. SDK type declarations show client expectations, not proof of server validation or exhaustive enums.

## Sources

- [Meta official Instagram Postman collection, pinned](https://github.com/fbsamples/messenger-platform-samples/blob/3c3853a84a5d87d2567e75e649655822f535fdf6/postman/instagram-platform-api.postman_collection.json)
- [Meta official Business SDK](https://github.com/facebook/facebook-python-business-sdk); individual immutable links are embedded in every SDK operation and sdk-inventory.json.
- [Meta Instagram documentation](https://developers.facebook.com/docs/instagram-platform/) (inaccessible during review).

## Runtime boundary

The existing openapi.yaml files remain the smaller simulator contract. New *.openapi.yaml files are contract-only. Generated internal/contracts packages are deliberately not imported by providers or handlers. Unknown runtime endpoints/versions continue returning errors. Adding a contract never adds a success stub.

Run make spec-validate, make generate and make generate-check after edits. The validator checks version/path, contract-only provenance and host constraints. Generation uses pinned oapi-codegen; generation does not establish upstream accuracy.

Additional messaging source: [Meta Messenger Platform collection, pinned](https://github.com/fbsamples/messenger-platform-samples/blob/3c3853a84a5d87d2567e75e649655822f535fdf6/postman/messenger-platform-api.postman_collection.json).
Its Facebook-login messaging evidence is version-neutral; it does not establish per-release parity with the Instagram collection.

Facebook messaging Page/conversation fields also use SDK revision efd8423a2e595ea8d4c04eb824ce113f2f1d68cd. These supplementary source claims are explicitly version-neutral and do not independently verify v24/v25 messaging behavior.
