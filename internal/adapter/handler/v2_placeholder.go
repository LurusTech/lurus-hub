package handler

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
// - GetBillingSummary: GET /api/v2/user/billing/summary (platform-wide, no tenant)
// - GetBillingPaymentMethods: GET /api/v2/user/billing/payment-methods
// - CreateBillingCheckout: POST /api/v2/user/billing/checkout
// - GetBillingCheckoutStatus: GET /api/v2/user/billing/checkout/:order_no/status
// - TopUpV2: POST /api/v2/:tenant_slug/billing/topup (wallet-to-quota transfer)
// - GetTopUpsV2: GET /api/v2/:tenant_slug/billing/topups (topup history)
//
// Redemption Controllers (v2_redemption.go):
// - RedeemCodeV2: POST /api/v2/:tenant_slug/redeem
// - ListRedemptionsV2: GET /api/v2/:tenant_slug/redemptions
// - CreateRedemptionV2: POST /api/v2/:tenant_slug/redemptions
// - DeleteRedemptionV2: DELETE /api/v2/:tenant_slug/redemptions/:id
//
// Client API Controllers (v2_client_api.go) — for other Lurus products:
// - ClientGetProfile: GET /api/v2/client/profile
// - ClientGetUsageSummary: GET /api/v2/client/usage/summary
// - ClientGetUsageByModel: GET /api/v2/client/usage/models
// - ClientGetUsageDaily: GET /api/v2/client/usage/daily
// - ClientGetTokens: GET /api/v2/client/tokens
// - ClientGetSessions: GET /api/v2/client/sessions
//
// Platform Admin Controllers (v2_admin.go):
// - ListUserMappingsV2: GET /api/v2/admin/mappings
// - GetUserMappingV2: GET /api/v2/admin/mappings/:id
// - DeleteUserMappingV2: DELETE /api/v2/admin/mappings/:id
// - GetSystemStatsV2: GET /api/v2/admin/stats
//
// ============================================================================
