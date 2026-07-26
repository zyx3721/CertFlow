package notify

import (
	"encoding/json"
	"errors"
	"strings"
)

func normalizeWebhookHeaders(value any) (map[string]string, error) {
	if value == nil {
		return nil, nil
	}
	if text, ok := value.(string); ok {
		text = strings.TrimSpace(text)
		if text == "" {
			return nil, nil
		}
		value = json.RawMessage(text)
	}
	var headers map[string]string
	switch item := value.(type) {
	case json.RawMessage:
		if err := json.Unmarshal(item, &headers); err != nil {
			return nil, errors.New("请求头 JSON 格式不正确")
		}
	case map[string]string:
		headers = item
	case map[string]any:
		headers = make(map[string]string, len(item))
		for key, raw := range item {
			text, ok := raw.(string)
			if !ok {
				return nil, errors.New("请求头 JSON 格式不正确")
			}
			headers[key] = text
		}
	default:
		return nil, errors.New("请求头 JSON 格式不正确")
	}
	for key := range headers {
		if strings.TrimSpace(key) == "" {
			return nil, errors.New("请求头 JSON 格式不正确")
		}
	}
	return headers, nil
}

func webhookHeaders(config map[string]any) map[string]string {
	headers, err := normalizeWebhookHeaders(config["headers"])
	if err != nil {
		return nil
	}
	return headers
}
