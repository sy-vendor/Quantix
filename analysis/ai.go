package analysis

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"path/filepath"
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
