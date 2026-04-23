package relay

import (
	"strconv"

	"github.com/QuantumNous/lurus-api/internal/pkg/constant"
	"github.com/QuantumNous/lurus-api/internal/adapter/provider"
	"github.com/QuantumNous/lurus-api/internal/adapter/provider/ali"
	"github.com/QuantumNous/lurus-api/internal/adapter/provider/aws"
	"github.com/QuantumNous/lurus-api/internal/adapter/provider/baidu"
	"github.com/QuantumNous/lurus-api/internal/adapter/provider/baidu_v2"
	"github.com/QuantumNous/lurus-api/internal/adapter/provider/claude"
	"github.com/QuantumNous/lurus-api/internal/adapter/provider/cloudflare"
	"github.com/QuantumNous/lurus-api/internal/adapter/provider/cohere"
	"github.com/QuantumNous/lurus-api/internal/adapter/provider/coze"
	"github.com/QuantumNous/lurus-api/internal/adapter/provider/deepseek"
	"github.com/QuantumNous/lurus-api/internal/adapter/provider/dify"
	"github.com/QuantumNous/lurus-api/internal/adapter/provider/gemini"
	"github.com/QuantumNous/lurus-api/internal/adapter/provider/jimeng"
	"github.com/QuantumNous/lurus-api/internal/adapter/provider/jina"
	"github.com/QuantumNous/lurus-api/internal/adapter/provider/minimax"
	"github.com/QuantumNous/lurus-api/internal/adapter/provider/mistral"
	"github.com/QuantumNous/lurus-api/internal/adapter/provider/mokaai"
	"github.com/QuantumNous/lurus-api/internal/adapter/provider/moonshot"
	"github.com/QuantumNous/lurus-api/internal/adapter/provider/ollama"
	"github.com/QuantumNous/lurus-api/internal/adapter/provider/openai"
	"github.com/QuantumNous/lurus-api/internal/adapter/provider/palm"
	"github.com/QuantumNous/lurus-api/internal/adapter/provider/perplexity"
	"github.com/QuantumNous/lurus-api/internal/adapter/provider/replicate"
	"github.com/QuantumNous/lurus-api/internal/adapter/provider/siliconflow"
	"github.com/QuantumNous/lurus-api/internal/adapter/provider/submodel"
	taskali "github.com/QuantumNous/lurus-api/internal/adapter/provider/task/ali"
	taskdoubao "github.com/QuantumNous/lurus-api/internal/adapter/provider/task/doubao"
	taskGemini "github.com/QuantumNous/lurus-api/internal/adapter/provider/task/gemini"
	"github.com/QuantumNous/lurus-api/internal/adapter/provider/task/hailuo"
	taskjimeng "github.com/QuantumNous/lurus-api/internal/adapter/provider/task/jimeng"
	"github.com/QuantumNous/lurus-api/internal/adapter/provider/task/kling"
	tasksora "github.com/QuantumNous/lurus-api/internal/adapter/provider/task/sora"
	taskmusic "github.com/QuantumNous/lurus-api/internal/adapter/provider/task/music"
	"github.com/QuantumNous/lurus-api/internal/adapter/provider/task/suno"
	taskvertex "github.com/QuantumNous/lurus-api/internal/adapter/provider/task/vertex"
	taskVidu "github.com/QuantumNous/lurus-api/internal/adapter/provider/task/vidu"
	"github.com/QuantumNous/lurus-api/internal/adapter/provider/tencent"
	"github.com/QuantumNous/lurus-api/internal/adapter/provider/vertex"
	"github.com/QuantumNous/lurus-api/internal/adapter/provider/volcengine"
	"github.com/QuantumNous/lurus-api/internal/adapter/provider/xai"
	"github.com/QuantumNous/lurus-api/internal/adapter/provider/xunfei"
	"github.com/QuantumNous/lurus-api/internal/adapter/provider/zhipu"
	"github.com/QuantumNous/lurus-api/internal/adapter/provider/zhipu_4v"
	"github.com/gin-gonic/gin"
)

func GetAdaptor(apiType int) provider.Adaptor {
	switch apiType {
	case constant.APITypeAli:
		return &ali.Adaptor{}
	case constant.APITypeAnthropic:
		return &claude.Adaptor{}
	case constant.APITypeBaidu:
		return &baidu.Adaptor{}
	case constant.APITypeGemini:
		return &gemini.Adaptor{}
	case constant.APITypeOpenAI:
		return &openai.Adaptor{}
	case constant.APITypePaLM:
		return &palm.Adaptor{}
	case constant.APITypeTencent:
		return &tencent.Adaptor{}
	case constant.APITypeXunfei:
		return &xunfei.Adaptor{}
	case constant.APITypeZhipu:
		return &zhipu.Adaptor{}
	case constant.APITypeZhipuV4:
		return &zhipu_4v.Adaptor{}
	case constant.APITypeOllama:
		return &ollama.Adaptor{}
	case constant.APITypePerplexity:
		return &perplexity.Adaptor{}
	case constant.APITypeAws:
		return &aws.Adaptor{}
	case constant.APITypeCohere:
		return &cohere.Adaptor{}
	case constant.APITypeDify:
		return &dify.Adaptor{}
	case constant.APITypeJina:
		return &jina.Adaptor{}
	case constant.APITypeCloudflare:
		return &cloudflare.Adaptor{}
	case constant.APITypeSiliconFlow:
		return &siliconflow.Adaptor{}
	case constant.APITypeVertexAi:
		return &vertex.Adaptor{}
	case constant.APITypeMistral:
		return &mistral.Adaptor{}
	case constant.APITypeDeepSeek:
		return &deepseek.Adaptor{}
	case constant.APITypeMokaAI:
		return &mokaai.Adaptor{}
	case constant.APITypeVolcEngine:
		return &volcengine.Adaptor{}
	case constant.APITypeBaiduV2:
		return &baidu_v2.Adaptor{}
	case constant.APITypeOpenRouter:
		return &openai.Adaptor{}
	case constant.APITypeXinference:
		return &openai.Adaptor{}
	case constant.APITypeXai:
		return &xai.Adaptor{}
	case constant.APITypeCoze:
		return &coze.Adaptor{}
	case constant.APITypeJimeng:
		return &jimeng.Adaptor{}
	case constant.APITypeMoonshot:
		return &moonshot.Adaptor{} // Moonshot uses Claude API
	case constant.APITypeSubmodel:
		return &submodel.Adaptor{}
	case constant.APITypeMiniMax:
		return &minimax.Adaptor{}
	case constant.APITypeReplicate:
		return &replicate.Adaptor{}
	}
	return nil
}

func GetTaskPlatform(c *gin.Context) constant.TaskPlatform {
	channelType := c.GetInt("channel_type")
	if channelType > 0 {
		return constant.TaskPlatform(strconv.Itoa(channelType))
	}
	return constant.TaskPlatform(c.GetString("platform"))
}

func GetTaskAdaptor(platform constant.TaskPlatform) provider.TaskAdaptor {
	switch platform {
	//case constant.APITypeAIProxyLibrary:
	//	return &aiproxy.Adaptor{}
	case constant.TaskPlatformSuno:
		return &suno.TaskAdaptor{}
	case constant.TaskPlatformMusic:
		return &taskmusic.TaskAdaptor{}
	}
	if channelType, err := strconv.ParseInt(string(platform), 10, 64); err == nil {
		switch channelType {
		case constant.ChannelTypeAli:
			return &taskali.TaskAdaptor{}
		case constant.ChannelTypeKling:
			return &kling.TaskAdaptor{}
		case constant.ChannelTypeJimeng:
			return &taskjimeng.TaskAdaptor{}
		case constant.ChannelTypeVertexAi:
			return &taskvertex.TaskAdaptor{}
		case constant.ChannelTypeVidu:
			return &taskVidu.TaskAdaptor{}
		case constant.ChannelTypeDoubaoVideo:
			return &taskdoubao.TaskAdaptor{}
		case constant.ChannelTypeSora, constant.ChannelTypeOpenAI:
			return &tasksora.TaskAdaptor{}
		case constant.ChannelTypeGemini:
			return &taskGemini.TaskAdaptor{}
		case constant.ChannelTypeMiniMax:
			return &hailuo.TaskAdaptor{}
		}
	}
	return nil
}
