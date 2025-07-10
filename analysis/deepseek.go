package analysis

// DeepSeekConfig 用于存储API Key和API地址
var (
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
