package analysis

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
)

// 发送钉钉/企业微信 webhook 消息
func SendWebhook(webhookURL, content string) error {
	body := map[string]interface{}{
		"msgtype": "text",
		"text":    map[string]string{"content": content},
	}
	b, _ := json.Marshal(body)
	resp, err := http.Post(webhookURL, "application/json", bytes.NewReader(b))
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return fmt.Errorf("Webhook 推送失败: %s", resp.Status)
	}
	return nil
}
