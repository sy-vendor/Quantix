package main

import (
	"Quantix/analysis"
	"Quantix/config"
	"bufio"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"
)

func promptForAPIKey() string {
	fmt.Print("请输入 DeepSeek API Key: ")
	reader := bufio.NewReader(os.Stdin)
	key, _ := reader.ReadString('\n')
	return strings.TrimSpace(key)
}

func fetchDeepSeekModels(apiKey string, _ string) ([]string, error) {
	// 直接用 DeepSeek 官方 API
	req, err := http.NewRequest("GET", "https://api.deepseek.com/models", nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+apiKey)
	req.Header.Set("Accept", "application/json")
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("http status: %d", resp.StatusCode)
	}
	var result struct {
		Data []struct {
			ID string `json:"id"`
		} `json:"data"`
	}
	err = json.NewDecoder(resp.Body).Decode(&result)
	if err != nil {
		return nil, err
	}
	models := make([]string, 0, len(result.Data))
	for _, m := range result.Data {
		models = append(models, m.ID)
	}
	return models, nil
}

func promptForModel(models []string) string {
	fmt.Println("可用 DeepSeek 模型列表:")
	for i, m := range models {
		fmt.Printf("[%d] %s\n", i, m)
	}
	fmt.Print("请输入模型编号: ")
	reader := bufio.NewReader(os.Stdin)
	input, _ := reader.ReadString('\n')
	input = strings.TrimSpace(input)
	idx := 0
	fmt.Sscanf(input, "%d", &idx)
	if idx < 0 || idx >= len(models) {
		fmt.Println("输入无效，默认选择第一个模型")
		idx = 0
	}
	return models[idx]
}

func promptForSearchMode() bool {
	fmt.Println("请选择分析模式:")
	fmt.Println("[1] 深度思考（仅用模型推理）")
	fmt.Println("[2] 联网搜索（结合最新互联网信息）")
	fmt.Print("请输入模式编号（默认1）: ")
	reader := bufio.NewReader(os.Stdin)
	input, _ := reader.ReadString('\n')
	input = strings.TrimSpace(input)
	if input == "2" {
		return true // 联网搜索
	}
	return false // 深度思考
}

func promptForPredictionOptions() (periods, dims, outputFormat, searchScope []string, needConfidence bool, riskPref string) {
	reader := bufio.NewReader(os.Stdin)
	fmt.Println("请选择预测周期（可多选，用逗号分隔，默认1周,1月,3月）：")
	fmt.Println("可选：1天,1周,1月,3月,半年,1年")
	fmt.Print("输入: ")
	periodInput, _ := reader.ReadString('\n')
	periodInput = strings.TrimSpace(periodInput)
	if periodInput == "" {
		periods = []string{"1周", "1月", "3月"}
	} else {
		periods = splitAndTrim(periodInput)
	}

	fmt.Println("请选择分析维度（可多选，用逗号分隔，默认技术面,基本面）：")
	fmt.Println("可选：技术面,基本面,资金面,行业对比,情绪分析")
	fmt.Print("输入: ")
	dimInput, _ := reader.ReadString('\n')
	dimInput = strings.TrimSpace(dimInput)
	if dimInput == "" {
		dims = []string{"技术面", "基本面"}
	} else {
		dims = splitAndTrim(dimInput)
	}

	fmt.Println("请选择输出格式（默认结构化表格）：")
	fmt.Println("可选：结构化表格,要点,详细长文,摘要")
	fmt.Print("输入: ")
	outInput, _ := reader.ReadString('\n')
	outInput = strings.TrimSpace(outInput)
	if outInput == "" {
		outputFormat = []string{"结构化表格"}
	} else {
		outputFormat = splitAndTrim(outInput)
	}

	fmt.Print("是否需要置信度/概率说明？(Y/N, 默认Y): ")
	confInput, _ := reader.ReadString('\n')
	confInput = strings.TrimSpace(confInput)
	needConfidence = (confInput == "" || confInput == "Y" || confInput == "y")

	fmt.Println("请选择风险/机会偏好（默认保守）：")
	fmt.Println("可选：保守,激进,风险为主,机会为主")
	fmt.Print("输入: ")
	riskInput, _ := reader.ReadString('\n')
	riskInput = strings.TrimSpace(riskInput)
	if riskInput == "" {
		riskPref = "保守"
	} else {
		riskPref = riskInput
	}

	fmt.Println("请选择联网搜索内容范围（可多选，用逗号分隔，默认新闻,公告）：")
	fmt.Println("可选：新闻,研报,公告,论坛")
	fmt.Print("输入: ")
	scopeInput, _ := reader.ReadString('\n')
	scopeInput = strings.TrimSpace(scopeInput)
	if scopeInput == "" {
		searchScope = []string{"新闻", "公告"}
	} else {
		searchScope = splitAndTrim(scopeInput)
	}

	return
}

func splitAndTrim(s string) []string {
	arr := strings.Split(s, ",")
	for i := range arr {
		arr[i] = strings.TrimSpace(arr[i])
	}
	return arr
}

func main() {
	cfg, err := config.LoadConfig("config.yaml")
	if err != nil {
		panic(err)
	}
	config.GlobalConfig = cfg

	apiKey := promptForAPIKey()
	fmt.Println("正在获取可用 DeepSeek 模型...")
	models, err := fetchDeepSeekModels(apiKey, "")
	deepseekModels := make([]string, 0)
	for _, m := range models {
		if strings.Contains(m, "deepseek") {
			deepseekModels = append(deepseekModels, m)
		}
	}
	model := ""
	if err == nil && len(deepseekModels) > 0 {
		model = promptForModel(deepseekModels)
	} else {
		fmt.Println("未找到可用的 DeepSeek 模型，可手动输入模型名（如 deepseek/deepseek-r1:free）")
		reader := bufio.NewReader(os.Stdin)
		model, _ = reader.ReadString('\n')
		model = strings.TrimSpace(model)
	}

	fmt.Print("请输入股票代码: ")
	reader := bufio.NewReader(os.Stdin)
	stockCode, _ := reader.ReadString('\n')
	stockCode = strings.TrimSpace(stockCode)

	today := time.Now()
	defaultEnd := today.Format("2006-01-02")
	defaultStart := today.AddDate(0, 0, -30).Format("2006-01-02")
	fmt.Printf("请输入开始日期(YYYY-MM-DD, 默认%s): ", defaultStart)
	start, _ := reader.ReadString('\n')
	start = strings.TrimSpace(start)
	if start == "" {
		start = defaultStart
	}
	fmt.Printf("请输入结束日期(YYYY-MM-DD, 默认%s): ", defaultEnd)
	end, _ := reader.ReadString('\n')
	end = strings.TrimSpace(end)
	if end == "" {
		end = defaultEnd
	}

	searchMode := promptForSearchMode()
	periods, dims, outputFormat, searchScope, needConfidence, riskPref := promptForPredictionOptions()

	fmt.Println("\n=== AI 智能分析报告 ===")
	prompt := fmt.Sprintf(
		"请对股票代码 %s 在 %s 到 %s 期间的行情进行详细分析，内容包括：\n"+
			"1. 分析维度：%s\n"+
			"2. 预测周期：%s\n"+
			"3. 输出格式：%s\n"+
			"4. 联网搜索内容范围：%s\n"+
			"5. 风险/机会偏好：%s\n"+
			"6. %s\n"+
			"请结合以上要求，输出结构化、分段、要点或表格内容。",
		stockCode, start, end,
		strings.Join(dims, ", "),
		strings.Join(periods, ", "),
		strings.Join(outputFormat, ", "),
		strings.Join(searchScope, ", "),
		riskPref,
		func() string {
			if needConfidence {
				return "请对每个预测结论给出置信度或概率区间，并简要说明理由。"
			}
			return ""
		}(),
	)
	report, err := analysis.GenerateAIReportWithConfigAndSearch(stockCode, prompt, apiKey, "https://api.deepseek.com/v1/chat/completions", model, searchMode)
	if err != nil {
		fmt.Println("[AI] 生成失败:", err)
	} else {
		fmt.Println(report)
	}
}
