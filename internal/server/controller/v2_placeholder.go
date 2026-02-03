package controller

// ============================================================================
// V2 API Controller Status
// ============================================================================
//
// All V2 controllers have been implemented in separate files:
//
// User Controllers (v2_user.go):
// - GetSelfV2: GET /api/v2/:tenant_slug/user/me
// - UpdateSelfV2: PUT /api/v2/:tenant_slug/user/me
//
// Token Controllers (v2_token.go):
// - ListTokensV2: GET /api/v2/:tenant_slug/tokens
// - CreateTokenV2: POST /api/v2/:tenant_slug/tokens
// - UpdateTokenV2: PUT /api/v2/:tenant_slug/tokens/:id
// - DeleteTokenV2: DELETE /api/v2/:tenant_slug/tokens/:id
//
// Log Controllers (v2_log.go):
// - GetLogsV2: GET /api/v2/:tenant_slug/logs
// - GetAllLogsV2: GET /api/v2/:tenant_slug/logs/all
//
// Channel Controllers (v2_channel.go):
// - ListChannelsV2: GET /api/v2/:tenant_slug/channels
// - GetChannelV2: GET /api/v2/:tenant_slug/channels/:id
// - CreateChannelV2: POST /api/v2/:tenant_slug/channels
// - UpdateChannelV2: PUT /api/v2/:tenant_slug/channels/:id
// - DeleteChannelV2: DELETE /api/v2/:tenant_slug/channels/:id
//
// Billing Controllers (v2_billing.go):
// - GetTopUpsV2: GET /api/v2/:tenant_slug/billing/topups
// - TopUpV2: POST /api/v2/:tenant_slug/billing/topup
// - GetSubscriptionsV2: GET /api/v2/:tenant_slug/billing/subscriptions
// - SubscribeV2: POST /api/v2/:tenant_slug/billing/subscribe
// - CancelSubscriptionV2: POST /api/v2/:tenant_slug/billing/subscriptions/:id/cancel
//
// Redemption Controllers (v2_redemption.go):
// - RedeemCodeV2: POST /api/v2/:tenant_slug/redeem
// - ListRedemptionsV2: GET /api/v2/:tenant_slug/redemptions
// - CreateRedemptionV2: POST /api/v2/:tenant_slug/redemptions
// - DeleteRedemptionV2: DELETE /api/v2/:tenant_slug/redemptions/:id
//
// Platform Admin Controllers (v2_admin.go):
// - ListUserMappingsV2: GET /api/v2/admin/mappings
// - GetUserMappingV2: GET /api/v2/admin/mappings/:id
// - DeleteUserMappingV2: DELETE /api/v2/admin/mappings/:id
// - GetSystemStatsV2: GET /api/v2/admin/stats
//
// ============================================================================
