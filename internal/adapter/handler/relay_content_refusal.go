package handler

import (
	"strings"

	"github.com/LurusTech/lurus-hub/internal/pkg/types"
)

// isProviderContentRefusal reports a 400 whose text is a provider safety
// filter rather than a bad payload. Another channel may accept the same
// prompt. Context-window 400s stay non-retryable: sibling channels of the
// same model name share the window, so a retry cannot succeed.
func isProviderContentRefusal(err *types.NewAPIError) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error() + " " + err.UpstreamBodyHint)
	for _, needle := range []string{
		"content_policy",
		"content policy",
		"content_filter",
		"content filter",
		"responsibleaipolicyviolation",
		"usage policy",
		"content management policy",
	} {
		if strings.Contains(msg, needle) {
			return true
		}
	}
	return false
}
