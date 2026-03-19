package handler

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/QuantumNous/lurus-api/internal/pkg/common"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// getIdentityAccountID extracts the platform account ID set by ZitadelAuth middleware.
// Returns 0 if not available (platform was unreachable during auth).
func getIdentityAccountID(c *gin.Context) int64 {
	v, ok := c.Get("identity_account_id")
	if !ok {
		return 0
	}
	id, ok := v.(int64)
	if !ok {
		return 0
	}
	return id
}

// GetBillingSummary returns the authenticated user's aggregated billing info
// from lurus-platform (wallet balance, frozen, pre-auths, pending orders).
// GET /api/v2/user/billing/summary
func GetBillingSummary(c *gin.Context) {
	accountID := getIdentityAccountID(c)
	if accountID == 0 {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"success": false,
			"message": "platform account not linked",
		})
		return
	}

	bs, err := common.GetBillingSummary(c.Request.Context(), accountID)
	if err != nil || bs == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"success": false,
			"message": "billing service unavailable",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    bs,
	})
}

// GetBillingPaymentMethods returns available payment methods from lurus-platform.
// GET /api/v2/user/billing/payment-methods
func GetBillingPaymentMethods(c *gin.Context) {
	methods, _ := common.GetPaymentMethods(c.Request.Context())
	if methods == nil {
		methods = []common.PaymentMethod{}
	}
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    methods,
	})
}

// createCheckoutRequest is the input for creating a checkout session.
type createCheckoutRequest struct {
	AmountCNY     float64 `json:"amount_cny" binding:"required,gt=0"`
	PaymentMethod string  `json:"payment_method" binding:"required"`
	ReturnURL     string  `json:"return_url"`
}

// CreateBillingCheckout creates a wallet topup checkout session via lurus-platform.
// POST /api/v2/user/billing/checkout
func CreateBillingCheckout(c *gin.Context) {
	accountID := getIdentityAccountID(c)
	if accountID == 0 {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"success": false,
			"message": "platform account not linked",
		})
		return
	}

	var req createCheckoutRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": fmt.Sprintf("invalid request: %v", err),
		})
		return
	}

	// Generate idempotency key from request context to prevent duplicate orders
	idempotencyKey := fmt.Sprintf("api-%d-%s", accountID, uuid.New().String()[:8])

	result, err := common.CreateCheckout(
		c.Request.Context(),
		accountID,
		req.AmountCNY,
		req.PaymentMethod,
		"lurus-api",
		idempotencyKey,
		req.ReturnURL,
	)
	if err != nil {
		status := http.StatusServiceUnavailable
		msg := "checkout service unavailable"
		if strings.Contains(err.Error(), "insufficient") {
			status = http.StatusBadRequest
			msg = err.Error()
		}
		c.JSON(status, gin.H{
			"success": false,
			"message": msg,
		})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"success": true,
		"data":    result,
	})
}

// GetBillingCheckoutStatus polls the status of a checkout order.
// GET /api/v2/user/billing/checkout/:order_no/status
func GetBillingCheckoutStatus(c *gin.Context) {
	orderNo := c.Param("order_no")
	if orderNo == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "order_no is required",
		})
		return
	}

	status, err := common.GetCheckoutStatus(c.Request.Context(), orderNo)
	if err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"success": false,
			"message": "checkout status unavailable",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    status,
	})
}
