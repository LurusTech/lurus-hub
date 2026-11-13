package catalog_sync

import (
	"context"
	"fmt"
	"time"

	"github.com/LurusTech/lurus-hub/internal/pkg/common"
)

// AutoCatalogSyncWithContext is the leader-only ticker for daily catalog sync.
// frequencyMinutes <= 0 disables the loop; manual API triggers still work.
func AutoCatalogSyncWithContext(ctx context.Context, frequencyMinutes int) {
	if !common.IsMasterNode {
		common.SysLog("catalog_sync: skipped on slave node")
		return
	}
	if frequencyMinutes <= 0 {
		common.SysLog("catalog_sync: disabled (frequency <= 0)")
		return
	}

	ticker := time.NewTicker(time.Duration(frequencyMinutes) * time.Minute)
	defer ticker.Stop()

	common.SysLog(fmt.Sprintf("catalog_sync: started, frequency=%d min", frequencyMinutes))
	engine := NewEngine()

	for {
		select {
		case <-ctx.Done():
			common.SysLog("catalog_sync: stopped")
			return
		case <-ticker.C:
			if !common.IsLeader() {
				continue
			}
			result, err := engine.Run(ctx)
			if err != nil {
				common.SysLog("catalog_sync tick failed: " + err.Error())
				continue
			}
			added, stale, breakers, errs := 0, 0, 0, 0
			for _, ch := range result.Channels {
				added += len(ch.Added)
				stale += len(ch.Stale)
				if ch.CircuitBreaker {
					breakers++
				}
				if ch.Error != "" {
					errs++
				}
			}
			common.SysLog(fmt.Sprintf(
				"catalog_sync tick: channels=%d added=%d stale=%d breakers=%d errors=%d",
				len(result.Channels), added, stale, breakers, errs,
			))
		}
	}
}
