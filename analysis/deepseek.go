package analysis

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
)

// DeepSeekConfig 用于存储API Key和API地址
var (
	DeepSeekAPIKey = os.Getenv("DEEPSEEK_API_KEY")
	DeepSeekAPIURL = "https://openrouter.ai/api/v1/chat/completions" // 可配置
	DeepSeekModel  = "deepseek/deepseek-r1:free"                     // 可配置
)

type deepSeekRequest struct {
	Model    string        `json:"model"`
	Messages []deepMessage `json:"messages"`
}

type deepMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type deepSeekResponse struct {
	Choices []struct {
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
	} `json:"choices"`
}

// GenerateAIReportWithConfig 直接传递 key、url、model
func GenerateAIReportWithConfig(stockCode, analysisData, apiKey, apiURL, model string) (string, error) {
	if apiKey == "" {
		return "", fmt.Errorf("DeepSeek API Key 未设置")
	}
	prompt := fmt.Sprintf("请根据以下股票分析数据，生成一份面向投资者的详细分析报告，内容包括基本面、技术面、风险、未来趋势、操作建议等：\n股票代码：%s\n%s", stockCode, analysisData)
	requestBody := deepSeekRequest{
		Model: model,
		Messages: []deepMessage{{
			Role:    "user",
			Content: prompt,
		}},
	}
	body, err := json.Marshal(requestBody)
	if err != nil {
		return "", err
	}
	client := &http.Client{}
	req, err := http.NewRequest("POST", apiURL, bytes.NewBuffer(body))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+apiKey)
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	respBody, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	var dsResp deepSeekResponse
	if err := json.Unmarshal(respBody, &dsResp); err != nil {
		return "", err
	}
	if len(dsResp.Choices) == 0 {
		return "", fmt.Errorf("DeepSeek API 无返回内容")
	}
	return dsResp.Choices[0].Message.Content, nil
}

// GenerateAIReportWithConfigAndSearch 支持 search 参数
func GenerateAIReportWithConfigAndSearch(stockCode, analysisData, apiKey, apiURL, model string, search bool) (string, error) {
	if apiKey == "" {
		return "", fmt.Errorf("DeepSeek API Key 未设置")
	}
	requestBody := deepSeekRequest{
		Model: model,
		Messages: []deepMessage{{
			Role:    "user",
			Content: analysisData,
		}},
	}
	if search {
		// 仅在需要联网搜索时加字段
		tmp := map[string]interface{}{}
		b, _ := json.Marshal(requestBody)
		json.Unmarshal(b, &tmp)
		tmp["search"] = true
		b2, _ := json.Marshal(tmp)
		return callDeepSeekAPI(apiKey, apiURL, b2)
	}
	body, err := json.Marshal(requestBody)
	if err != nil {
		return "", err
	}
	return callDeepSeekAPI(apiKey, apiURL, body)
}

func callDeepSeekAPI(apiKey, apiURL string, body []byte) (string, error) {
	client := &http.Client{}
	req, err := http.NewRequest("POST", apiURL, bytes.NewBuffer(body))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+apiKey)
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	respBody, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	var dsResp deepSeekResponse
	if err := json.Unmarshal(respBody, &dsResp); err != nil {
		return "", err
	}
	if len(dsResp.Choices) == 0 {
		return "", fmt.Errorf("DeepSeek API 无返回内容")
	}
	return dsResp.Choices[0].Message.Content, nil
}
