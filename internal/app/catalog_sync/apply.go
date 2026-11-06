package catalog_sync

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/LurusTech/lurus-hub/internal/adapter/repo"
	"github.com/LurusTech/lurus-hub/internal/domain/entity"
	"github.com/LurusTech/lurus-hub/internal/pkg/setting/ratio_setting"
)

// ConfirmStaleRequest removes confirmed stale models from a channel.
type ConfirmStaleRequest struct {
	SourceVersion string   `json:"source_version"`
	ChannelId     int      `json:"channel_id"`
	Models        []string `json:"models"`
}

// ApplyPriceRequest applies selected price proposals into live model_ratio.
type ApplyPriceRequest struct {
	SourceVersion string  `json:"source_version"`
	IDs           []int64 `json:"ids"`
}

// ConfirmStaleRemovals drops selected stale models from channel.models + abilities.
func ConfirmStaleRemovals(req ConfirmStaleRequest) error {
	if req.ChannelId <= 0 || len(req.Models) == 0 {
		return ErrEmptySelection
	}
	if req.SourceVersion == "" {
		return ErrSourceVersionMismatch
	}
	var open []entity.CatalogStalePending
	if err := db().Where("channel_id = ? AND resolved_at IS NULL AND model_name IN ?", req.ChannelId, req.Models).
		Find(&open).Error; err != nil {
		return err
	}
	if len(open) == 0 {
		return ErrEmptySelection
	}
	for _, row := range open {
		if row.SourceVersion != "" && row.SourceVersion != req.SourceVersion {
			return ErrSourceVersionMismatch
		}
	}
	ch, err := repo.GetChannelById(req.ChannelId, true)
	if err != nil {
		return err
	}
	remove := setOf(req.Models)
	kept := make([]string, 0)
	for _, m := range splitModels(ch.Models) {
		if !remove[m] {
			kept = append(kept, m)
		}
	}
	ch.Models = joinModels(kept)
	if err := ch.Update(); err != nil {
		return err
	}
	now := time.Now().UTC()
	return db().Model(&entity.CatalogStalePending{}).
		Where("channel_id = ? AND model_name IN ?", req.ChannelId, req.Models).
		Update("resolved_at", now).Error
}

// ApplyPriceProposals merges selected upstream ratios into live model_ratio options.
func ApplyPriceProposals(req ApplyPriceRequest) error {
	if len(req.IDs) == 0 {
		return ErrEmptySelection
	}
	if req.SourceVersion == "" {
		return ErrSourceVersionMismatch
	}
	var rows []entity.CatalogPriceProposal
	if err := db().Where("id IN ? AND applied_at IS NULL AND dismissed_at IS NULL", req.IDs).Find(&rows).Error; err != nil {
		return err
	}
	if len(rows) == 0 {
		return ErrEmptySelection
	}
	for _, row := range rows {
		if row.SourceVersion != "" && row.SourceVersion != req.SourceVersion {
			return ErrSourceVersionMismatch
		}
		if row.Field != "model_ratio" {
			return fmt.Errorf("catalog_sync: unsupported price field %q", row.Field)
		}
	}
	ratios := ratio_setting.GetModelRatioCopy()
	for _, row := range rows {
		ratios[row.ModelName] = row.UpstreamValue
	}
	raw, err := json.Marshal(ratios)
	if err != nil {
		return err
	}
	if err := ratio_setting.UpdateModelRatioByJSONString(string(raw)); err != nil {
		return err
	}
	// Persist option row used by boot/reload.
	if err := repo.UpdateOption("ModelRatio", string(raw)); err != nil {
		return err
	}
	now := time.Now().UTC()
	return db().Model(&entity.CatalogPriceProposal{}).
		Where("id IN ?", req.IDs).
		Update("applied_at", now).Error
}

// DismissPriceProposals marks proposals as dismissed without applying.
func DismissPriceProposals(ids []int64, sourceVersion string) error {
	if len(ids) == 0 {
		return ErrEmptySelection
	}
	var rows []entity.CatalogPriceProposal
	if err := db().Where("id IN ? AND applied_at IS NULL AND dismissed_at IS NULL", ids).Find(&rows).Error; err != nil {
		return err
	}
	if len(rows) == 0 {
		return ErrEmptySelection
	}
	if sourceVersion != "" {
		for _, row := range rows {
			if row.SourceVersion != "" && row.SourceVersion != sourceVersion {
				return ErrSourceVersionMismatch
			}
		}
	}
	now := time.Now().UTC()
	return db().Model(&entity.CatalogPriceProposal{}).
		Where("id IN ?", ids).
		Update("dismissed_at", now).Error
}
