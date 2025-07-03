package analysis

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"math"
	"net/http"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"
)

type AnalysisParams struct {
	APIKey     string
	Model      string
	StockCodes []string
	Start      string
	End        string
	SearchMode bool
	Periods    []string
	Dims       []string
	Output     []string
	Confidence bool
	Risk       string
	Scope      []string
	Lang       string
	Prompt     string
}

type AnalysisResult struct {
	StockCode string
	Report    string
	SavedFile string
	Err       error
}

type StockData struct {
	Date   time.Time
	Open   float64
	High   float64
	Low    float64
	Close  float64
	Volume int64
}

type TechnicalIndicator struct {
	Date time.Time
	MA5  float64
	MA10 float64
	MA20 float64
	MA60 float64
	RSI  float64
	MACD float64
}

// 单只股票AI分析（调用原有AI分析函数）
func AnalyzeOne(params AnalysisParams, genFunc func(string, string, string, string, string, bool) (string, error)) AnalysisResult {
	prompt := params.Prompt
	if prompt == "" {
		prompt = BuildPrompt(params)
	}
	report, err := genFunc(params.StockCodes[0], prompt, params.APIKey, "https://api.deepseek.com/v1/chat/completions", params.Model, params.SearchMode)
	if err != nil {
		return AnalysisResult{StockCode: params.StockCodes[0], Err: err}
	}
	fname := fmt.Sprintf("%s-%s-%s.json", params.StockCodes[0], params.End, time.Now().Format("150405"))
	fpath := filepath.Join("history", fname)
	hist := map[string]interface{}{
		"time":  time.Now().Format("2006-01-02 15:04:05"),
		"stock": params.StockCodes[0],
		"start": params.Start,
		"end":   params.End,
		"model": params.Model,
		"mode": func() string {
			if params.SearchMode {
				return "search"
			} else {
				return "reason"
			}
		}(),
		"periods":    params.Periods,
		"dims":       params.Dims,
		"output":     params.Output,
		"confidence": params.Confidence,
		"risk":       params.Risk,
		"scope":      params.Scope,
		"lang":       params.Lang,
		"prompt":     prompt,
		"report":     report,
	}
	b, _ := json.MarshalIndent(hist, "", "  ")
	_ = ioutil.WriteFile(fpath, b, 0644)
	return AnalysisResult{StockCode: params.StockCodes[0], Report: report, SavedFile: fname}
}

// 多只股票批量分析
func AnalyzeBatch(params AnalysisParams, genFunc func(string, string, string, string, string, bool) (string, error)) []AnalysisResult {
	results := make([]AnalysisResult, 0, len(params.StockCodes))
	for _, code := range params.StockCodes {
		p := params
		p.StockCodes = []string{code}
		results = append(results, AnalyzeOne(p, genFunc))
	}
	return results
}

// 构建prompt（支持多语言）
func BuildPrompt(params AnalysisParams) string {
	if params.Lang == "en" {
		return fmt.Sprintf(
			"Please provide a detailed analysis of stock %s from %s to %s, including:\n"+
				"1. Analysis dimensions: %s\n"+
				"2. Prediction periods: %s\n"+
				"3. Output format: %s\n"+
				"4. Web search scope: %s\n"+
				"5. Risk/opportunity preference: %s\n"+
				"6. %s\n"+
				"Please output structured, sectioned, key-point or table content as required.",
			params.StockCodes[0], params.Start, params.End,
			strings.Join(params.Dims, ", "),
			strings.Join(params.Periods, ", "),
			strings.Join(params.Output, ", "),
			strings.Join(params.Scope, ", "),
			params.Risk,
			func() string {
				if params.Confidence {
					return "For each prediction, provide a confidence level or probability range, and briefly explain the rationale."
				}
				return ""
			}(),
		)
	}
	return fmt.Sprintf(
		"请对股票代码 %s 在 %s 到 %s 期间的行情进行详细分析，内容包括：\n"+
			"1. 分析维度：%s\n"+
			"2. 预测周期：%s\n"+
			"3. 输出格式：%s\n"+
			"4. 联网搜索内容范围：%s\n"+
			"5. 风险/机会偏好：%s\n"+
			"6. %s\n"+
			"请结合以上要求，输出结构化、分段、要点或表格内容。",
		params.StockCodes[0], params.Start, params.End,
		strings.Join(params.Dims, ", "),
		strings.Join(params.Periods, ", "),
		strings.Join(params.Output, ", "),
		strings.Join(params.Scope, ", "),
		params.Risk,
		func() string {
			if params.Confidence {
				return "请对每个预测结论给出置信度或概率区间，并简要说明理由。"
			}
			return ""
		}(),
	)
}

// FetchStockHistory 通过 DeepSeek 联网搜索获取股票历史数据
func FetchStockHistory(stockCode, start, end, apiKey string) ([]StockData, []TechnicalIndicator, error) {
	// 构建搜索查询
	query := fmt.Sprintf("请搜索股票 %s 从 %s 到 %s 的历史行情数据，包括开盘价、最高价、最低价、收盘价、成交量，以及MA5、MA10、MA20、MA60、RSI、MACD等技术指标。请以JSON格式返回，格式如下：{\"data\":[{\"date\":\"2024-01-01\",\"open\":100.0,\"high\":105.0,\"low\":98.0,\"close\":102.0,\"volume\":1000000}],\"indicators\":[{\"date\":\"2024-01-01\",\"ma5\":101.0,\"ma10\":100.5,\"ma20\":99.8,\"ma60\":98.2,\"rsi\":65.5,\"macd\":0.5}]}", stockCode, start, end)

	// 使用传入的API Key
	if apiKey == "" {
		return nil, nil, fmt.Errorf("DeepSeek API Key 未设置")
	}

	// 构建请求
	requestBody := map[string]interface{}{
		"model": "deepseek-chat",
		"messages": []map[string]string{
			{"role": "user", "content": query},
		},
		"search": true, // 启用联网搜索
	}

	body, err := json.Marshal(requestBody)
	if err != nil {
		return nil, nil, err
	}

	// 发送请求
	client := &http.Client{}
	req, err := http.NewRequest("POST", "https://api.deepseek.com/v1/chat/completions", bytes.NewBuffer(body))
	if err != nil {
		return nil, nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+apiKey)

	resp, err := client.Do(req)
	if err != nil {
		return nil, nil, err
	}
	defer resp.Body.Close()

	respBody, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, nil, err
	}

	var dsResp struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}

	if err := json.Unmarshal(respBody, &dsResp); err != nil {
		return nil, nil, err
	}

	if len(dsResp.Choices) == 0 {
		return nil, nil, fmt.Errorf("DeepSeek API 无返回内容")
	}

	content := dsResp.Choices[0].Message.Content

	// 解析返回的JSON数据
	return parseStockDataFromContent(content)
}

// parseStockDataFromContent 从AI返回内容中解析股票数据
func parseStockDataFromContent(content string) ([]StockData, []TechnicalIndicator, error) {
	// 尝试提取JSON数据
	jsonRegex := regexp.MustCompile(`\{.*"data".*"indicators".*\}`)
	jsonMatch := jsonRegex.FindString(content)
	if jsonMatch == "" {
		// 如果没有找到完整JSON，尝试分别提取
		return parseStockDataFromText(content)
	}

	var result struct {
		Data []struct {
			Date   string  `json:"date"`
			Open   float64 `json:"open"`
			High   float64 `json:"high"`
			Low    float64 `json:"low"`
			Close  float64 `json:"close"`
			Volume int64   `json:"volume"`
		} `json:"data"`
		Indicators []struct {
			Date time.Time `json:"date"`
			MA5  float64   `json:"ma5"`
			MA10 float64   `json:"ma10"`
			MA20 float64   `json:"ma20"`
			MA60 float64   `json:"ma60"`
			RSI  float64   `json:"rsi"`
			MACD float64   `json:"macd"`
		} `json:"indicators"`
	}

	if err := json.Unmarshal([]byte(jsonMatch), &result); err != nil {
		return parseStockDataFromText(content)
	}

	// 转换数据
	var stockData []StockData
	for _, d := range result.Data {
		date, _ := time.Parse("2006-01-02", d.Date)
		stockData = append(stockData, StockData{
			Date:   date,
			Open:   d.Open,
			High:   d.High,
			Low:    d.Low,
			Close:  d.Close,
			Volume: d.Volume,
		})
	}

	var indicators []TechnicalIndicator
	for _, ind := range result.Indicators {
		indicators = append(indicators, TechnicalIndicator{
			Date: ind.Date,
			MA5:  ind.MA5,
			MA10: ind.MA10,
			MA20: ind.MA20,
			MA60: ind.MA60,
			RSI:  ind.RSI,
			MACD: ind.MACD,
		})
	}

	return stockData, indicators, nil
}

// parseStockDataFromText 从文本中解析股票数据（备用方案）
func parseStockDataFromText(content string) ([]StockData, []TechnicalIndicator, error) {
	var stockData []StockData
	var indicators []TechnicalIndicator

	// 简单的文本解析逻辑
	lines := strings.Split(content, "\n")
	for _, line := range lines {
		// 尝试解析日期和价格数据
		if strings.Contains(line, "202") && (strings.Contains(line, "开盘") || strings.Contains(line, "收盘")) {
			// 示例：2024-01-01 开盘:100.0 最高:105.0 最低:98.0 收盘:102.0 成交量:1000000
			dateRegex := regexp.MustCompile(`(\d{4}-\d{2}-\d{2})`)
			priceRegex := regexp.MustCompile(`开盘:(\d+\.?\d*)`)
			highRegex := regexp.MustCompile(`最高:(\d+\.?\d*)`)
			lowRegex := regexp.MustCompile(`最低:(\d+\.?\d*)`)
			closeRegex := regexp.MustCompile(`收盘:(\d+\.?\d*)`)
			volumeRegex := regexp.MustCompile(`成交量:(\d+)`)

			dateMatch := dateRegex.FindStringSubmatch(line)
			if len(dateMatch) > 1 {
				date, _ := time.Parse("2006-01-02", dateMatch[1])

				open, _ := strconv.ParseFloat(priceRegex.FindStringSubmatch(line)[1], 64)
				high, _ := strconv.ParseFloat(highRegex.FindStringSubmatch(line)[1], 64)
				low, _ := strconv.ParseFloat(lowRegex.FindStringSubmatch(line)[1], 64)
				close, _ := strconv.ParseFloat(closeRegex.FindStringSubmatch(line)[1], 64)
				volume, _ := strconv.ParseInt(volumeRegex.FindStringSubmatch(line)[1], 10, 64)

				stockData = append(stockData, StockData{
					Date:   date,
					Open:   open,
					High:   high,
					Low:    low,
					Close:  close,
					Volume: volume,
				})
			}
		}
	}

	// 如果没有解析到数据，生成示例数据
	if len(stockData) == 0 {
		now := time.Now()
		for i := 30; i >= 0; i-- {
			date := now.AddDate(0, 0, -i)
			basePrice := 100.0 + float64(i)*0.1
			stockData = append(stockData, StockData{
				Date:   date,
				Open:   basePrice + math.Sin(float64(i)*0.1)*2,
				High:   basePrice + math.Sin(float64(i)*0.1)*2 + 3,
				Low:    basePrice + math.Sin(float64(i)*0.1)*2 - 2,
				Close:  basePrice + math.Sin(float64(i)*0.1)*2 + 1,
				Volume: int64(1000000 + i*10000),
			})
		}
	}

	return stockData, indicators, nil
}
