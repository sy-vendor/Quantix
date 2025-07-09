package main

import (
	"Quantix/analysis"
	"bufio"
	"encoding/csv"
	"encoding/json"
	"errors"
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
	"github.com/AlecAivazis/survey/v2/core"
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
	compat := "\033[33m[兼容性提示] 如无法选择请用本地原生终端（如 macOS Terminal/iTerm2/Windows Terminal），避免VSCode嵌入终端或远程SSH窗口。\033[0m"
	prompt := &survey.MultiSelect{
		Message: title + "\n" + compat + "\n操作说明：↑↓箭头键移动，空格键选择/取消，右箭头键全选，左箭头键全不选，回车键确认",
		Options: options,
		Default: defaultSelected,
		Help:    "操作说明：↑↓箭头键移动，空格键选择/取消，右箭头键全选，左箭头键全不选，回车键确认。\n" + compat,
	}

	err := survey.AskOne(prompt, &result,
		// 自定义校验器，提示中文
		survey.WithValidator(func(ans interface{}) error {
			if arr, ok := ans.([]core.OptionAnswer); ok {
				if len(arr) == 0 {
					return errors.New("请至少选择一项")
				}
			}
			if arr, ok := ans.([]string); ok {
				if len(arr) == 0 {
					return errors.New("请至少选择一项")
				}
			}
			return nil
		}),
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
	// 预测周期多选 - 扩展更多时间维度
	periodOptions := []string{
		"1天", "3天", "1周", "2周", "1月", "2月", "3月", "半年", "1年", "2年", "3年",
		"短期(1-7天)", "中期(1-3月)", "长期(3-12月)", "超长期(1年以上)",
	}
	defaultPeriods := []string{"1周", "1月", "3月"}
	periods = interactiveSelectList("请选择预测周期（可多选）：", periodOptions, defaultPeriods)

	// 分析维度多选 - 扩展更多分析角度
	dimOptions := []string{
		"技术面", "基本面", "资金面", "行业对比", "情绪分析",
		"K线形态", "均线系统", "成交量分析", "技术指标", "支撑阻力",
		"财务数据", "盈利能力", "估值分析", "行业地位", "管理层",
		"主力资金", "北向资金", "大宗交易", "机构持仓", "散户情绪",
		"新闻舆情", "研报分析", "公告解读", "论坛讨论", "社交媒体",
		"宏观经济", "政策影响", "国际环境", "产业链", "竞争格局",
	}
	defaultDims := []string{"技术面", "基本面", "资金面", "行业对比", "情绪分析"}
	dims = interactiveSelectList("请选择分析维度（可多选）：", dimOptions, defaultDims)

	// 输出格式单选
	outputOptions := []string{"结构化表格", "要点", "详细长文", "摘要", "图表化报告", "多维度对比"}
	outputFormat = interactiveSingleSelect("请选择输出格式：", outputOptions, outputOptions[0])

	// 置信度单选
	confOptions := []string{"需要置信度/概率说明", "不需要置信度/概率说明"}
	needConfidence = interactiveSingleSelect("是否需要置信度/概率说明？", confOptions, confOptions[0]) == confOptions[0]

	// 风险偏好单选
	riskOptions := []string{"保守", "稳健", "激进", "风险为主", "机会为主", "平衡型"}
	riskPref = interactiveSingleSelect("请选择风险/机会偏好：", riskOptions, riskOptions[0])

	// 联网搜索范围多选
	scopeOptions := []string{"新闻", "研报", "公告", "论坛", "社交媒体", "政策文件", "行业报告", "专家观点"}
	defaultScope := []string{"新闻", "公告", "研报"}
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
		return basePrompt + `
【极致详细分析要求】
请将每个分析维度细分到最小颗粒度，涵盖：

1. 技术面深度分析：
   - K线形态：头肩顶/底、双顶/底、三角形、旗形、楔形等
   - 均线系统：MA5/10/20/60/120/250排列、金叉死叉、均线粘合
   - 成交量：量价关系、放量缩量、量能背离、筹码分布
   - 技术指标：MACD、KDJ、RSI、BOLL、CCI、OBV、DMI等
   - 支撑阻力：历史支撑阻力位、心理价位、技术位

2. 基本面深度分析：
   - 财务数据：营收、净利润、毛利率、净利率、ROE、ROA
   - 盈利能力：EPS、PE、PB、PS、PEG、股息率
   - 估值分析：DCF模型、相对估值、行业对比
   - 行业地位：市场份额、竞争优势、护城河
   - 管理层：管理能力、战略规划、执行力

3. 资金面深度分析：
   - 主力资金：大单流入流出、机构持仓变化
   - 北向资金：外资流入流出、持股比例
   - 大宗交易：折溢价、交易对手、目的分析
   - 机构持仓：基金、保险、券商持仓变化
   - 散户情绪：融资融券、股东人数变化

4. 情绪面深度分析：
   - 新闻舆情：正面/负面新闻比例、热点事件影响
   - 研报分析：评级变化、目标价调整、分析师观点
   - 公告解读：重大事项、业绩预告、股权变动
   - 论坛讨论：投资者情绪、关注度变化
   - 社交媒体：话题热度、情感倾向

5. 多周期预测：
   - 短期(1-7天)：技术反弹、消息面影响
   - 中期(1-3月)：趋势延续、基本面变化
   - 长期(3-12月)：估值修复、行业周期
   - 超长期(1年以上)：成长性、战略价值

6. 风险与机会：
   - 系统性风险：宏观经济、政策变化
   - 个股风险：经营风险、财务风险、流动性风险
   - 机会识别：估值修复、业绩改善、政策利好

所有结论都要有数据和理由支撑，输出结构化表格+要点+详细长文，适合专业投资者参考。`
	}

	if detail == "detailed" {
		return basePrompt + `
【详细分析要求】
请对每个分析维度进行细致展开，涵盖：

1. 技术面分析：K线形态、均线系统、成交量、技术指标、支撑阻力
2. 基本面分析：财务数据、盈利能力、估值分析、行业地位、管理层
3. 资金面分析：主力资金、北向资金、大宗交易、机构持仓、散户情绪
4. 情绪面分析：新闻舆情、研报分析、公告解读、论坛讨论、社交媒体
5. 多周期预测：短期、中期、长期趋势预测
6. 操作建议：买入/持有/卖出建议，仓位控制
7. 风险与机会：主要风险点、潜在机会

所有结论都要有理由和数据支撑，给出多周期预测、操作建议、风险与机会。`
	}

	return basePrompt + `
【标准分析要求】
请提供以下分析：

1. 技术面：K线形态、均线系统、成交量、技术指标
2. 基本面：财务数据、盈利能力、估值分析
3. 资金面：主力资金、北向资金、机构持仓
4. 情绪面：新闻舆情、研报分析、公告解读
5. 多周期预测：短期、中期、长期趋势
6. 操作建议：买入/持有/卖出建议
7. 风险提示：主要风险点

请结合上方K线图、均线图、成交量图，对当前股票的走势、支撑阻力、均线形态、量价关系等进行详细分析，给出趋势判断、操作建议和风险提示。`
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

	// Step 5.5: 详细预测选项
	printStepBox("Step 5.5: Detailed Prediction Options",
		"选择具体的预测项目和参数",
		"说明：可多选，回车确认",
	)

	// 预测类型选择
	predictionTypeOptions := []string{
		"价格预测", "波动率预测", "成交量预测", "涨跌概率预测",
		"技术指标预测", "基本面指标预测", "情绪评分预测",
		"市场定位预测", "竞争优势预测", "风险等级预测",
	}
	defaultPredictionTypes := []string{"价格预测", "涨跌概率预测", "风险等级预测"}
	predictionTypes := interactiveSelectList("请选择预测类型（可多选）：", predictionTypeOptions, defaultPredictionTypes)

	// 具体预测项目选择
	predictionItemOptions := []string{
		"目标价位预测", "止损位预测", "止盈位预测", "波动率预测",
		"成交量预测", "涨跌概率预测", "风险等级预测", "趋势强度预测",
		"支撑阻力位预测", "技术信号预测", "基本面指标预测", "情绪评分预测",
		"市场定位分析", "竞争优势分析",
	}
	defaultPredictionItems := []string{"目标价位预测", "止损位预测", "止盈位预测", "涨跌概率预测"}
	predictionItems := interactiveSelectList("请选择具体预测项目（可多选）：", predictionItemOptions, defaultPredictionItems)

	printStepBox("Step 5.5: Detailed Prediction Options",
		fmt.Sprintf("[预测类型]: %s", strings.Join(predictionTypes, ", ")),
		fmt.Sprintf("[预测项目]: %s", strings.Join(predictionItems, ", ")),
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
		APIKey:               apiKey,
		Model:                model,
		StockCodes:           stockCodes,
		Start:                start,
		End:                  end,
		Periods:              periods,
		Dims:                 dims,
		Output:               exportFormats,
		Confidence:           needConfidence,
		Risk:                 riskPref,
		Scope:                searchScope,
		Lang:                 lang,
		PredictionTypes:      predictionTypes,
		TargetPrice:          contains(predictionItems, "目标价位预测"),
		StopLoss:             contains(predictionItems, "止损位预测"),
		TakeProfit:           contains(predictionItems, "止盈位预测"),
		Volatility:           contains(predictionItems, "波动率预测"),
		Volume:               contains(predictionItems, "成交量预测"),
		Probability:          contains(predictionItems, "涨跌概率预测"),
		RiskLevel:            contains(predictionItems, "风险等级预测"),
		TrendStrength:        contains(predictionItems, "趋势强度预测"),
		SupportResistance:    contains(predictionItems, "支撑阻力位预测"),
		TechnicalSignals:     contains(predictionItems, "技术信号预测"),
		FundamentalMetrics:   contains(predictionItems, "基本面指标预测"),
		SentimentScore:       contains(predictionItems, "情绪评分预测"),
		MarketPosition:       contains(predictionItems, "市场定位分析"),
		CompetitiveAdvantage: contains(predictionItems, "竞争优势分析"),
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

	// Step 5.5: 详细预测选项
	printStepBox("Step 5.5: Detailed Prediction Options",
		"选择具体的预测项目和参数",
		"说明：可多选，回车确认",
	)

	// 预测类型选择
	predictionTypeOptions := []string{
		"价格预测", "波动率预测", "成交量预测", "涨跌概率预测",
		"技术指标预测", "基本面指标预测", "情绪评分预测",
		"市场定位预测", "竞争优势预测", "风险等级预测",
	}
	defaultPredictionTypes := []string{"价格预测", "涨跌概率预测", "风险等级预测"}
	predictionTypes := interactiveSelectList("请选择预测类型（可多选）：", predictionTypeOptions, defaultPredictionTypes)

	// 具体预测项目选择
	predictionItemOptions := []string{
		"目标价位预测", "止损位预测", "止盈位预测", "波动率预测",
		"成交量预测", "涨跌概率预测", "风险等级预测", "趋势强度预测",
		"支撑阻力位预测", "技术信号预测", "基本面指标预测", "情绪评分预测",
		"市场定位分析", "竞争优势分析",
	}
	defaultPredictionItems := []string{"目标价位预测", "止损位预测", "止盈位预测", "涨跌概率预测"}
	predictionItems := interactiveSelectList("请选择具体预测项目（可多选）：", predictionItemOptions, defaultPredictionItems)

	printStepBox("Step 5.5: Detailed Prediction Options",
		fmt.Sprintf("[预测类型]: %s", strings.Join(predictionTypes, ", ")),
		fmt.Sprintf("[预测项目]: %s", strings.Join(predictionItems, ", ")),
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
		APIKey:               apiKey,
		Model:                model,
		StockCodes:           stockCodes,
		Start:                start,
		End:                  end,
		Periods:              periods,
		Dims:                 dims,
		Output:               exportFormats,
		Confidence:           needConfidence,
		Risk:                 riskPref,
		Scope:                searchScope,
		Lang:                 lang,
		PredictionTypes:      predictionTypes,
		TargetPrice:          contains(predictionItems, "目标价位预测"),
		StopLoss:             contains(predictionItems, "止损位预测"),
		TakeProfit:           contains(predictionItems, "止盈位预测"),
		Volatility:           contains(predictionItems, "波动率预测"),
		Volume:               contains(predictionItems, "成交量预测"),
		Probability:          contains(predictionItems, "涨跌概率预测"),
		RiskLevel:            contains(predictionItems, "风险等级预测"),
		TrendStrength:        contains(predictionItems, "趋势强度预测"),
		SupportResistance:    contains(predictionItems, "支撑阻力位预测"),
		TechnicalSignals:     contains(predictionItems, "技术信号预测"),
		FundamentalMetrics:   contains(predictionItems, "基本面指标预测"),
		SentimentScore:       contains(predictionItems, "情绪评分预测"),
		MarketPosition:       contains(predictionItems, "市场定位分析"),
		CompetitiveAdvantage: contains(predictionItems, "竞争优势分析"),
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
	survey.ErrorTemplate = `
{{- color "red"}}提示：{{.Error.Error}}{{color "reset"}}
`
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
	updateActualFlag := flag.Bool("update-actual", false, "批量补全预测的实际行情（T+1、T+5、T+20）")
	flag.Parse()

	if *updateActualFlag {
		updateActualPricesWithDeepSeek()
		return
	}
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

// contains 检查字符串数组中是否包含指定字符串
func contains(arr []string, item string) bool {
	for _, i := range arr {
		if i == item {
			return true
		}
	}
	return false
}

// updateActualPricesWithDeepSeek 遍历predictions.csv，调用DeepSeek联网补全实际收盘价
func updateActualPricesWithDeepSeek() {
	fmt.Println("[预测追踪] 开始批量补全实际行情...")

	// 获取 API Key 和模型
	apiKey := promptForAPIKey()
	if apiKey == "" {
		fmt.Fprintf(os.Stderr, "[预测追踪] 需要 DeepSeek API Key\n")
		return
	}

	// 获取可用模型
	models, err := fetchDeepSeekModels(apiKey, "")
	if err != nil {
		fmt.Fprintf(os.Stderr, "[预测追踪] 获取模型失败: %v\n", err)
		return
	}

	deepseekModels := make([]string, 0)
	for _, m := range models {
		if strings.Contains(m, "deepseek") {
			deepseekModels = append(deepseekModels, m)
		}
	}

	if len(deepseekModels) == 0 {
		fmt.Fprintf(os.Stderr, "[预测追踪] 未找到可用的 DeepSeek 模型\n")
		return
	}

	model := promptForModel(deepseekModels)
	if model == "" {
		fmt.Fprintf(os.Stderr, "[预测追踪] 需要选择 DeepSeek 模型\n")
		return
	}

	csvPath := "history/predictions.csv"
	f, err := os.Open(csvPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "[预测追踪] 无法打开CSV: %v\n", err)
		return
	}
	defer f.Close()

	reader := csv.NewReader(f)
	records, err := reader.ReadAll()
	if err != nil {
		fmt.Fprintf(os.Stderr, "[预测追踪] 读取CSV失败: %v\n", err)
		return
	}

	if len(records) == 0 {
		fmt.Println("[预测追踪] CSV为空，无需补全。")
		return
	}

	// 检查是否有实际价格列，如果没有则添加
	headers := records[0]
	hasActualCols := false
	for _, h := range headers {
		if strings.Contains(h, "实际") {
			hasActualCols = true
			break
		}
	}

	if !hasActualCols {
		// 添加实际价格列
		headers = append(headers, "T+1实际收盘价", "T+5实际收盘价", "T+20实际收盘价")
		records[0] = headers
	}

	// 找到实际价格列的索引
	t1Idx := -1
	t5Idx := -1
	t20Idx := -1
	for i, h := range headers {
		if h == "T+1实际收盘价" {
			t1Idx = i
		} else if h == "T+5实际收盘价" {
			t5Idx = i
		} else if h == "T+20实际收盘价" {
			t20Idx = i
		}
	}

	// 处理每一行预测记录
	updated := false
	for i := 1; i < len(records); i++ {
		row := records[i]
		stock := row[0]
		predDate := row[1]

		// 检查是否已经有实际价格
		if t1Idx < len(row) && t5Idx < len(row) && t20Idx < len(row) {
			if row[t1Idx] != "" && row[t5Idx] != "" && row[t20Idx] != "" {
				continue // 已有实际价格，跳过
			}
		}

		// 计算目标日期
		layout := "2006-01-02"
		base, err := time.Parse(layout, predDate)
		if err != nil {
			fmt.Fprintf(os.Stderr, "[预测追踪] 解析日期失败 %s: %v\n", predDate, err)
			continue
		}

		dates := []time.Time{
			base.AddDate(0, 0, 1),
			base.AddDate(0, 0, 5),
			base.AddDate(0, 0, 20),
		}

		// 生成查询 prompt
		dateStrs := make([]string, len(dates))
		for j, d := range dates {
			dateStrs[j] = d.Format(layout)
		}

		prompt := fmt.Sprintf(`请联网查询股票%s在%s的收盘价，并严格按照以下markdown表格格式输出：

| 日期 | 收盘价 |
|------|--------|
| %s |        |
| %s |        |
| %s |        |

请确保：
1. 联网查询最新准确的收盘价数据
2. 严格按照表格格式输出，不要添加其他内容
3. 如果某个日期是周末或节假日，请标注"休市"`,
			stock, strings.Join(dateStrs, "、"), dateStrs[0], dateStrs[1], dateStrs[2])

		fmt.Printf("[预测追踪] 查询 %s %s 的实际收盘价...\n", stock, predDate)

		// 调用 DeepSeek API
		result, err := analysis.GenerateAIReportWithConfigAndSearch(
			stock, prompt, apiKey, "https://api.deepseek.com/v1/chat/completions",
			model, true, true) // 启用联网搜索

		if err != nil {
			fmt.Fprintf(os.Stderr, "[预测追踪] 查询失败 %s %s: %v\n", stock, predDate, err)
			continue
		}

		// 解析表格结果
		prices := parseActualPricesFromTable(result)
		if len(prices) == 3 {
			// 确保行有足够的列
			for len(row) <= t20Idx {
				row = append(row, "")
			}

			// 更新实际价格
			if t1Idx >= 0 && t1Idx < len(row) {
				row[t1Idx] = prices[0]
			}
			if t5Idx >= 0 && t5Idx < len(row) {
				row[t5Idx] = prices[1]
			}
			if t20Idx >= 0 && t20Idx < len(row) {
				row[t20Idx] = prices[2]
			}

			records[i] = row
			updated = true
			fmt.Printf("[预测追踪] ✓ %s %s: T+1=%s, T+5=%s, T+20=%s\n",
				stock, predDate, prices[0], prices[1], prices[2])
		} else {
			fmt.Fprintf(os.Stderr, "[预测追踪] 解析失败 %s %s: 期望3个价格，实际%d个\n",
				stock, predDate, len(prices))
		}

		// 避免 API 调用过于频繁
		time.Sleep(1 * time.Second)
	}

	// 写回 CSV
	if updated {
		f.Close()
		f, err = os.Create(csvPath)
		if err != nil {
			fmt.Fprintf(os.Stderr, "[预测追踪] 无法创建CSV: %v\n", err)
			return
		}
		defer f.Close()

		writer := csv.NewWriter(f)
		defer writer.Flush()

		for _, record := range records {
			writer.Write(record)
		}

		fmt.Printf("[预测追踪] ✓ 已更新 %d 条预测记录的实际价格\n", len(records)-1)
	} else {
		fmt.Println("[预测追踪] 所有记录都已包含实际价格，无需更新")
	}
}

// parseActualPricesFromTable 从 AI 返回的 markdown 表格中解析实际价格
func parseActualPricesFromTable(result string) []string {
	prices := make([]string, 0)

	// 查找表格行
	lines := strings.Split(result, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "|") && strings.Contains(line, "|") {
			// 跳过表头和分隔线
			if strings.Contains(line, "日期") || strings.Contains(line, "---") {
				continue
			}

			// 解析表格行
			parts := strings.Split(line, "|")
			if len(parts) >= 3 {
				price := strings.TrimSpace(parts[2])
				if price != "" && price != "收盘价" {
					prices = append(prices, price)
				}
			}
		}
	}

	return prices
}
