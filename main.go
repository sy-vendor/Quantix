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
	"strconv"
	"strings"
	"time"

	"github.com/AlecAivazis/survey/v2"
	"github.com/mattn/go-runewidth"
	"golang.org/x/term"
)

var globalAPIKey string // 全局缓存API Key

func promptForAPIKey() string {
	if globalAPIKey != "" {
		fmt.Printf("当前API Key: %s...\n", globalAPIKey[:8])
		fmt.Print("是否更换API Key? (Y/N, 默认N): ")
		reader := bufio.NewReader(os.Stdin)
		input, _ := reader.ReadString('\n')
		input = strings.TrimSpace(input)
		if input != "Y" && input != "y" {
			return globalAPIKey
		}
	}

	fmt.Print("请输入 DeepSeek API Key: ")
	reader := bufio.NewReader(os.Stdin)
	key, _ := reader.ReadString('\n')
	key = strings.TrimSpace(key)
	if key != "" {
		globalAPIKey = key // 缓存API Key
	}
	return key
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
	// 改为上下键选择
	var selected string
	prompt := &survey.Select{
		Message: "请选择要使用的AI模型：",
		Options: models,
	}
	survey.AskOne(prompt, &selected)
	return selected
}

func promptForSearchMode() bool {
	modeOptions := []string{"深度思考（仅用模型推理）", "联网搜索（结合最新互联网信息）"}
	defaultMode := []string{"深度思考（仅用模型推理）"}

	result := interactiveSelectList("请选择分析模式：", modeOptions, defaultMode)
	if len(result) > 0 && result[0] == "联网搜索（结合最新互联网信息）" {
		return true // 联网搜索
	}
	return false // 深度思考
}

// survey多选
func interactiveSelectList(title string, options []string, defaultSelected []string) []string {
	var result []string
	prompt := &survey.MultiSelect{
		Message: title,
		Options: options,
		Default: defaultSelected,
		Help:    "操作说明：↑↓箭头键移动，空格键选择/取消，右箭头键全选，左箭头键全不选，回车键确认",
	}

	err := survey.AskOne(prompt, &result,
		survey.WithHelpInput('?'),
		survey.WithIcons(func(icons *survey.IconSet) {
			icons.SelectFocus.Text = ">"
			icons.MarkedOption.Text = "[✓]"
			icons.UnmarkedOption.Text = "[ ]"
		}),
	)
	if err != nil {
		fmt.Fprintf(os.Stderr, "选择失败: %v\n", err)
		os.Exit(1)
	}
	return result
}

// survey单选
func interactiveSingleSelect(title string, options []string, defaultSelected string) string {
	var result string
	prompt := &survey.Select{
		Message: title,
		Options: options,
		Default: defaultSelected,
		Help:    "操作说明：↑↓箭头键移动选择，回车键确认",
	}
	err := survey.AskOne(prompt, &result, survey.WithHelpInput('?'), survey.WithIcons(func(icons *survey.IconSet) {
		icons.SelectFocus.Text = ">"
	}))
	if err != nil {
		fmt.Fprintf(os.Stderr, "选择失败: %v\n", err)
		os.Exit(1)
	}
	return result
}

// survey输入
func interactiveInput(title, defaultValue string) string {
	var result string
	prompt := &survey.Input{
		Message: title,
		Default: defaultValue,
		Help:    "操作说明：直接输入内容，回车键确认",
	}
	err := survey.AskOne(prompt, &result, survey.WithHelpInput('?'))
	if err != nil {
		fmt.Fprintf(os.Stderr, "输入失败: %v\n", err)
		os.Exit(1)
	}
	return result
}

// survey浮点数输入
func interactiveInputFloat(title string, defaultValue float64) float64 {
	var result string
	prompt := &survey.Input{
		Message: title,
		Default: fmt.Sprintf("%g", defaultValue),
		Help:    "直接输入数字，回车确认",
	}
	err := survey.AskOne(prompt, &result, survey.WithHelpInput('?'))
	if err != nil {
		fmt.Fprintf(os.Stderr, "输入失败: %v\n", err)
		os.Exit(1)
	}
	f, err := strconv.ParseFloat(strings.TrimSpace(result), 64)
	if err != nil {
		return defaultValue
	}
	return f
}

// survey确认输入
func interactiveConfirm(title string, defaultValue bool) bool {
	var result bool
	prompt := &survey.Confirm{
		Message: title,
		Default: defaultValue,
	}
	err := survey.AskOne(prompt, &result)
	if err != nil {
		fmt.Fprintf(os.Stderr, "输入失败: %v\n", err)
		os.Exit(1)
	}
	return result
}

func promptForPredictionOptions() (periods, dims, searchScope []string, outputFormat string, needConfidence bool, riskPref string) {
	// 预测周期多选
	periodOptions := []string{"1天", "1周", "1月", "3月", "半年", "1年"}
	defaultPeriods := []string{"1周", "1月", "3月"}
	periods = interactiveSelectList("请选择预测周期（可多选）：", periodOptions, defaultPeriods)

	// 分析维度多选
	dimOptions := []string{"技术面", "基本面", "资金面", "行业对比", "情绪分析"}
	defaultDims := []string{"技术面", "基本面"}
	dims = interactiveSelectList("请选择分析维度（可多选）：", dimOptions, defaultDims)

	// 输出格式单选
	outputOptions := []string{"结构化表格", "要点", "详细长文", "摘要"}
	outputFormat = interactiveSingleSelect("请选择输出格式：", outputOptions, outputOptions[0])

	// 置信度单选
	confOptions := []string{"需要置信度/概率说明", "不需要置信度/概率说明"}
	needConfidence = interactiveSingleSelect("是否需要置信度/概率说明？", confOptions, confOptions[0]) == confOptions[0]

	// 风险偏好单选
	riskOptions := []string{"保守", "激进", "风险为主", "机会为主"}
	riskPref = interactiveSingleSelect("请选择风险/机会偏好：", riskOptions, riskOptions[0])

	// 联网搜索范围多选
	scopeOptions := []string{"新闻", "研报", "公告", "论坛"}
	defaultScope := []string{"新闻", "公告"}
	searchScope = interactiveSelectList("请选择联网搜索内容范围（可多选）：", scopeOptions, defaultScope)

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
	// 使用survey.Select替代手动输入
	for {
		menuOptions := []string{
			"新建AI分析（支持批量，股票代码用逗号分隔）",
			"定时任务（自动定时分析/推送）",
			"查看历史记录列表",
			"查看指定历史分析",
			"更换DeepSeek API Key",
			"退出程序",
		}
		title := centerText("=== Quantix 智能股票分析系统 ===", 50)
		msg := title + "\n单选：使用 ↑↓ 箭头键移动选择，回车键确认\n多选：使用 ↑↓ 箭头键移动，空格键选择/取消，右箭头键全选，左箭头键全不选，回车键确认\n按 ? 键可查看详细操作说明\n\n请选择操作："
		var selected string
		prompt := &survey.Select{
			Message: msg,
			Options: menuOptions,
		}
		survey.AskOne(prompt, &selected)

		switch selected {
		case menuOptions[0]:
			aiAnalysisInteractiveMenu()
		case menuOptions[1]:
			aiScheduleInteractiveMenu()
		case menuOptions[2]:
			listHistoryFiles()
		case menuOptions[3]:
			var filename string
			_ = survey.AskOne(&survey.Input{Message: "请输入文件名:"}, &filename)
			if filename != "" {
				showHistoryFile(filename)
			}
		case menuOptions[4]:
			globalAPIKey = ""
			fmt.Println("API Key已重置，下次分析时将重新输入")
		case menuOptions[5]:
			fmt.Println("再见！")
			return
		}
	}
}

// 居中辅助函数
func centerText(s string, width int) string {
	if len([]rune(s)) >= width {
		return s
	}
	pad := (width - len([]rune(s))) / 2
	return strings.Repeat(" ", pad) + s
}

func showAnalyzingAnimation(done chan struct{}) {
	dots := 1
	for {
		select {
		case <-done:
			fmt.Print("\r                \r") // 清除动画行
			return
		default:
			fmt.Printf("\r分析中%s", strings.Repeat(".", dots))
			dots = dots%3 + 1
			time.Sleep(500 * time.Millisecond)
		}
	}
}

func buildPromptWithDetail(params analysis.AnalysisParams, detail string) string {
	basePrompt := analysis.BuildPrompt(params)
	if detail == "extreme" {
		return basePrompt + "\n请将每个分析维度细分到最小颗粒度，涵盖K线形态、均线系统、成交量、技术指标、支撑阻力、财务数据、盈利能力、估值、行业地位、管理层、分红、主力资金、北向资金、大宗交易、行业对比、情绪分析（新闻、公告、研报、论坛、社交媒体）、多周期预测（1天、1周、1月、3月），每项都要详细说明，所有结论都要有数据和理由支撑，输出结构化表格+要点+详细长文，适合专业投资者参考。"
	}
	if detail == "detailed" {
		return basePrompt + "\n请对每个分析维度进行细致展开，涵盖技术面、基本面、资金面、行业对比、情绪分析的各个子项，给出多周期预测、操作建议、风险与机会，所有结论都要有理由和数据支撑。"
	}
	return basePrompt
}

func getBoxWidth() int {
	w, _, err := term.GetSize(int(os.Stdout.Fd()))
	if err != nil || w <= 0 {
		return 120 // fallback
	}
	bw := int(float64(w) * 0.98)
	if bw < 60 {
		bw = 60
	}
	return bw
}

func printStepBox(title string, lines ...string) {
	width := getBoxWidth()
	titleWidth := runewidth.StringWidth(title)
	sideLen := (width - 2 - titleWidth) / 2
	top := "┌" + strings.Repeat("─", sideLen) + title + strings.Repeat("─", width-2-titleWidth-sideLen) + "┐"
	fmt.Println(top)
	for _, l := range lines {
		maxContent := width - 2
		lWidth := runewidth.StringWidth(l)
		if lWidth > maxContent {
			l = runewidth.Truncate(l, maxContent-1, "…")
			lWidth = runewidth.StringWidth(l)
		}
		pad := maxContent - lWidth
		if pad < 0 {
			pad = 0
		}
		fmt.Printf("│%s%s│\n", l, strings.Repeat(" ", pad))
	}
	fmt.Println("└" + strings.Repeat("─", width-2) + "┘")
}

func aiAnalysisInteractiveMenu() {
	reader := bufio.NewReader(os.Stdin)

	fmt.Println("\n================= AI 智能分析配置 =================")
	fmt.Println("单选：使用 ↑↓ 箭头键移动选择，回车键确认")
	fmt.Println("多选：使用 ↑↓ 箭头键移动，空格键选择/取消，右箭头键全选，左箭头键全不选，回车键确认")
	fmt.Println("按 ? 键可查看详细操作说明")
	fmt.Println("==================================================\n")

	// Step 0: API Key
	printStepBox("Step 0: API Key",
		"请输入 DeepSeek API Key",
		"说明：用于访问 DeepSeek LLM 服务",
	)
	apiKey := promptForAPIKey()
	printStepBox("Step 0: API Key", fmt.Sprintf("[当前API Key]: %s...", func() string {
		if len(apiKey) > 8 {
			return apiKey[:8]
		} else {
			return apiKey
		}
	}()))

	// Step 1: 选择模型
	printStepBox("Step 1: AI Model",
		"选择要使用的AI模型",
		"说明：不同模型分析能力和速度略有差异",
		"Default: 自动推荐 DeepSeek 模型",
		"正在获取可用 DeepSeek 模型...",
	)
	models, err := fetchDeepSeekModels(apiKey, "")
	deepseekModels := make([]string, 0)
	for _, m := range models {
		if strings.Contains(m, "deepseek") {
			deepseekModels = append(deepseekModels, m)
		}
	}
	var model string
	if err == nil && len(deepseekModels) > 0 {
		model = promptForModel(deepseekModels)
	} else {
		fmt.Println("未找到可用的 DeepSeek 模型，请手动输入模型名（如 deepseek/deepseek-r1:free）")
		model, _ = reader.ReadString('\n')
		model = strings.TrimSpace(model)
	}
	printStepBox("Step 1: AI Model", fmt.Sprintf("[当前选择]: %s", model))

	// Step 2: 股票代码
	printStepBox("Step 2: Ticker Symbol",
		"Enter the ticker symbol(s) to analyze",
		"说明：可批量，逗号分隔。如 600036,000001",
		"Default: 600036",
	)
	stockInput := interactiveInput("请输入股票代码（可批量，逗号分隔）:", "")
	stockCodes := splitAndTrim(stockInput)
	if len(stockCodes) == 0 || stockCodes[0] == "" {
		fmt.Println("股票代码不能为空！")
		return
	}
	printStepBox("Step 2: Ticker Symbol", fmt.Sprintf("[当前选择]: %s", strings.Join(stockCodes, ", ")))

	// Step 3: 日期
	today := time.Now()
	defaultEnd := today.Format("2006-01-02")
	defaultStart := today.AddDate(0, 0, -30).Format("2006-01-02")
	printStepBox("Step 3: Analysis Date",
		"Enter the analysis date range (YYYY-MM-DD)",
		fmt.Sprintf("Default: %s ~ %s", defaultStart, defaultEnd),
	)
	start := interactiveInput(fmt.Sprintf("请输入开始日期(YYYY-MM-DD, 默认%s):", defaultStart), defaultStart)
	end := interactiveInput(fmt.Sprintf("请输入结束日期(YYYY-MM-DD, 默认%s):", defaultEnd), defaultEnd)
	printStepBox("Step 3: Analysis Date", fmt.Sprintf("[当前选择]: %s ~ %s", start, end))

	// Step 4: 分析模式
	printStepBox("Step 4: Analysis Mode",
		"Select your analysis mode",
		"说明：深度思考仅用模型推理，联网搜索结合互联网信息，混合模式自动融合",
	)
	modeOptions := []string{"深度思考（仅用模型推理）", "联网搜索（结合最新互联网信息）", "深度思考+联网搜索（自动融合）"}
	defaultMode := []string{"深度思考（仅用模型推理）"}
	searchModes := interactiveSelectList("请选择分析模式（可多选）：", modeOptions, defaultMode)
	printStepBox("Step 4: Analysis Mode", fmt.Sprintf("[当前选择]: %s", strings.Join(searchModes, ", ")))

	// Step 5: 预测参数
	printStepBox("Step 5: Prediction Options",
		"选择预测周期、分析维度、输出格式等",
		"说明：可多选，回车确认",
	)
	periods, dims, searchScope, outputFormat, needConfidence, riskPref := promptForPredictionOptions()
	printStepBox("Step 5: Prediction Options",
		fmt.Sprintf("[周期]: %s", strings.Join(periods, ", ")),
		fmt.Sprintf("[维度]: %s", strings.Join(dims, ", ")),
		fmt.Sprintf("[输出]: %s", outputFormat),
		fmt.Sprintf("[置信度]: %v", needConfidence),
		fmt.Sprintf("[风险偏好]: %s", riskPref),
		fmt.Sprintf("[联网范围]: %s", strings.Join(searchScope, ", ")),
	)

	// Step 6: 语言选择
	printStepBox("Step 6: Language",
		"Select analysis language",
		"说明：支持中文和英文",
	)
	langOptions := []string{"中文", "英文"}
	defaultLang := []string{"中文"}
	langResult := interactiveSelectList("请选择分析语言：", langOptions, defaultLang)
	var lang string
	if len(langResult) > 0 && langResult[0] == "英文" {
		lang = "en"
	} else {
		lang = "zh"
	}
	printStepBox("Step 6: Language", fmt.Sprintf("[当前选择]: %s", lang))

	// Step 7: 导出格式
	printStepBox("Step 7: Export Format",
		"Select export format(s)",
		"说明：可多选，支持 Markdown/HTML/PDF",
	)
	exportOptions := []string{"Markdown", "HTML", "PDF"}
	defaultExport := []string{"Markdown"}
	exportResult := interactiveSelectList("请选择导出格式（可多选）：", exportOptions, defaultExport)
	exportFormats := make([]string, 0, len(exportResult))
	for _, fmtx := range exportResult {
		switch fmtx {
		case "Markdown":
			exportFormats = append(exportFormats, "md")
		case "HTML":
			exportFormats = append(exportFormats, "html")
		case "PDF":
			exportFormats = append(exportFormats, "pdf")
		}
	}
	if len(exportFormats) == 0 {
		exportFormats = []string{"md"}
	}
	printStepBox("Step 7: Export Format", fmt.Sprintf("[当前选择]: %s", strings.Join(exportFormats, ", ")))

	// Step 8: 邮件推送
	printStepBox("Step 8: Email Push",
		"如需邮件推送请输入收件人邮箱（可逗号分隔，留空跳过）",
	)
	emailInput := interactiveInput("如需邮件推送请输入收件人邮箱（可逗号分隔，留空跳过）:", "")
	emails := splitAndTrim(emailInput)
	var smtpServer, smtpUser, smtpPass string
	if len(emails) > 0 && emails[0] != "" {
		fmt.Println("SMTP服务器、端口、用户名、密码依次输入：")
		fmt.Print("SMTP服务器: ")
		smtpServer, _ = reader.ReadString('\n')
		smtpServer = strings.TrimSpace(smtpServer)
		fmt.Print("SMTP端口(默认465): ")
		portInput := interactiveInput("SMTP端口(默认465):", "")
		if portInput == "" {
			// smtpPort = 465
		}
		fmt.Print("SMTP用户名: ")
		smtpUser, _ = reader.ReadString('\n')
		smtpUser = strings.TrimSpace(smtpUser)
		fmt.Print("SMTP密码: ")
		smtpPass, _ = reader.ReadString('\n')
		smtpPass = strings.TrimSpace(smtpPass)
	}
	printStepBox("Step 8: Email Push", fmt.Sprintf("[当前邮箱]: %s", strings.Join(emails, ", ")))

	// Step 9: IM 推送
	printStepBox("Step 9: IM Push",
		"如需IM推送请输入Webhook地址（钉钉/企业微信，留空跳过）",
	)
	webhook := interactiveInput("如需IM推送请输入Webhook地址（钉钉/企业微信，留空跳过）:", "")
	printStepBox("Step 9: IM Push", fmt.Sprintf("[当前Webhook]: %s", webhook))

	// Step 10: 分析详细程度
	printStepBox("Step 10: Research Depth",
		"Select your research depth level",
		"说明：影响分析细致程度和报告内容",
	)
	detailOptions := []string{"普通分析 - 基础技术指标和简要分析", "详细分析 - 多维度深度分析，包含更多指标", "极致分析 - 最全面的分析，包含所有可用指标和深度洞察"}
	defaultDetail := []string{"普通分析 - 基础技术指标和简要分析"}
	detailResult := interactiveSelectList("请选择分析详细程度：", detailOptions, defaultDetail)
	var detailInput string
	var detailText string
	if len(detailResult) > 0 {
		switch detailResult[0] {
		case "普通分析 - 基础技术指标和简要分析":
			detailInput = "normal"
		case "详细分析 - 多维度深度分析，包含更多指标":
			detailInput = "detailed"
		case "极致分析 - 最全面的分析，包含所有可用指标和深度洞察":
			detailInput = "extreme"
		default:
			detailInput = "normal"
		}
		detailText = detailResult[0]
	} else {
		detailInput = "normal"
		detailText = "普通分析 - 基础技术指标和简要分析"
	}
	printStepBox("Step 10: Research Depth", fmt.Sprintf("[当前选择]: %s", detailText))

	params := analysis.AnalysisParams{
		APIKey:     apiKey,
		Model:      model,
		StockCodes: stockCodes,
		Start:      start,
		End:        end,
		Periods:    periods,
		Dims:       dims,
		Output:     exportFormats,
		Confidence: needConfidence,
		Risk:       riskPref,
		Scope:      searchScope,
		Lang:       lang,
	}

	fmt.Println("\n=== 开始AI智能分析 ===")
	fmt.Printf("分析股票：%s\n", strings.Join(stockCodes, ", "))
	fmt.Printf("分析期间：%s 至 %s\n", start, end)
	fmt.Printf("分析模式：%s\n", func() string {
		if searchModes[0] == "深度思考（仅用模型推理）" {
			return "深度思考模式"
		}
		return "深度思考+联网搜索模式"
	}())
	fmt.Println("正在生成分析报告，请稍候...")

	prompt := buildPromptWithDetail(params, detailInput)
	done := make(chan struct{})
	go showAnalyzingAnimation(done)
	results := make([]analysis.AnalysisResult, 0, len(params.StockCodes)*len(searchModes))
	for _, mode := range searchModes {
		for _, code := range params.StockCodes {
			p := params
			p.StockCodes = []string{code}
			p.SearchMode = (mode == "联网搜索（结合最新互联网信息）")
			p.HybridSearch = (mode == "深度思考+联网搜索（自动融合）")
			result := analysis.AnalyzeOne(p, func(stock, _prompt, apiKey, apiURL, model string, searchMode bool, hybridSearch bool) (string, error) {
				return analysis.GenerateAIReportWithConfigAndSearch(stock, prompt, apiKey, "https://api.deepseek.com/v1/chat/completions", model, searchMode, hybridSearch)
			})
			results = append(results, result)
		}
	}
	for _, r := range results {
		fmt.Printf("\n=== [%s] AI 智能分析报告 ===\n", r.StockCode)
		if r.Err != nil {
			fmt.Println("[AI] 生成失败:", r.Err)
		} else {
			// 分离图片引用和正文
			reportLines := strings.Split(r.Report, "\n")
			var imgLines, textLines []string
			for _, l := range reportLines {
				if strings.HasPrefix(l, "![图表](") {
					imgLines = append(imgLines, l)
				} else if strings.TrimSpace(l) != "" {
					textLines = append(textLines, l)
				}
			}
			// 先输出图片引用
			for _, l := range imgLines {
				fmt.Println(l)
			}
			// 用框输出正文
			if len(textLines) > 0 {
				printStepBox("AI 智能分析报告", textLines...)
			}
			fmt.Printf("[历史已保存: %s]\n", r.SavedFile)

			// 邮件推送
			if len(emails) > 0 && emails[0] != "" && smtpServer != "" && smtpUser != "" && smtpPass != "" {
				var attachs []string
				for _, fmtx := range exportFormats {
					if fmtx == "html" {
						attachs = append(attachs, "history/"+r.SavedFile[:len(r.SavedFile)-5]+"."+fmtx)
					}
				}
				err := analysis.SendEmail(smtpServer, 465, smtpUser, smtpPass, emails, "Quantix分析报告", r.Report, attachs)
				if err != nil {
					fmt.Println("[邮件发送失败]", err)
				} else {
					fmt.Println("[邮件已发送]")
				}
			}
			// IM推送
			if webhook != "" {
				err := analysis.SendWebhook(webhook, r.Report)
				if err != nil {
					fmt.Println("[IM推送失败]", err)
				} else {
					fmt.Println("[IM已推送]")
				}
			}
		}
	}
	close(done)
}

func parseSchedule(s string) (time.Duration, error) {
	s = strings.ToLower(strings.TrimSpace(s))
	if s == "daily" {
		now := time.Now()
		next := now.AddDate(0, 0, 1).Truncate(24 * time.Hour)
		return next.Sub(now), nil
	}
	if strings.HasSuffix(s, "h") {
		h, err := strconv.Atoi(strings.TrimSuffix(s, "h"))
		if err != nil || h <= 0 {
			return 0, fmt.Errorf("无效的小时数")
		}
		return time.Duration(h) * time.Hour, nil
	}
	if strings.HasSuffix(s, "m") {
		m, err := strconv.Atoi(strings.TrimSuffix(s, "m"))
		if err != nil || m <= 0 {
			return 0, fmt.Errorf("无效的分钟数")
		}
		return time.Duration(m) * time.Minute, nil
	}
	return 0, fmt.Errorf("不支持的定时格式，仅支持 10m、1h、daily 等")
}

// 定时任务交互式菜单
func aiScheduleInteractiveMenu() {
	reader := bufio.NewReader(os.Stdin)
	fmt.Println("\n================= 定时任务配置 =================")
	fmt.Println("本功能支持自动定时分析、推送，无需人工值守。Ctrl+C 可随时终止。\n")

	// 复用 aiAnalysisInteractiveMenu 的参数交互
	// Step 0: API Key
	printStepBox("Step 0: API Key",
		"请输入 DeepSeek API Key",
		"说明：用于访问 DeepSeek LLM 服务",
	)
	apiKey := promptForAPIKey()
	printStepBox("Step 0: API Key", fmt.Sprintf("[当前API Key]: %s...", func() string {
		if len(apiKey) > 8 {
			return apiKey[:8]
		} else {
			return apiKey
		}
	}()))

	// Step 1: 选择模型
	printStepBox("Step 1: AI Model",
		"选择要使用的AI模型",
		"说明：不同模型分析能力和速度略有差异",
		"Default: 自动推荐 DeepSeek 模型",
		"正在获取可用 DeepSeek 模型...",
	)
	models, err := fetchDeepSeekModels(apiKey, "")
	deepseekModels := make([]string, 0)
	for _, m := range models {
		if strings.Contains(m, "deepseek") {
			deepseekModels = append(deepseekModels, m)
		}
	}
	var model string
	if err == nil && len(deepseekModels) > 0 {
		model = promptForModel(deepseekModels)
	} else {
		fmt.Println("未找到可用的 DeepSeek 模型，请手动输入模型名（如 deepseek/deepseek-r1:free）")
		model, _ = reader.ReadString('\n')
		model = strings.TrimSpace(model)
	}
	printStepBox("Step 1: AI Model", fmt.Sprintf("[当前选择]: %s", model))

	// Step 2: 股票代码
	printStepBox("Step 2: Ticker Symbol",
		"Enter the ticker symbol(s) to analyze",
		"说明：可批量，逗号分隔。如 600036,000001",
		"Default: 600036",
	)
	stockInput := interactiveInput("请输入股票代码（可批量，逗号分隔）:", "")
	stockCodes := splitAndTrim(stockInput)
	if len(stockCodes) == 0 || stockCodes[0] == "" {
		fmt.Println("股票代码不能为空！")
		return
	}
	printStepBox("Step 2: Ticker Symbol", fmt.Sprintf("[当前选择]: %s", strings.Join(stockCodes, ", ")))

	// Step 3: 日期
	today := time.Now()
	defaultEnd := today.Format("2006-01-02")
	defaultStart := today.AddDate(0, 0, -30).Format("2006-01-02")
	printStepBox("Step 3: Analysis Date",
		"Enter the analysis date range (YYYY-MM-DD)",
		fmt.Sprintf("Default: %s ~ %s", defaultStart, defaultEnd),
	)
	start := interactiveInput(fmt.Sprintf("请输入开始日期(YYYY-MM-DD, 默认%s):", defaultStart), defaultStart)
	end := interactiveInput(fmt.Sprintf("请输入结束日期(YYYY-MM-DD, 默认%s):", defaultEnd), defaultEnd)
	printStepBox("Step 3: Analysis Date", fmt.Sprintf("[当前选择]: %s ~ %s", start, end))

	// Step 4: 分析模式
	printStepBox("Step 4: Analysis Mode",
		"Select your analysis mode",
		"说明：深度思考仅用模型推理，联网搜索结合互联网信息，混合模式自动融合",
	)
	modeOptions := []string{"深度思考（仅用模型推理）", "联网搜索（结合最新互联网信息）", "深度思考+联网搜索（自动融合）"}
	defaultMode := []string{"深度思考（仅用模型推理）"}
	searchModes := interactiveSelectList("请选择分析模式（可多选）：", modeOptions, defaultMode)
	printStepBox("Step 4: Analysis Mode", fmt.Sprintf("[当前选择]: %s", strings.Join(searchModes, ", ")))

	// Step 5: 预测参数
	printStepBox("Step 5: Prediction Options",
		"选择预测周期、分析维度、输出格式等",
		"说明：可多选，回车确认",
	)
	periods, dims, searchScope, outputFormat, needConfidence, riskPref := promptForPredictionOptions()
	printStepBox("Step 5: Prediction Options",
		fmt.Sprintf("[周期]: %s", strings.Join(periods, ", ")),
		fmt.Sprintf("[维度]: %s", strings.Join(dims, ", ")),
		fmt.Sprintf("[输出]: %s", outputFormat),
		fmt.Sprintf("[置信度]: %v", needConfidence),
		fmt.Sprintf("[风险偏好]: %s", riskPref),
		fmt.Sprintf("[联网范围]: %s", strings.Join(searchScope, ", ")),
	)

	// Step 6: 语言选择
	printStepBox("Step 6: Language",
		"Select analysis language",
		"说明：支持中文和英文",
	)
	langOptions := []string{"中文", "英文"}
	defaultLang := []string{"中文"}
	langResult := interactiveSelectList("请选择分析语言：", langOptions, defaultLang)
	var lang string
	if len(langResult) > 0 && langResult[0] == "英文" {
		lang = "en"
	} else {
		lang = "zh"
	}
	printStepBox("Step 6: Language", fmt.Sprintf("[当前选择]: %s", lang))

	// Step 7: 导出格式
	printStepBox("Step 7: Export Format",
		"Select export format(s)",
		"说明：可多选，支持 Markdown/HTML/PDF",
	)
	exportOptions := []string{"Markdown", "HTML", "PDF"}
	defaultExport := []string{"Markdown"}
	exportResult := interactiveSelectList("请选择导出格式（可多选）：", exportOptions, defaultExport)
	exportFormats := make([]string, 0, len(exportResult))
	for _, fmtx := range exportResult {
		switch fmtx {
		case "Markdown":
			exportFormats = append(exportFormats, "md")
		case "HTML":
			exportFormats = append(exportFormats, "html")
		case "PDF":
			exportFormats = append(exportFormats, "pdf")
		}
	}
	if len(exportFormats) == 0 {
		exportFormats = []string{"md"}
	}
	printStepBox("Step 7: Export Format", fmt.Sprintf("[当前选择]: %s", strings.Join(exportFormats, ", ")))

	// Step 8: 邮件推送
	printStepBox("Step 8: Email Push",
		"如需邮件推送请输入收件人邮箱（可逗号分隔，留空跳过）",
	)
	emailInput := interactiveInput("如需邮件推送请输入收件人邮箱（可逗号分隔，留空跳过）:", "")
	emails := splitAndTrim(emailInput)
	var smtpServer, smtpUser, smtpPass string
	if len(emails) > 0 && emails[0] != "" {
		fmt.Println("SMTP服务器、端口、用户名、密码依次输入：")
		fmt.Print("SMTP服务器: ")
		smtpServer, _ = reader.ReadString('\n')
		smtpServer = strings.TrimSpace(smtpServer)
		fmt.Print("SMTP端口(默认465): ")
		portInput := interactiveInput("SMTP端口(默认465):", "")
		if portInput == "" {
			// smtpPort = 465
		}
		fmt.Print("SMTP用户名: ")
		smtpUser, _ = reader.ReadString('\n')
		smtpUser = strings.TrimSpace(smtpUser)
		fmt.Print("SMTP密码: ")
		smtpPass, _ = reader.ReadString('\n')
		smtpPass = strings.TrimSpace(smtpPass)
	}
	printStepBox("Step 8: Email Push", fmt.Sprintf("[当前邮箱]: %s", strings.Join(emails, ", ")))

	// Step 9: IM 推送
	printStepBox("Step 9: IM Push",
		"如需IM推送请输入Webhook地址（钉钉/企业微信，留空跳过）",
	)
	webhook := interactiveInput("如需IM推送请输入Webhook地址（钉钉/企业微信，留空跳过）:", "")
	printStepBox("Step 9: IM Push", fmt.Sprintf("[当前Webhook]: %s", webhook))

	// Step 10: 分析详细程度
	printStepBox("Step 10: Research Depth",
		"Select your research depth level",
		"说明：影响分析细致程度和报告内容",
	)
	detailOptions := []string{"普通分析 - 基础技术指标和简要分析", "详细分析 - 多维度深度分析，包含更多指标", "极致分析 - 最全面的分析，包含所有可用指标和深度洞察"}
	defaultDetail := []string{"普通分析 - 基础技术指标和简要分析"}
	detailResult := interactiveSelectList("请选择分析详细程度：", detailOptions, defaultDetail)
	var detailInput string
	var detailText string
	if len(detailResult) > 0 {
		switch detailResult[0] {
		case "普通分析 - 基础技术指标和简要分析":
			detailInput = "normal"
		case "详细分析 - 多维度深度分析，包含更多指标":
			detailInput = "detailed"
		case "极致分析 - 最全面的分析，包含所有可用指标和深度洞察":
			detailInput = "extreme"
		default:
			detailInput = "normal"
		}
		detailText = detailResult[0]
	} else {
		detailInput = "normal"
		detailText = "普通分析 - 基础技术指标和简要分析"
	}
	printStepBox("Step 10: Research Depth", fmt.Sprintf("[当前选择]: %s", detailText))

	// Step 12: 定时周期
	printStepBox("Step 12: Schedule",
		"请输入定时任务周期，如 10m、1h、daily（分钟/小时/每天）",
		"示例：10m 表示每10分钟，1h 表示每小时，daily 表示每天0点",
	)
	schedule := interactiveInput("请输入定时任务周期（如10m/1h/daily）:", "1h")
	dur, err := parseSchedule(schedule)
	if err != nil {
		fmt.Println("[定时任务] 周期格式错误：", err)
		return
	}
	printStepBox("Step 12: Schedule", fmt.Sprintf("[当前周期]: %s", schedule))

	params := analysis.AnalysisParams{
		APIKey:     apiKey,
		Model:      model,
		StockCodes: stockCodes,
		Start:      start,
		End:        end,
		Periods:    periods,
		Dims:       dims,
		Output:     exportFormats,
		Confidence: needConfidence,
		Risk:       riskPref,
		Scope:      searchScope,
		Lang:       lang,
	}

	fmt.Println("\n=== 定时任务已启动，Ctrl+C 可随时终止 ===")
	for {
		fmt.Printf("\n[%s] 批量分析开始\n", time.Now().Format("2006-01-02 15:04:05"))
		done := make(chan struct{})
		go showAnalyzingAnimation(done)
		prompt := buildPromptWithDetail(params, detailInput)

		results := make([]analysis.AnalysisResult, 0, len(params.StockCodes)*len(searchModes))
		for _, mode := range searchModes {
			for _, code := range params.StockCodes {
				p := params
				p.StockCodes = []string{code}
				p.SearchMode = (mode == "联网搜索（结合最新互联网信息）")
				p.HybridSearch = (mode == "深度思考+联网搜索（自动融合）")
				result := analysis.AnalyzeOne(p, func(stock, _prompt, apiKey, apiURL, model string, searchMode bool, hybridSearch bool) (string, error) {
					return analysis.GenerateAIReportWithConfigAndSearch(stock, prompt, apiKey, "https://api.deepseek.com/v1/chat/completions", model, searchMode, hybridSearch)
				})
				results = append(results, result)
			}
		}
		for _, r := range results {
			fmt.Printf("\n=== [%s] AI 智能分析报告 ===\n", r.StockCode)
			if r.Err != nil {
				fmt.Println("[AI] 生成失败:", r.Err)
			} else {
				// 分离图片引用和正文
				reportLines := strings.Split(r.Report, "\n")
				var imgLines, textLines []string
				for _, l := range reportLines {
					if strings.HasPrefix(l, "![图表](") {
						imgLines = append(imgLines, l)
					} else if strings.TrimSpace(l) != "" {
						textLines = append(textLines, l)
					}
				}
				// 先输出图片引用
				for _, l := range imgLines {
					fmt.Println(l)
				}
				// 用框输出正文
				if len(textLines) > 0 {
					printStepBox("AI 智能分析报告", textLines...)
				}
				fmt.Printf("[历史已保存: %s]\n", r.SavedFile)
			}
		}
		close(done)
		fmt.Printf("[定时任务] 下一次将在 %s 后运行，Ctrl+C 可终止。\n", dur)
		time.Sleep(dur)
	}
}

func main() {
	survey.MultiSelectQuestionTemplate = `
{{- define "option"}}
    {{- if eq $.SelectedIndex $.CurrentIndex }}{{color "cyan"}}> {{if index $.Checked $.CurrentOpt.Index }}[✓]{{else}}[ ]{{end}} {{$.CurrentOpt.Value}}{{color "reset"}}{{else}}  {{if index $.Checked $.CurrentOpt.Index }}[✓]{{else}}[ ]{{end}} {{$.CurrentOpt.Value}}{{end}}
{{"\n"}}
{{- end}}
{{- if .ShowHelp }}{{color "cyan"}}{{ .Help }}{{color "reset"}}{{"\n"}}{{end}}
{{- color "cyan"}}{{ .Message }}{{ .FilterMessage }}
{{- if .ShowAnswer}}{{color "cyan"}} {{.Answer}}{{color "reset"}}{{"\n"}}
{{- else }}
  {{- "  "}}{{- color "cyan"}}[使用箭头键移动，空格键选择，右箭头键全选，左箭头键全不选，输入过滤，? 查看帮助]{{color "reset"}}
  {{- "\n"}}
  {{- range $ix, $option := .PageEntries}}
    {{- template "option" ($.IterateOption $ix $option) }}
  {{- end}}
{{- end}}`

	survey.SelectQuestionTemplate = `
{{- if .ShowHelp }}{{color "cyan"}}{{ .Help }}{{color "reset"}}{{"\n"}}{{end}}
{{- color "cyan"}}{{ .Message }}{{ .FilterMessage }}
{{- if .ShowAnswer}}{{color "cyan"}} {{.Answer}}{{color "reset"}}{{"\n"}}
{{- else }}
  {{- "  "}}{{- color "cyan"}}[使用箭头键移动，回车键确认，? 查看帮助]{{color "reset"}}
  {{- "\n"}}
  {{- range $ix, $option := .PageEntries}}
    {{- if eq $.SelectedIndex $ix }}{{color "cyan"}}> {{ $option.Value }}{{color "reset"}}{{else}}  {{ $option.Value }}{{end}}
    {{- "\n"}}
  {{- end}}
{{- end}}`

	// 命令行参数模式：有参数则分析一次后退出，无参数则进入主菜单
	apiKeyFlag := flag.String("apikey", "", "DeepSeek API Key")
	modelFlag := flag.String("model", "", "DeepSeek 模型名")
	stockFlag := flag.String("stock", "", "股票代码（可批量，逗号分隔）")
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
	scheduleFlag := flag.String("schedule", "", "定时任务周期，如 1h、daily（预留）")
	exportFlag := flag.String("export", "md", "导出格式，逗号分隔，支持md,html,pdf")
	emailFlag := flag.String("email", "", "收件人邮箱，逗号分隔")
	smtpServerFlag := flag.String("smtp-server", "", "SMTP服务器")
	smtpPortFlag := flag.Int("smtp-port", 465, "SMTP端口")
	smtpUserFlag := flag.String("smtp-user", "", "SMTP用户名")
	smtpPassFlag := flag.String("smtp-pass", "", "SMTP密码")
	webhookFlag := flag.String("webhook", "", "IM webhook地址")
	detailFlag := flag.String("detail", "normal", "分析详细程度 normal/detailed/extreme")
	flag.Parse()

	if *historyFlag {
		analysis.ListHistoryFiles()
		return
	}
	if *showFlag != "" {
		analysis.ShowHistoryFile(*showFlag)
		return
	}
	// 判断是否为命令行参数模式
	if *apiKeyFlag != "" && *modelFlag != "" && *stockFlag != "" {
		stockCodes := splitAndTrim(*stockFlag)
		var hybridSearch bool
		if *modeFlag == "hybrid" {
			hybridSearch = true
		}
		// 定义 searchModes
		var searchModes []string
		switch *modeFlag {
		case "search":
			searchModes = []string{"联网搜索（结合最新互联网信息）"}
		case "hybrid":
			searchModes = []string{"深度思考+联网搜索（自动融合）"}
		default:
			searchModes = []string{"深度思考（仅用模型推理）"}
		}
		params := analysis.AnalysisParams{
			APIKey:       *apiKeyFlag,
			Model:        *modelFlag,
			StockCodes:   stockCodes,
			Start:        *startFlag,
			End:          *endFlag,
			SearchMode:   (*modeFlag == "search"),
			HybridSearch: hybridSearch,
			Periods:      splitAndTrim(*periodsFlag),
			Dims:         splitAndTrim(*dimsFlag),
			Output:       splitAndTrim(*outputFlag),
			Confidence:   (*confidenceFlag == "Y" || *confidenceFlag == "y"),
			Risk:         *riskFlag,
			Scope:        splitAndTrim(*scopeFlag),
			Lang:         *langFlag,
		}
		emails := splitAndTrim(*emailFlag)
		exportFormats := splitAndTrim(*exportFlag)
		if len(exportFormats) == 0 || exportFormats[0] == "" {
			exportFormats = []string{"md"}
		}
		if schedule := strings.TrimSpace(os.Getenv("SCHEDULE")); schedule != "" {
			fmt.Println("[定时任务] 环境变量 SCHEDULE 已设置，优先生效。")
			*scheduleFlag = schedule
		}
		if schedule := strings.TrimSpace(*scheduleFlag); schedule != "" {
			dur, err := parseSchedule(schedule)
			if err != nil {
				fmt.Println("[定时任务] 格式错误：", err)
				return
			}
			fmt.Printf("[定时任务] 启动，周期：%s\n", schedule)
			for {
				fmt.Printf("\n[%s] 批量分析开始\n", time.Now().Format("2006-01-02 15:04:05"))
				done := make(chan struct{})
				go showAnalyzingAnimation(done)
				prompt := buildPromptWithDetail(params, *detailFlag)
				results := make([]analysis.AnalysisResult, 0, len(params.StockCodes)*len(searchModes))
				for _, mode := range searchModes {
					for _, code := range params.StockCodes {
						p := params
						p.StockCodes = []string{code}
						p.SearchMode = (mode == "联网搜索（结合最新互联网信息）")
						p.HybridSearch = (mode == "深度思考+联网搜索（自动融合）")
						result := analysis.AnalyzeOne(p, func(stock, _prompt, apiKey, apiURL, model string, searchMode bool, hybridSearch bool) (string, error) {
							return analysis.GenerateAIReportWithConfigAndSearch(stock, prompt, apiKey, "https://api.deepseek.com/v1/chat/completions", model, searchMode, hybridSearch)
						})
						results = append(results, result)
					}
				}
				for _, r := range results {
					fmt.Printf("\n=== [%s] AI 智能分析报告 ===\n", r.StockCode)
					if r.Err != nil {
						fmt.Println("[AI] 生成失败:", r.Err)
					} else {
						// 分离图片引用和正文
						reportLines := strings.Split(r.Report, "\n")
						var imgLines, textLines []string
						for _, l := range reportLines {
							if strings.HasPrefix(l, "![图表](") {
								imgLines = append(imgLines, l)
							} else if strings.TrimSpace(l) != "" {
								textLines = append(textLines, l)
							}
						}
						// 先输出图片引用
						for _, l := range imgLines {
							fmt.Println(l)
						}
						// 用框输出正文
						if len(textLines) > 0 {
							printStepBox("AI 智能分析报告", textLines...)
						}
						fmt.Printf("[历史已保存: %s]\n", r.SavedFile)

						// 导出报告功能已移除
						if len(emails) > 0 && emails[0] != "" && *smtpServerFlag != "" && *smtpUserFlag != "" && *smtpPassFlag != "" {
							var attachs []string
							err := analysis.SendEmail(*smtpServerFlag, *smtpPortFlag, *smtpUserFlag, *smtpPassFlag, emails, "Quantix分析报告", r.Report, attachs)
							if err != nil {
								fmt.Println("[邮件发送失败]", err)
							} else {
								fmt.Println("[邮件已发送]")
							}
						}
						if *webhookFlag != "" {
							err := analysis.SendWebhook(*webhookFlag, r.Report)
							if err != nil {
								fmt.Println("[IM推送失败]", err)
							} else {
								fmt.Println("[IM已推送]")
							}
						}
					}
				}
				fmt.Printf("[定时任务] 下一次将在 %s 后运行，Ctrl+C 可终止。\n", dur)
				time.Sleep(dur)
				if schedule == "daily" {
					dur, _ = parseSchedule("daily") // 重新计算到明天0点的间隔
				}
				close(done)
			}
			return
		}
		done := make(chan struct{})
		go showAnalyzingAnimation(done)
		prompt := buildPromptWithDetail(params, *detailFlag)
		results := make([]analysis.AnalysisResult, 0, len(params.StockCodes)*len(searchModes))
		for _, mode := range searchModes {
			for _, code := range params.StockCodes {
				p := params
				p.StockCodes = []string{code}
				p.SearchMode = (mode == "联网搜索（结合最新互联网信息）")
				p.HybridSearch = (mode == "深度思考+联网搜索（自动融合）")
				result := analysis.AnalyzeOne(p, func(stock, _prompt, apiKey, apiURL, model string, searchMode bool, hybridSearch bool) (string, error) {
					return analysis.GenerateAIReportWithConfigAndSearch(stock, prompt, apiKey, "https://api.deepseek.com/v1/chat/completions", model, searchMode, hybridSearch)
				})
				results = append(results, result)
			}
		}
		for _, r := range results {
			fmt.Printf("\n=== [%s] AI 智能分析报告 ===\n", r.StockCode)
			if r.Err != nil {
				fmt.Println("[AI] 生成失败:", r.Err)
			} else {
				// 分离图片引用和正文
				reportLines := strings.Split(r.Report, "\n")
				var imgLines, textLines []string
				for _, l := range reportLines {
					if strings.HasPrefix(l, "![图表](") {
						imgLines = append(imgLines, l)
					} else if strings.TrimSpace(l) != "" {
						textLines = append(textLines, l)
					}
				}
				// 先输出图片引用
				for _, l := range imgLines {
					fmt.Println(l)
				}
				// 用框输出正文
				if len(textLines) > 0 {
					printStepBox("AI 智能分析报告", textLines...)
				}
				fmt.Printf("[历史已保存: %s]\n", r.SavedFile)

				// 导出报告功能已移除
				if len(emails) > 0 && emails[0] != "" && *smtpServerFlag != "" && *smtpUserFlag != "" && *smtpPassFlag != "" {
					var attachs []string
					err := analysis.SendEmail(*smtpServerFlag, *smtpPortFlag, *smtpUserFlag, *smtpPassFlag, emails, "Quantix分析报告", r.Report, attachs)
					if err != nil {
						fmt.Println("[邮件发送失败]", err)
					} else {
						fmt.Println("[邮件已发送]")
					}
				}
				if *webhookFlag != "" {
					err := analysis.SendWebhook(*webhookFlag, r.Report)
					if err != nil {
						fmt.Println("[IM推送失败]", err)
					} else {
						fmt.Println("[IM已推送]")
					}
				}
			}
		}
		close(done)
		return
	}
	// 否则进入主菜单循环
	mainMenu()
}
