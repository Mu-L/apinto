package google

import (
	"encoding/json"
	"fmt"

	"github.com/eolinker/eosc"

	"github.com/eolinker/apinto/convert"
	"github.com/eolinker/eosc/eocontext"
	http_context "github.com/eolinker/eosc/eocontext/http-context"
)

type FNewModelMode func(string) IModelMode

var (
	modelModes = map[string]FNewModelMode{
		convert.ModeChat.String(): NewChat,
	}
)

type ModelFactory struct {
}

type IModelMode interface {
	Endpoint() string
	convert.IConverter
}

type Chat struct {
	endPoint string
}

func NewChat(model string) IModelMode {
	return &Chat{
		endPoint: fmt.Sprintf("/v1beta/models/%s:generateContent", model),
	}
}

func (c *Chat) Endpoint() string {
	return c.endPoint
}

func (c *Chat) RequestConvert(ctx eocontext.EoContext, extender map[string]interface{}) error {
	httpContext, err := http_context.Assert(ctx)
	if err != nil {
		return err
	}
	body, err := httpContext.Proxy().Body().RawBody()
	if err != nil {
		return err
	}
	// 设置转发地址
	httpContext.Proxy().URI().SetPath(c.endPoint)
	baseCfg := eosc.NewBase[convert.ClientRequest]()
	err = json.Unmarshal(body, baseCfg)
	if err != nil {
		return err
	}
	messages := make([]Content, 0, len(baseCfg.Config.Messages)+1)
	for _, m := range baseCfg.Config.Messages {
		role := "user"
		if m.Role == "system" && len(baseCfg.Config.Messages) > 1 {
			role = "model"
		}
		parts := make([]map[string]interface{}, 0, 1)
		if m.Content != "" {
			parts = append(parts, map[string]interface{}{
				"text": m.Content,
			})
		}
		messages = append(messages, Content{
			Role:  role,
			Parts: parts,
		})
	}
	baseCfg.SetAppend("contents", messages)
	for k, v := range extender {
		baseCfg.SetAppend(k, v)
	}
	body, err = json.Marshal(baseCfg.Append)
	if err != nil {
		return err
	}
	httpContext.Proxy().Body().SetRaw("application/json", body)

	return nil
}

func (c *Chat) ResponseConvert(ctx eocontext.EoContext) error {
	httpContext, err := http_context.Assert(ctx)
	if err != nil {
		return err
	}
	body := httpContext.Response().GetBody()
	data := eosc.NewBase[Response]()
	err = json.Unmarshal(body, data)
	if err != nil {
		return err
	}
	// 针对不同响应做出处理
	switch httpContext.Response().StatusCode() {
	case 200:
		// Calculate the token consumption for a successful request.
		usage := data.Config.UsageMetadata
		convert.SetAIStatusNormal(ctx)
		convert.SetAIModelInputToken(ctx, usage.PromptTokenCount)
		convert.SetAIModelOutputToken(ctx, usage.CandidatesTokenCount)
		convert.SetAIModelTotalToken(ctx, usage.TotalTokenCount)
	case 400:
		// Handle the bad request error.
		convert.SetAIStatusInvalidRequest(ctx)
	case 429:
		// Handle exceed
		convert.SetAIStatusExceeded(ctx)
	case 401:
		// Handle authentication failure
		convert.SetAIStatusInvalid(ctx)
	}
	responseBody := &convert.ClientResponse{}
	if len(data.Config.Candidates) > 0 {
		msg := data.Config.Candidates[0]
		role := "user"
		if msg.Content.Role == "model" {
			role = "assistant"
		}
		text := ""
		if len(msg.Content.Parts) > 0 {
			if v, ok := msg.Content.Parts[0]["text"]; ok {
				text = v.(string)
			}
		}

		responseBody.Message = &convert.Message{
			Role:    role,
			Content: text,
		}
		responseBody.FinishReason = msg.FinishReason
	} else {
		responseBody.Code = -1
		responseBody.Error = data.Config.Error.Message
	}
	body, err = json.Marshal(responseBody)
	if err != nil {
		return err
	}
	httpContext.Response().SetBody(body)
	return nil
}
