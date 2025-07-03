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
	reader := bufio.NewReader(os.Stdin)
	for {
		fmt.Println("\n=== Quantix 智能股票分析系统 ===")
		fmt.Println("1. new      - 新建AI分析（支持批量，股票代码用逗号分隔）")
		fmt.Println("2. history  - 查看历史记录列表")
		fmt.Println("3. show <文件名> - 查看指定历史分析")
		fmt.Println("4. change-key - 更换DeepSeek API Key")
		fmt.Println("5. exit     - 退出程序")
		fmt.Println("提示：直接输入数字或命令名称即可")
		fmt.Print("请输入指令: ")
		input, _ := reader.ReadString('\n')
		input = strings.TrimSpace(input)

		switch input {
		case "new", "1":
			aiAnalysisInteractiveMenu()
		case "history", "2":
			listHistoryFiles()
		case "show", "3":
			fmt.Print("请输入文件名: ")
			filename, _ := reader.ReadString('\n')
			filename = strings.TrimSpace(filename)
			if filename != "" {
				showHistoryFile(filename)
			}
		case "change-key", "4":
			globalAPIKey = "" // 清空缓存，下次会重新输入
			fmt.Println("API Key已重置，下次分析时将重新输入")
		case "exit", "5":
			fmt.Println("再见！")
			return
		default:
			if strings.HasPrefix(input, "show ") {
				filename := strings.TrimSpace(strings.TrimPrefix(input, "show "))
				if filename != "" {
					showHistoryFile(filename)
				}
			} else {
				fmt.Println("无效指令，请重新输入")
			}
		}
	}
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

func aiAnalysisInteractiveMenu() {
	reader := bufio.NewReader(os.Stdin)

	// 添加操作说明
	fmt.Println("\n=== AI 智能分析配置 ===")
	fmt.Println("单选：使用 ↑↓ 箭头键移动选择，回车键确认")
	fmt.Println("多选：使用 ↑↓ 箭头键移动，空格键选择/取消，右箭头键全选，左箭头键全不选，回车键确认")
	fmt.Println("按 ? 键可查看详细操作说明")
	fmt.Println()

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
		fmt.Println("请选择要使用的AI模型：")
		model = promptForModel(deepseekModels)
	} else {
		fmt.Println("未找到可用的 DeepSeek 模型，请手动输入模型名（如 deepseek/deepseek-r1:free）")
		model, _ = reader.ReadString('\n')
		model = strings.TrimSpace(model)
	}
	fmt.Print("请输入股票代码（可批量，逗号分隔）: ")
	stockInput := interactiveInput("请输入股票代码（可批量，逗号分隔）:", "")
	stockCodes := splitAndTrim(stockInput)
	if len(stockCodes) == 0 || stockCodes[0] == "" {
		fmt.Println("股票代码不能为空！")
		return
	}
	today := time.Now()
	defaultEnd := today.Format("2006-01-02")
	defaultStart := today.AddDate(0, 0, -30).Format("2006-01-02")
	fmt.Printf("请输入开始日期(YYYY-MM-DD, 默认%s): ", defaultStart)
	start := interactiveInput(fmt.Sprintf("请输入开始日期(YYYY-MM-DD, 默认%s):", defaultStart), defaultStart)
	fmt.Printf("请输入结束日期(YYYY-MM-DD, 默认%s): ", defaultEnd)
	end := interactiveInput(fmt.Sprintf("请输入结束日期(YYYY-MM-DD, 默认%s):", defaultEnd), defaultEnd)
	searchMode := promptForSearchMode()
	periods, dims, searchScope, outputFormat, needConfidence, riskPref := promptForPredictionOptions()
	// 语言选择
	langOptions := []string{"中文", "英文"}
	defaultLang := []string{"中文"}
	langResult := interactiveSelectList("请选择分析语言：", langOptions, defaultLang)
	var lang string
	if len(langResult) > 0 && langResult[0] == "英文" {
		lang = "en"
	} else {
		lang = "zh"
	}

	// 导出格式选择
	exportOptions := []string{"Markdown", "HTML"}
	defaultExport := []string{"Markdown"}
	exportResult := interactiveSelectList("请选择导出格式（可多选）：", exportOptions, defaultExport)
	exportFormats := make([]string, 0, len(exportResult))
	for _, fmt := range exportResult {
		switch fmt {
		case "Markdown":
			exportFormats = append(exportFormats, "md")
		case "HTML":
			exportFormats = append(exportFormats, "html")
		}
	}
	if len(exportFormats) == 0 {
		exportFormats = []string{"md"}
	}
	fmt.Print("如需邮件推送请输入收件人邮箱（可逗号分隔，留空跳过）: ")
	emailInput := interactiveInput("如需邮件推送请输入收件人邮箱（可逗号分隔，留空跳过）:", "")
	emails := splitAndTrim(emailInput)
	var smtpServer, smtpUser, smtpPass string
	var smtpPort int
	if len(emails) > 0 && emails[0] != "" {
		fmt.Print("SMTP服务器: ")
		smtpServer, _ = reader.ReadString('\n')
		smtpServer = strings.TrimSpace(smtpServer)
		fmt.Print("SMTP端口(默认465): ")
		portInput := interactiveInput("SMTP端口(默认465):", "")
		if portInput == "" {
			smtpPort = 465
		} else {
			smtpPort, _ = strconv.Atoi(portInput)
		}
		fmt.Print("SMTP用户名: ")
		smtpUser, _ = reader.ReadString('\n')
		smtpUser = strings.TrimSpace(smtpUser)
		fmt.Print("SMTP密码: ")
		smtpPass, _ = reader.ReadString('\n')
		smtpPass = strings.TrimSpace(smtpPass)
	}
	fmt.Print("如需IM推送请输入Webhook地址（钉钉/企业微信，留空跳过）: ")
	webhook := interactiveInput("如需IM推送请输入Webhook地址（钉钉/企业微信，留空跳过）:", "")
	// 分析详细程度选择
	detailOptions := []string{"普通分析 - 基础技术指标和简要分析", "详细分析 - 多维度深度分析，包含更多指标", "极致分析 - 最全面的分析，包含所有可用指标和深度洞察"}
	defaultDetail := []string{"普通分析 - 基础技术指标和简要分析"}

	detailResult := interactiveSelectList("请选择分析详细程度：", detailOptions, defaultDetail)
	var detailInput string
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
	} else {
		detailInput = "normal"
	}
	params := analysis.AnalysisParams{
		APIKey:     apiKey,
		Model:      model,
		StockCodes: stockCodes,
		Start:      start,
		End:        end,
		SearchMode: searchMode,
		Periods:    periods,
		Dims:       dims,
		Output:     []string{outputFormat},
		Confidence: needConfidence,
		Risk:       riskPref,
		Scope:      searchScope,
		Lang:       lang,
	}

	fmt.Println("\n=== 开始AI智能分析 ===")
	fmt.Printf("分析股票：%s\n", strings.Join(stockCodes, ", "))
	fmt.Printf("分析期间：%s 至 %s\n", start, end)
	fmt.Printf("分析模式：%s\n", func() string {
		if searchMode {
			return "联网搜索模式"
		}
		return "深度思考模式"
	}())
	fmt.Println("正在生成分析报告，请稍候...")

	prompt := buildPromptWithDetail(params, detailInput)
	done := make(chan struct{})
	go showAnalyzingAnimation(done)
	results := make([]analysis.AnalysisResult, 0, len(params.StockCodes))
	for _, code := range params.StockCodes {
		p := params
		p.StockCodes = []string{code}
		result := analysis.AnalyzeOne(p, func(stock, _prompt, apiKey, apiURL, model string, searchMode bool) (string, error) {
			return analysis.GenerateAIReportWithConfigAndSearch(stock, prompt, apiKey, "https://api.deepseek.com/v1/chat/completions", model, searchMode)
		})
		results = append(results, result)
	}
	for _, r := range results {
		fmt.Printf("\n=== [%s] AI 智能分析报告 ===\n", r.StockCode)
		if r.Err != nil {
			fmt.Println("[AI] 生成失败:", r.Err)
		} else {
			fmt.Println(r.Report)
			fmt.Printf("[历史已保存: %s]\n", r.SavedFile)

			// 导出报告功能已移除

			// 邮件推送
			if len(emails) > 0 && emails[0] != "" && smtpServer != "" && smtpUser != "" && smtpPass != "" {
				var attachs []string
				for _, fmtx := range exportFormats {
					if fmtx == "html" {
						attachs = append(attachs, "history/"+r.SavedFile[:len(r.SavedFile)-5]+"."+fmtx)
					}
				}
				err := analysis.SendEmail(smtpServer, smtpPort, smtpUser, smtpPass, emails, "Quantix分析报告", r.Report, attachs)
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

func main() {
	survey.MultiSelectQuestionTemplate = `
{{- define "option"}}
    {{- if eq $.SelectedIndex $.CurrentIndex }}{{color "cyan"}}> {{if index $.Checked $.CurrentOpt.Index }}[✓]{{else}}[ ]{{end}} {{$.CurrentOpt.Value}}{{color "reset"}}{{else}}  {{if index $.Checked $.CurrentOpt.Index }}[✓]{{else}}[ ]{{end}} {{$.CurrentOpt.Value}}{{end}}
{{"\n"}}
{{- end}}
{{- if .ShowHelp }}{{color "cyan"}}{{ .Help }}{{color "reset"}}{{"\n"}}{{end}}
{{- color "cyan"}}?{{color "reset"}} {{ .Message }}{{ .FilterMessage }}
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
{{- color "cyan"}}?{{color "reset"}} {{ .Message }}{{ .FilterMessage }}
{{- if .ShowAnswer}}{{color "cyan"}} {{.Answer}}{{color "reset"}}{{"\n"}}
{{- else }}
  {{- "  "}}{{- color "cyan"}}[使用箭头键移动，回车键确认，? 查看帮助]{{color "reset"}}
  {{- "\n"}}
  {{- range $ix, $option := .PageEntries}}
    {{- if eq $.SelectedIndex $ix }}{{color "cyan"}}> {{else}}  {{end}}
    {{- " "}}{{- $option.Value}}{{ if ne ($.GetDescription $option) "" }} - {{color "cyan"}}{{ $.GetDescription $option }}{{color "reset"}}{{end}}
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
	exportFlag := flag.String("export", "md", "导出格式，逗号分隔，支持md,html")
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
		params := analysis.AnalysisParams{
			APIKey:     *apiKeyFlag,
			Model:      *modelFlag,
			StockCodes: stockCodes,
			Start:      *startFlag,
			End:        *endFlag,
			SearchMode: (*modeFlag == "search"),
			Periods:    splitAndTrim(*periodsFlag),
			Dims:       splitAndTrim(*dimsFlag),
			Output:     splitAndTrim(*outputFlag),
			Confidence: (*confidenceFlag == "Y" || *confidenceFlag == "y"),
			Risk:       *riskFlag,
			Scope:      splitAndTrim(*scopeFlag),
			Lang:       *langFlag,
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
				results := make([]analysis.AnalysisResult, 0, len(params.StockCodes))
				for _, code := range params.StockCodes {
					p := params
					p.StockCodes = []string{code}
					result := analysis.AnalyzeOne(p, func(stock, _prompt, apiKey, apiURL, model string, searchMode bool) (string, error) {
						return analysis.GenerateAIReportWithConfigAndSearch(stock, prompt, apiKey, "https://api.deepseek.com/v1/chat/completions", model, searchMode)
					})
					results = append(results, result)
				}
				for _, r := range results {
					fmt.Printf("\n=== [%s] AI 智能分析报告 ===\n", r.StockCode)
					if r.Err != nil {
						fmt.Println("[AI] 生成失败:", r.Err)
					} else {
						fmt.Println(r.Report)
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
		results := make([]analysis.AnalysisResult, 0, len(params.StockCodes))
		for _, code := range params.StockCodes {
			p := params
			p.StockCodes = []string{code}
			result := analysis.AnalyzeOne(p, func(stock, _prompt, apiKey, apiURL, model string, searchMode bool) (string, error) {
				return analysis.GenerateAIReportWithConfigAndSearch(stock, prompt, apiKey, "https://api.deepseek.com/v1/chat/completions", model, searchMode)
			})
			results = append(results, result)
		}
		for _, r := range results {
			fmt.Printf("\n=== [%s] AI 智能分析报告 ===\n", r.StockCode)
			if r.Err != nil {
				fmt.Println("[AI] 生成失败:", r.Err)
			} else {
				fmt.Println(r.Report)
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
