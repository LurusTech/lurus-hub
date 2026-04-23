package app

import (
	"strings"

	"github.com/LurusTech/lurus-api/internal/pkg/constant"
)

func CoverTaskActionToModelName(platform constant.TaskPlatform, action string) string {
	return strings.ToLower(string(platform)) + "_" + strings.ToLower(action)
}
