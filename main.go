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
		fmt.Println("1. new      - 新建AI分析（支持批量，股票代码用逗号分隔）")
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
			analysis.ListHistoryFiles()
		} else if strings.HasPrefix(input, "show ") {
			parts := strings.SplitN(input, " ", 2)
			if len(parts) == 2 {
				analysis.ShowHistoryFile(parts[1])
			} else {
				fmt.Println("用法: show <文件名>")
			}
		} else if input == "new" || input == "1" {
			aiAnalysisInteractiveMenu()
		} else {
			fmt.Println("无效指令，请重试。")
		}
	}
}

func aiAnalysisInteractiveMenu() {
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
	fmt.Print("请输入股票代码（可批量，逗号分隔）: ")
	stockInput, _ := reader.ReadString('\n')
	stockInput = strings.TrimSpace(stockInput)
	stockCodes := splitAndTrim(stockInput)
	if len(stockCodes) == 0 || stockCodes[0] == "" {
		fmt.Println("股票代码不能为空！")
		return
	}
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
	fmt.Print("请选择导出格式（可多选，逗号分隔，支持md,html,pdf，默认md）: ")
	exportInput, _ := reader.ReadString('\n')
	exportInput = strings.TrimSpace(exportInput)
	exportFormats := splitAndTrim(exportInput)
	if len(exportFormats) == 0 || exportFormats[0] == "" {
		exportFormats = []string{"md"}
	}
	fmt.Print("如需邮件推送请输入收件人邮箱（可逗号分隔，留空跳过）: ")
	emailInput, _ := reader.ReadString('\n')
	emailInput = strings.TrimSpace(emailInput)
	emails := splitAndTrim(emailInput)
	var smtpServer, smtpUser, smtpPass string
	var smtpPort int
	if len(emails) > 0 && emails[0] != "" {
		fmt.Print("SMTP服务器: ")
		smtpServer, _ = reader.ReadString('\n')
		smtpServer = strings.TrimSpace(smtpServer)
		fmt.Print("SMTP端口(默认465): ")
		portInput, _ := reader.ReadString('\n')
		portInput = strings.TrimSpace(portInput)
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
	webhook, _ := reader.ReadString('\n')
	webhook = strings.TrimSpace(webhook)
	params := analysis.AnalysisParams{
		APIKey:     apiKey,
		Model:      model,
		StockCodes: stockCodes,
		Start:      start,
		End:        end,
		SearchMode: searchMode,
		Periods:    periods,
		Dims:       dims,
		Output:     outputFormat,
		Confidence: needConfidence,
		Risk:       riskPref,
		Scope:      searchScope,
		Lang:       lang,
	}
	results := analysis.AnalyzeBatch(params, analysis.GenerateAIReportWithConfigAndSearch)
	for _, r := range results {
		fmt.Printf("\n=== [%s] AI 智能分析报告 ===\n", r.StockCode)
		if r.Err != nil {
			fmt.Println("[AI] 生成失败:", r.Err)
		} else {
			fmt.Println(r.Report)
			fmt.Printf("[历史已保存: %s]\n", r.SavedFile)
			var attachs []string
			for _, fmtx := range exportFormats {
				fname, err := analysis.ExportReport(r, fmtx)
				if err == nil && fname != "" {
					fmt.Printf("[已导出: %s]\n", fname)
					if fmtx == "pdf" || fmtx == "md" || fmtx == "html" {
						attachs = append(attachs, "history/"+fname)
					}
				}
			}
			// 邮件推送
			if len(emails) > 0 && emails[0] != "" && smtpServer != "" && smtpUser != "" && smtpPass != "" {
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
				results := analysis.AnalyzeBatch(params, analysis.GenerateAIReportWithConfigAndSearch)
				for _, r := range results {
					fmt.Printf("\n=== [%s] AI 智能分析报告 ===\n", r.StockCode)
					if r.Err != nil {
						fmt.Println("[AI] 生成失败:", r.Err)
					} else {
						fmt.Println(r.Report)
						fmt.Printf("[历史已保存: %s]\n", r.SavedFile)
						var attachs []string
						for _, fmtx := range exportFormats {
							fname, err := analysis.ExportReport(r, fmtx)
							if err == nil && fname != "" {
								fmt.Printf("[已导出: %s]\n", fname)
								if fmtx == "pdf" || fmtx == "md" || fmtx == "html" {
									attachs = append(attachs, "history/"+fname)
								}
							}
						}
						if len(emails) > 0 && emails[0] != "" && *smtpServerFlag != "" && *smtpUserFlag != "" && *smtpPassFlag != "" {
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
			}
			return
		}
		results := analysis.AnalyzeBatch(params, analysis.GenerateAIReportWithConfigAndSearch)
		for _, r := range results {
			fmt.Printf("\n=== [%s] AI 智能分析报告 ===\n", r.StockCode)
			if r.Err != nil {
				fmt.Println("[AI] 生成失败:", r.Err)
			} else {
				fmt.Println(r.Report)
				fmt.Printf("[历史已保存: %s]\n", r.SavedFile)
				var attachs []string
				for _, fmtx := range exportFormats {
					fname, err := analysis.ExportReport(r, fmtx)
					if err == nil && fname != "" {
						fmt.Printf("[已导出: %s]\n", fname)
						if fmtx == "pdf" || fmtx == "md" || fmtx == "html" {
							attachs = append(attachs, "history/"+fname)
						}
					}
				}
				if len(emails) > 0 && emails[0] != "" && *smtpServerFlag != "" && *smtpUserFlag != "" && *smtpPassFlag != "" {
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
		return
	}
	// 否则进入主菜单循环
	mainMenu()
}
