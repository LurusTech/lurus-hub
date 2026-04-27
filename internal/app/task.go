package app

import (
	"strings"

	"github.com/LurusTech/lurus-hub/internal/pkg/constant"
)

func CoverTaskActionToModelName(platform constant.TaskPlatform, action string) string {
	return strings.ToLower(string(platform)) + "_" + strings.ToLower(action)
}
