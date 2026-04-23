package handler

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/LurusTech/lurus-api/internal/adapter/repo"
	"github.com/LurusTech/lurus-api/internal/pkg/common"
	"github.com/LurusTech/lurus-api/internal/pkg/constant"
)

// AutoSyncChannelModelsWithContext periodically fetches upstream model lists
// for all enabled channels and appends newly discovered models.
func AutoSyncChannelModelsWithContext(ctx context.Context, frequencyMinutes int) {
	if frequencyMinutes <= 0 {
		common.SysLog("model auto-sync disabled: invalid frequency")
		return
	}

	ticker := time.NewTicker(time.Duration(frequencyMinutes) * time.Minute)
	defer ticker.Stop()

	common.SysLog(fmt.Sprintf("model auto-sync started, frequency: %d minutes", frequencyMinutes))

	for {
		select {
		case <-ctx.Done():
			common.SysLog("model auto-sync stopped")
			return
		case <-ticker.C:
			syncAllChannelModels(ctx)
		}
	}
}

func syncAllChannelModels(ctx context.Context) {
	if repo.DB == nil {
		common.SysLog("model auto-sync: database not initialized, skipping")
		return
	}

	channels, err := repo.GetAllChannels(0, 0, true, false)
	if err != nil {
		common.SysLog("model auto-sync: failed to get channels: " + err.Error())
		return
	}

	synced, skipped, failed := 0, 0, 0
	for _, channel := range channels {
		select {
		case <-ctx.Done():
			return
		default:
		}

		if channel.Status != common.ChannelStatusEnabled {
			skipped++
			continue
		}

		newModels, err := fetchAndMergeModels(channel)
		if err != nil {
			failed++
			common.SysLog(fmt.Sprintf("model auto-sync: channel %d (%s) failed: %s", channel.Id, channel.Name, err.Error()))
			continue
		}

		if len(newModels) > 0 {
			common.SysLog(fmt.Sprintf("model auto-sync: channel %d (%s) found %d new models: %s",
				channel.Id, channel.Name, len(newModels), strings.Join(newModels, ", ")))
		}
		synced++
	}

	common.SysLog(fmt.Sprintf("model auto-sync complete: synced=%d, skipped=%d, failed=%d", synced, skipped, failed))
}

func fetchAndMergeModels(channel *repo.Channel) ([]string, error) {
	baseURL := constant.ChannelBaseURLs[channel.Type]
	if channel.GetBaseURL() != "" {
		baseURL = channel.GetBaseURL()
	}
	if baseURL == "" {
		return nil, fmt.Errorf("no base URL for channel type %d", channel.Type)
	}

	// Build the models endpoint URL based on channel type
	url := buildModelsURL(channel.Type, baseURL)

	// Get API key
	key, _, apiErr := channel.GetNextEnabledKey()
	if apiErr != nil {
		return nil, fmt.Errorf("no available key: %s", apiErr.Error())
	}
	key = strings.TrimSpace(key)

	headers, err := buildFetchModelsHeaders(channel, key)
	if err != nil {
		return nil, fmt.Errorf("build headers: %w", err)
	}

	body, err := GetResponseBody("GET", url, channel, headers)
	if err != nil {
		return nil, fmt.Errorf("fetch models: %w", err)
	}

	var result OpenAIModelsResponse
	if err = json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("parse response: %w", err)
	}

	// Extract model IDs from response
	upstreamModels := make(map[string]bool)
	for _, model := range result.Data {
		id := model.ID
		if channel.Type == constant.ChannelTypeGemini {
			id = strings.TrimPrefix(id, "models/")
		}
		if id != "" {
			upstreamModels[id] = true
		}
	}

	// Parse existing models
	existingModels := make(map[string]bool)
	if channel.Models != "" {
		for _, m := range strings.Split(channel.Models, ",") {
			m = strings.TrimSpace(m)
			if m != "" {
				existingModels[m] = true
			}
		}
	}

	// Find new models
	var newModels []string
	for m := range upstreamModels {
		if !existingModels[m] {
			newModels = append(newModels, m)
		}
	}

	if len(newModels) == 0 {
		return nil, nil
	}

	// Append new models to existing list
	updatedModels := channel.Models
	for _, m := range newModels {
		if updatedModels != "" {
			updatedModels += ","
		}
		updatedModels += m
	}

	// Update channel models in DB
	err = repo.UpdateChannelModels(channel.Id, updatedModels)
	if err != nil {
		return newModels, fmt.Errorf("update channel models: %w", err)
	}

	// Update abilities table
	channel.Models = updatedModels
	if err := channel.UpdateAbilities(nil); err != nil {
		common.SysLog(fmt.Sprintf("model auto-sync: channel %d failed to update abilities: %s", channel.Id, err.Error()))
	}

	// Auto-create model metadata entries for new models (Phase 1)
	ensureModelMetadataEntries(newModels, channel.Type)

	return newModels, nil
}

// channelTypeToVendorName maps channel types to vendor display names.
// Used when auto-creating model metadata entries during channel sync.
var channelTypeToVendorName = map[int]string{
	constant.ChannelTypeOpenAI:      "OpenAI",
	constant.ChannelTypeAnthropic:   "Anthropic",
	constant.ChannelTypeGemini:      "Google",
	constant.ChannelTypeAli:         "Alibaba",
	constant.ChannelTypeZhipu_v4:    "Zhipu",
	constant.ChannelTypeVolcEngine:  "Volcengine",
	constant.ChannelTypeMoonshot:    "Moonshot",
	constant.ChannelTypeDeepSeek:    "DeepSeek",
	constant.ChannelTypeBaidu:       "Baidu",
	constant.ChannelTypeBaiduV2:     "Baidu",
	constant.ChannelTypeMiniMax:     "MiniMax",
	constant.ChannelTypeXai:         "xAI",
	constant.ChannelTypeMistral:     "Mistral",
	constant.ChannelTypeCohere:      "Cohere",
	constant.ChannelTypeLingYiWanWu: "01.AI",
	constant.ChannelTypePerplexity:  "Perplexity",
	constant.ChannelTypeSiliconFlow: "SiliconFlow",
	constant.ChannelTypeAzure:       "Azure",
	constant.ChannelTypeAws:         "AWS",
	constant.ChannelTypeVertexAi:    "Google",
	constant.ChannelTypeTencent:     "Tencent",
}

// ensureModelMetadataEntries auto-creates model metadata for newly discovered models.
func ensureModelMetadataEntries(newModels []string, channelType int) {
	if len(newModels) == 0 {
		return
	}

	vendorName := channelTypeToVendorName[channelType]
	if vendorName == "" {
		vendorName = "Other"
	}

	vendorID, err := repo.GetOrCreateVendorByName(vendorName)
	if err != nil {
		common.SysLog(fmt.Sprintf("model auto-sync: failed to get/create vendor %q: %s", vendorName, err.Error()))
		return
	}

	created := 0
	for _, modelName := range newModels {
		exists, err := repo.IsModelNameDuplicated(0, modelName)
		if err != nil {
			common.SysLog(fmt.Sprintf("model auto-sync: failed to check model %q: %s", modelName, err.Error()))
			continue
		}
		if exists {
			continue
		}

		m := &repo.Model{
			ModelName:    modelName,
			VendorID:     vendorID,
			Status:       1,
			NameRule:     repo.NameRuleExact,
			SyncOfficial: 1,
		}
		if err := repo.ModelInsert(m); err != nil {
			common.SysLog(fmt.Sprintf("model auto-sync: failed to create model metadata for %q: %s", modelName, err.Error()))
			continue
		}
		created++
	}

	if created > 0 {
		common.SysLog(fmt.Sprintf("model auto-sync: auto-created %d model metadata entries (vendor=%s)", created, vendorName))
		repo.RefreshPricing()
	}
}

func buildModelsURL(channelType int, baseURL string) string {
	switch channelType {
	case constant.ChannelTypeGemini:
		return fmt.Sprintf("%s/v1beta/openai/models", baseURL)
	case constant.ChannelTypeAli:
		return fmt.Sprintf("%s/compatible-mode/v1/models", baseURL)
	case constant.ChannelTypeZhipu_v4:
		if plan, ok := constant.ChannelSpecialBases[baseURL]; ok && plan.OpenAIBaseURL != "" {
			return fmt.Sprintf("%s/models", plan.OpenAIBaseURL)
		}
		return fmt.Sprintf("%s/api/paas/v4/models", baseURL)
	case constant.ChannelTypeVolcEngine:
		if plan, ok := constant.ChannelSpecialBases[baseURL]; ok && plan.OpenAIBaseURL != "" {
			return fmt.Sprintf("%s/v1/models", plan.OpenAIBaseURL)
		}
		return fmt.Sprintf("%s/v1/models", baseURL)
	case constant.ChannelTypeMoonshot:
		if plan, ok := constant.ChannelSpecialBases[baseURL]; ok && plan.OpenAIBaseURL != "" {
			return fmt.Sprintf("%s/models", plan.OpenAIBaseURL)
		}
		return fmt.Sprintf("%s/v1/models", baseURL)
	default:
		return fmt.Sprintf("%s/v1/models", baseURL)
	}
}
