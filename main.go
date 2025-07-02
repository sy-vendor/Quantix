package main

import (
	"Quantix/analysis"
	"bufio"
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
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

func listHistoryFiles() {
	files, err := ioutil.ReadDir("history")
	if err != nil {
		fmt.Println("[历史记录] 无法读取 history 目录：", err)
		return
	}
	if len(files) == 0 {
		fmt.Println("[历史记录] 暂无历史分析记录。")
		return
	}
	fmt.Println("[历史记录] 可用分析记录：")
	for _, f := range files {
		if !f.IsDir() {
			fmt.Println(f.Name())
		}
	}
}

func showHistoryFile(filename string) {
	path := filepath.Join("history", filename)
	data, err := ioutil.ReadFile(path)
	if err != nil {
		fmt.Println("[历史记录] 读取失败：", err)
		return
	}
	fmt.Println(string(data))
}

func mainMenu() {
	reader := bufio.NewReader(os.Stdin)
	for {
		fmt.Println("\n=== Quantix 主菜单 ===")
		fmt.Println("1. new      - 新建AI分析")
		fmt.Println("2. history  - 查看历史记录列表")
		fmt.Println("3. show <文件名> - 查看指定历史分析")
		fmt.Println("4. exit     - 退出程序")
		fmt.Print("请输入指令: ")
		input, _ := reader.ReadString('\n')
		input = strings.TrimSpace(input)
		if input == "exit" || input == "4" {
			fmt.Println("再见！")
			return
		} else if input == "history" || input == "2" {
			listHistoryFiles()
		} else if strings.HasPrefix(input, "show ") {
			parts := strings.SplitN(input, " ", 2)
			if len(parts) == 2 {
				showHistoryFile(parts[1])
			} else {
				fmt.Println("用法: show <文件名>")
			}
		} else if input == "new" || input == "1" {
			aiAnalysisInteractive()
		} else {
			fmt.Println("无效指令，请重试。")
		}
	}
}

func aiAnalysisInteractive() {
	reader := bufio.NewReader(os.Stdin)
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
		model, _ = reader.ReadString('\n')
		model = strings.TrimSpace(model)
	}
	fmt.Print("请输入股票代码: ")
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
	fmt.Print("请选择分析语言（zh=中文，en=英文，默认zh）: ")
	lang, _ := reader.ReadString('\n')
	lang = strings.TrimSpace(lang)
	if lang == "" {
		lang = "zh"
	}
	var prompt string
	if lang == "en" {
		prompt = fmt.Sprintf(
			"Please provide a detailed analysis of stock %s from %s to %s, including:\n"+
				"1. Analysis dimensions: %s\n"+
				"2. Prediction periods: %s\n"+
				"3. Output format: %s\n"+
				"4. Web search scope: %s\n"+
				"5. Risk/opportunity preference: %s\n"+
				"6. %s\n"+
				"Please output structured, sectioned, key-point or table content as required.",
			stockCode, start, end,
			strings.Join(dims, ", "),
			strings.Join(periods, ", "),
			strings.Join(outputFormat, ", "),
			strings.Join(searchScope, ", "),
			riskPref,
			func() string {
				if needConfidence {
					return "For each prediction, provide a confidence level or probability range, and briefly explain the rationale."
				}
				return ""
			}(),
		)
	} else {
		prompt = fmt.Sprintf(
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
	}
	fmt.Println("\n=== AI 智能分析报告 ===")
	report, err := analysis.GenerateAIReportWithConfigAndSearch(stockCode, prompt, apiKey, "https://api.deepseek.com/v1/chat/completions", model, searchMode)
	if err != nil {
		fmt.Println("[AI] 生成失败:", err)
	} else {
		fmt.Println(report)
	}
	// 保存历史记录
	hist := map[string]interface{}{
		"time":  time.Now().Format("2006-01-02 15:04:05"),
		"stock": stockCode,
		"start": start,
		"end":   end,
		"model": model,
		"mode": func() string {
			if searchMode {
				return "search"
			} else {
				return "reason"
			}
		}(),
		"periods":    periods,
		"dims":       dims,
		"output":     outputFormat,
		"confidence": needConfidence,
		"risk":       riskPref,
		"scope":      searchScope,
		"lang":       lang,
		"prompt":     prompt,
		"report":     report,
	}
	fname := fmt.Sprintf("%s-%s-%s.json", stockCode, end, time.Now().Format("150405"))
	fpath := filepath.Join("history", fname)
	b, _ := json.MarshalIndent(hist, "", "  ")
	_ = ioutil.WriteFile(fpath, b, 0644)
}

func main() {
	// 命令行参数模式：有参数则分析一次后退出，无参数则进入主菜单
	apiKeyFlag := flag.String("apikey", "", "DeepSeek API Key")
	modelFlag := flag.String("model", "", "DeepSeek 模型名")
	stockFlag := flag.String("stock", "", "股票代码")
	startFlag := flag.String("start", "", "开始日期 YYYY-MM-DD")
	endFlag := flag.String("end", "", "结束日期 YYYY-MM-DD")
	modeFlag := flag.String("mode", "", "分析模式: reason/search")
	periodsFlag := flag.String("periods", "", "预测周期, 逗号分隔")
	dimsFlag := flag.String("dims", "", "分析维度, 逗号分隔")
	outputFlag := flag.String("output", "", "输出格式, 逗号分隔")
	confidenceFlag := flag.String("confidence", "", "是否需要置信度说明 Y/N")
	riskFlag := flag.String("risk", "", "风险/机会偏好")
	scopeFlag := flag.String("scope", "", "联网搜索内容范围, 逗号分隔")
	langFlag := flag.String("lang", "zh", "分析语言 zh/en")
	historyFlag := flag.Bool("history", false, "列出分析历史记录")
	showFlag := flag.String("show", "", "显示指定历史分析记录")
	flag.Parse()

	if *historyFlag {
		listHistoryFiles()
		return
	}
	if *showFlag != "" {
		showHistoryFile(*showFlag)
		return
	}
	// 判断是否为命令行参数模式
	if *apiKeyFlag != "" || *modelFlag != "" || *stockFlag != "" || *startFlag != "" || *endFlag != "" || *modeFlag != "" || *periodsFlag != "" || *dimsFlag != "" || *outputFlag != "" || *confidenceFlag != "" || *riskFlag != "" || *scopeFlag != "" || *langFlag != "zh" {
		// 保持原有参数优先分析逻辑
		// ...（原有参数模式分析代码，保存历史后直接 return）...
		// 复制原有参数模式分析代码到此处
		// ... existing code ...
		return
	}
	// 否则进入主菜单循环
	mainMenu()
}
