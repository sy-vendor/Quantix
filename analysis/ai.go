package analysis

import (
	"fmt"
	"io/ioutil"
	"math"
	"os"
	"path/filepath"
	"strings"
	"time"

	"encoding/json"
	"net/http"
	"strconv"

	"context"

	"regexp"

	"github.com/chromedp/cdproto/page"
	"github.com/chromedp/chromedp"
	"github.com/mattn/go-runewidth"
	"github.com/russross/blackfriday/v2"
	"golang.org/x/term"
)

// 类型定义补充

// AnalysisParams 用于传递分析参数
// 可根据main.go和ai.go的实际用法调整字段
// StockData、TechnicalIndicator用于chart.go

type AnalysisParams struct {
	APIKey       string
	Model        string
	StockCodes   []string
	Start        string
	End          string
	SearchMode   bool
	HybridSearch bool // 新增，混合模式
	Periods      []string
	Dims         []string
	Output       []string
	Confidence   bool
	Risk         string
	Scope        []string
	Lang         string
	Prompt       string // 可选，手动传递prompt

	// 新增：扩展预测参数
	PredictionTypes      []string // 预测类型：价格、波动率、成交量、涨跌概率等
	TargetPrice          bool     // 是否预测目标价位
	StopLoss             bool     // 是否预测止损位
	TakeProfit           bool     // 是否预测止盈位
	Volatility           bool     // 是否预测波动率
	Volume               bool     // 是否预测成交量
	Probability          bool     // 是否预测涨跌概率
	RiskLevel            bool     // 是否预测风险等级
	TrendStrength        bool     // 是否预测趋势强度
	SupportResistance    bool     // 是否预测支撑阻力位
	TechnicalSignals     bool     // 是否预测技术信号
	FundamentalMetrics   bool     // 是否预测基本面指标
	SentimentScore       bool     // 是否预测情绪评分
	MarketPosition       bool     // 是否预测市场定位
	CompetitiveAdvantage bool     // 是否预测竞争优势
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
	Close  float64
	Low    float64
	High   float64
	Volume float64
}

type TechnicalIndicator struct {
	MA5  float64
	MA10 float64
	MA20 float64
	MA60 float64

	// 新增：更多技术指标
	MA120 float64 // 120日均线
	MA250 float64 // 250日均线

	// MACD指标
	MACD          float64 // MACD线
	MACDSignal    float64 // MACD信号线
	MACDHistogram float64 // MACD柱状图

	// KDJ指标
	K float64 // K值
	D float64 // D值
	J float64 // J值

	// RSI指标
	RSI6  float64 // 6日RSI
	RSI12 float64 // 12日RSI
	RSI24 float64 // 24日RSI

	// BOLL指标
	BOLLUpper  float64 // 布林带上轨
	BOLLMiddle float64 // 布林带中轨
	BOLLLower  float64 // 布林带下轨

	// 成交量指标
	VolumeMA5  float64 // 5日成交量均线
	VolumeMA10 float64 // 10日成交量均线
	VolumeMA20 float64 // 20日成交量均线

	// 其他技术指标
	CCI       float64 // 顺势指标
	OBV       float64 // 能量潮指标
	ATR       float64 // 真实波幅
	WilliamsR float64 // 威廉指标
}

// 函数声明补充
func FetchStockHistory(stockCode, start, end, apiKey string) ([]StockData, []TechnicalIndicator, error) {
	// 腾讯API symbol格式：sh600036、sz000001
	symbol := stockCode
	if len(stockCode) == 6 && stockCode[0] == '6' {
		symbol = "sh" + stockCode
	} else if len(stockCode) == 6 && (stockCode[0] == '0' || stockCode[0] == '3') {
		symbol = "sz" + stockCode
	}
	url := "https://web.ifzq.gtimg.cn/appstock/app/kline/kline?param=" + symbol + ",day,,,320"
	client := &http.Client{}
	req, _ := http.NewRequest("GET", url, nil)
	req.Header.Set("User-Agent", "Mozilla/5.0")
	resp, err := client.Do(req)
	if err != nil {
		return nil, nil, err
	}
	defer resp.Body.Close()
	body, _ := ioutil.ReadAll(resp.Body)
	var data struct {
		Data map[string]struct {
			Day [][]interface{} `json:"day"`
		} `json:"data"`
	}
	err = json.Unmarshal(body, &data)
	if err != nil {
		return nil, nil, err
	}
	var stockData []StockData
	var closes []float64
	for _, v := range data.Data {
		for _, item := range v.Day {
			// item: [日期,开,收,高,低,成交量,...]
			if len(item) < 6 {
				continue
			}
			dateStr := item[0].(string)
			dt, _ := time.Parse("2006-01-02", dateStr)
			open, _ := strconv.ParseFloat(item[1].(string), 64)
			close, _ := strconv.ParseFloat(item[2].(string), 64)
			high, _ := strconv.ParseFloat(item[3].(string), 64)
			low, _ := strconv.ParseFloat(item[4].(string), 64)
			vol, _ := strconv.ParseFloat(item[5].(string), 64)
			stockData = append(stockData, StockData{
				Date:   dt,
				Open:   open,
				Close:  close,
				High:   high,
				Low:    low,
				Volume: vol,
			})
			closes = append(closes, close)
		}
	}
	// 本地计算技术指标
	ma := func(arr []float64, n int, idx int) float64 {
		if idx+1 < n {
			return 0
		}
		sum := 0.0
		for i := idx + 1 - n; i <= idx; i++ {
			sum += arr[i]
		}
		return sum / float64(n)
	}

	// 计算MACD
	calcMACD := func(prices []float64, idx int) (float64, float64, float64) {
		if idx < 25 {
			return 0, 0, 0
		}
		ema12 := 0.0
		ema26 := 0.0
		alpha12 := 2.0 / 13.0
		alpha26 := 2.0 / 27.0

		for i := 0; i <= idx; i++ {
			if i == 0 {
				ema12 = prices[i]
				ema26 = prices[i]
			} else {
				ema12 = alpha12*prices[i] + (1-alpha12)*ema12
				ema26 = alpha26*prices[i] + (1-alpha26)*ema26
			}
		}

		macd := ema12 - ema26
		signal := 0.0
		histogram := macd - signal

		return macd, signal, histogram
	}

	// 计算RSI
	calcRSI := func(prices []float64, period int, idx int) float64 {
		if idx < period {
			return 0
		}
		gains := 0.0
		losses := 0.0
		for i := idx - period + 1; i <= idx; i++ {
			if i > 0 {
				change := prices[i] - prices[i-1]
				if change > 0 {
					gains += change
				} else {
					losses -= change
				}
			}
		}
		avgGain := gains / float64(period)
		avgLoss := losses / float64(period)
		if avgLoss == 0 {
			return 100
		}
		rs := avgGain / avgLoss
		return 100 - (100 / (1 + rs))
	}

	// 计算BOLL
	calcBOLL := func(prices []float64, period int, idx int) (float64, float64, float64) {
		if idx < period-1 {
			return 0, 0, 0
		}
		sum := 0.0
		for i := idx - period + 1; i <= idx; i++ {
			sum += prices[i]
		}
		middle := sum / float64(period)

		variance := 0.0
		for i := idx - period + 1; i <= idx; i++ {
			variance += (prices[i] - middle) * (prices[i] - middle)
		}
		stdDev := math.Sqrt(variance / float64(period))

		upper := middle + 2*stdDev
		lower := middle - 2*stdDev

		return upper, middle, lower
	}

	// 计算成交量均线
	calcVolumeMA := func(volumes []float64, n int, idx int) float64 {
		if idx+1 < n {
			return 0
		}
		sum := 0.0
		for i := idx + 1 - n; i <= idx; i++ {
			sum += volumes[i]
		}
		return sum / float64(n)
	}

	var indicators []TechnicalIndicator
	var volumes []float64
	for _, d := range stockData {
		volumes = append(volumes, d.Volume)
	}

	for i := range stockData {
		// 计算MACD
		macd, signal, histogram := calcMACD(closes, i)

		// 计算RSI
		rsi6 := calcRSI(closes, 6, i)
		rsi12 := calcRSI(closes, 12, i)
		rsi24 := calcRSI(closes, 24, i)

		// 计算BOLL
		bollUpper, bollMiddle, bollLower := calcBOLL(closes, 20, i)

		// 计算成交量均线
		volMA5 := calcVolumeMA(volumes, 5, i)
		volMA10 := calcVolumeMA(volumes, 10, i)
		volMA20 := calcVolumeMA(volumes, 20, i)

		indicators = append(indicators, TechnicalIndicator{
			MA5:   ma(closes, 5, i),
			MA10:  ma(closes, 10, i),
			MA20:  ma(closes, 20, i),
			MA60:  ma(closes, 60, i),
			MA120: ma(closes, 120, i),
			MA250: ma(closes, 250, i),

			MACD:          macd,
			MACDSignal:    signal,
			MACDHistogram: histogram,

			RSI6:  rsi6,
			RSI12: rsi12,
			RSI24: rsi24,

			BOLLUpper:  bollUpper,
			BOLLMiddle: bollMiddle,
			BOLLLower:  bollLower,

			VolumeMA5:  volMA5,
			VolumeMA10: volMA10,
			VolumeMA20: volMA20,
		})
	}
	return stockData, indicators, nil
}

func BuildPrompt(params AnalysisParams) string {
	// 构建分析提示词
	prompt := fmt.Sprintf("请对股票代码 %s 进行智能分析。\n", strings.Join(params.StockCodes, ","))
	prompt += fmt.Sprintf("分析时间范围：%s 至 %s\n", params.Start, params.End)

	// 预测周期
	if len(params.Periods) > 0 {
		prompt += fmt.Sprintf("预测周期：%s\n", strings.Join(params.Periods, "、"))
	}

	// 分析维度
	if len(params.Dims) > 0 {
		prompt += fmt.Sprintf("分析维度：%s\n", strings.Join(params.Dims, "、"))
	}

	// 风险偏好
	if params.Risk != "" {
		prompt += fmt.Sprintf("风险偏好：%s\n", params.Risk)
	}

	// 输出语言
	if params.Lang != "" {
		prompt += fmt.Sprintf("输出语言：%s\n", params.Lang)
	}

	// 新增：详细预测要求
	prompt += "\n【预测要求】\n"

	// 预测类型
	if len(params.PredictionTypes) > 0 {
		prompt += fmt.Sprintf("预测类型：%s\n", strings.Join(params.PredictionTypes, "、"))
	}

	// 具体预测项目
	var predictions []string
	if params.TargetPrice {
		predictions = append(predictions, "目标价位预测")
	}
	if params.StopLoss {
		predictions = append(predictions, "止损位预测")
	}
	if params.TakeProfit {
		predictions = append(predictions, "止盈位预测")
	}
	if params.Volatility {
		predictions = append(predictions, "波动率预测")
	}
	if params.Volume {
		predictions = append(predictions, "成交量预测")
	}
	if params.Probability {
		predictions = append(predictions, "涨跌概率预测")
	}
	if params.RiskLevel {
		predictions = append(predictions, "风险等级评估")
	}
	if params.TrendStrength {
		predictions = append(predictions, "趋势强度预测")
	}
	if params.SupportResistance {
		predictions = append(predictions, "支撑阻力位预测")
	}
	if params.TechnicalSignals {
		predictions = append(predictions, "技术信号预测")
	}
	if params.FundamentalMetrics {
		predictions = append(predictions, "基本面指标预测")
	}
	if params.SentimentScore {
		predictions = append(predictions, "情绪评分预测")
	}
	if params.MarketPosition {
		predictions = append(predictions, "市场定位分析")
	}
	if params.CompetitiveAdvantage {
		predictions = append(predictions, "竞争优势分析")
	}

	if len(predictions) > 0 {
		prompt += fmt.Sprintf("具体预测项目：%s\n", strings.Join(predictions, "、"))
	}

	// 置信度要求
	if params.Confidence {
		prompt += "每个预测结论都需要提供置信度/概率区间\n"
	}

	prompt += "\n请提供详细的技术分析和投资建议，包含上述所有预测项目。"
	// 新增：统一要求用markdown表格输出多周期预测和综合预测结论
	prompt += "\n\n【格式要求】\n1. 多周期预测请用markdown表格输出，表头包含：周期、趋势判断、关键价位、置信度。\n2. 综合预测结论请用markdown表格输出，表头包含：预测项目、预测值/区间、置信度。\n3. 结论部分也请用表格和要点形式输出，便于阅读。"
	return prompt
}

func markdownToHTML(md string) string {
	html := blackfriday.Run([]byte(md))
	return string(html)
}

func replaceImagesWithAbsHTML(md string) string {
	imgRe := regexp.MustCompile(`!\[.*?\]\((.*?)\)`)
	return imgRe.ReplaceAllStringFunc(md, func(s string) string {
		m := imgRe.FindStringSubmatch(s)
		if len(m) < 2 {
			return s
		}
		imgPath := m[1]
		abs, err := filepath.Abs(imgPath)
		if err != nil {
			return s
		}
		return fmt.Sprintf(`<img src="file://%s" style="max-width:100%%;">`, abs)
	})
}

// 新增：将行情数据结构化为表格文本
func FormatStockDataTable(stockData []StockData, indicators []TechnicalIndicator) string {
	if len(stockData) == 0 {
		return ""
	}
	head := "\n【历史行情数据表】\n| 日期 | 开盘 | 收盘 | 最高 | 最低 | 成交量 | MA5 | MA10 | MA20 | MA60 | MA120 | MA250 | MACD | RSI6 | RSI12 | BOLL上轨 | BOLL中轨 | BOLL下轨 |\n|------|------|------|------|------|--------|-----|------|------|------|-------|-------|------|------|-------|----------|----------|----------|\n"
	rows := ""
	for i, d := range stockData {
		if i >= len(indicators) {
			break
		}
		row := fmt.Sprintf("| %s | %.2f | %.2f | %.2f | %.2f | %.0f | %.2f | %.2f | %.2f | %.2f | %.2f | %.2f | %.3f | %.1f | %.1f | %.2f | %.2f | %.2f |\n",
			d.Date.Format("2006-01-02"), d.Open, d.Close, d.High, d.Low, d.Volume,
			indicators[i].MA5, indicators[i].MA10, indicators[i].MA20, indicators[i].MA60,
			indicators[i].MA120, indicators[i].MA250, indicators[i].MACD,
			indicators[i].RSI6, indicators[i].RSI12,
			indicators[i].BOLLUpper, indicators[i].BOLLMiddle, indicators[i].BOLLLower)
		rows += row
		if i > 30 {
			break
		} // 只展示最近30天，防止prompt过长
	}
	return head + rows
}

// 只保留最近N个月的数据（支持动态起止）
func filterRecentDataToDate(stockData []StockData, indicators []TechnicalIndicator, endDate time.Time, months int) ([]StockData, []TechnicalIndicator) {
	if len(stockData) == 0 {
		return stockData, indicators
	}
	cutoff := endDate.AddDate(0, -months, 0)
	idx := 0
	for i, d := range stockData {
		if (d.Date.After(cutoff) || d.Date.Equal(cutoff)) && d.Date.Before(endDate.AddDate(0, 0, 1)) {
			idx = i
			break
		}
	}
	// 只保留截止endDate的半年数据
	var filteredData []StockData
	var filteredInd []TechnicalIndicator
	for i := idx; i < len(stockData); i++ {
		if stockData[i].Date.After(endDate) {
			break
		}
		filteredData = append(filteredData, stockData[i])
		filteredInd = append(filteredInd, indicators[i])
	}
	return filteredData, filteredInd
}

func AnalyzeOne(params AnalysisParams, genFunc func(string, string, string, string, string, bool, bool) (string, error)) AnalysisResult {
	prompt := params.Prompt
	if prompt == "" {
		prompt = BuildPrompt(params)
	}

	// 生成图表
	stockData, indicators, _ := FetchStockHistory(params.StockCodes[0], params.Start, params.End, params.APIKey)
	if len(stockData) > 0 {
		latest := stockData[len(stockData)-1].Date
		// fmt.Fprintf(os.Stderr, "[调试] 实际可用K线最新日期: %s\n", latest.Format("2006-01-02"))
		stockData, indicators = filterRecentDataToDate(stockData, indicators, latest, 12)
	}
	chartPaths, _ := GenerateCharts(params.StockCodes[0], stockData, indicators, "charts")

	// 新增：行情表格注入
	stockTable := FormatStockDataTable(stockData, indicators)
	prompt = stockTable + "\n" + prompt

	report, err := genFunc(params.StockCodes[0], prompt, params.APIKey, "https://api.deepseek.com/v1/chat/completions", params.Model, params.SearchMode, params.HybridSearch)
	if err != nil {
		return AnalysisResult{StockCode: params.StockCodes[0], Err: err}
	}

	// 在报告前插入图表引用
	var chartRefs string
	for _, p := range chartPaths {
		chartRefs += fmt.Sprintf("![图表](%s)\n", p)
	}
	report = chartRefs + "\n" + report

	// 自动修复：确保history目录存在
	os.MkdirAll("history", 0755)

	// 支持多格式导出
	exports := []string{"md"} // 默认md
	if len(params.Output) > 0 {
		exports = params.Output
	}
	var savedFile string
	var writeErr error
	for _, ext := range exports {
		var fname string
		fbase := fmt.Sprintf("%s-%s-%s", params.StockCodes[0], params.End, time.Now().Format("150405"))
		fpath := ""
		// 先处理图片
		reportHTML := replaceImagesWithAbsHTML(report)
		if ext == "md" {
			fname = fbase + ".md"
			fpath = filepath.Join("history", fname)
			err := ioutil.WriteFile(fpath, []byte(report), 0644)
			if err != nil {
				fmt.Fprintf(os.Stderr, "[错误] 写入Markdown文件失败: %s\n", err)
				writeErr = err
			} else {
				savedFile = fname
			}
		} else if ext == "html" {
			fname = fbase + ".html"
			fpath = filepath.Join("history", fname)
			html := "<meta charset=\"utf-8\">\n" + exportCSS + markdownToHTML(reportHTML)
			err := ioutil.WriteFile(fpath, []byte(html), 0644)
			if err != nil {
				fmt.Fprintf(os.Stderr, "[错误] 写入HTML文件失败: %s\n", err)
				writeErr = err
			} else {
				savedFile = fname
			}
		} else if ext == "pdf" {
			fname = fbase + ".pdf"
			fpath = filepath.Join("history", fname)
			htmlPath := fpath + ".tmp.html"
			htmlContent := "<meta charset=\"utf-8\">\n" + exportCSS + markdownToHTML(reportHTML)
			ioutil.WriteFile(htmlPath, []byte(htmlContent), 0644)
			err := htmlToPDF(htmlPath, fpath)
			os.Remove(htmlPath)
			if err != nil {
				fmt.Fprintf(os.Stderr, "[错误] 生成PDF失败: %s\n", err)
				writeErr = err
			} else {
				fmt.Println("[调试] 已写入PDF文件：", fpath)
				savedFile = fname
			}
		}
	}
	if writeErr != nil {
		return AnalysisResult{StockCode: params.StockCodes[0], Report: report, SavedFile: savedFile, Err: writeErr}
	}
	return AnalysisResult{StockCode: params.StockCodes[0], Report: report, SavedFile: savedFile}
}

// printStepBoxStr 返回美观的框包裹字符串（不直接输出）
func getBoxWidth() int {
	w, _, err := term.GetSize(int(os.Stdout.Fd()))
	if err != nil || w <= 0 {
		return 120 // fallback
	}
	bw := int(float64(w) * 0.8)
	if bw < 60 {
		bw = 60
	}
	if bw > 200 {
		bw = 200
	}
	return bw
}

func printStepBoxStr(title string, lines ...string) string {
	width := getBoxWidth()
	titleWidth := len([]rune(title))
	sideLen := (width - 2 - titleWidth) / 2
	top := "┌" + strings.Repeat("─", sideLen) + title + strings.Repeat("─", width-2-titleWidth-sideLen) + "┐\n"
	body := ""
	for _, l := range lines {
		maxContent := width - 2
		lWidth := len([]rune(l))
		if lWidth > maxContent {
			l = l[:maxContent-1] + "…"
			lWidth = len([]rune(l))
		}
		pad := maxContent - lWidth
		if pad < 0 {
			pad = 0
		}
		body += fmt.Sprintf("│%s%s│\n", l, strings.Repeat(" ", pad))
	}
	bottom := "└" + strings.Repeat("─", width-2) + "┘\n"
	return top + body + bottom
}

func htmlToPDF(htmlPath, pdfPath string) error {
	ctx, cancel := chromedp.NewContext(context.Background())
	defer cancel()
	var pdfBuf []byte
	absPath, _ := filepath.Abs(htmlPath)
	fileURL := "file://" + absPath
	err := chromedp.Run(ctx,
		chromedp.Navigate(fileURL),
		chromedp.ActionFunc(func(ctx context.Context) error {
			var err error
			pdfBuf, _, err = page.PrintToPDF().WithPrintBackground(true).Do(ctx)
			return err
		}),
	)
	if err != nil {
		return err
	}
	return os.WriteFile(pdfPath, pdfBuf, 0644)
}

const exportCSS = `
<style>
body { font-family: 'SF Pro', 'Arial', 'Microsoft YaHei', sans-serif; font-size: 15px; margin: 24px; }
h1,h2,h3,h4 { font-weight: bold; margin: 1.2em 0 0.6em 0; }
table { border-collapse: collapse; width: 100%; margin: 1em 0; }
th, td { border: 1px solid #888; padding: 8px 12px; text-align: left; }
th { background: #f0f0f0; }
pre { background: #222; color: #fff; border-radius: 6px; border: 1px solid #444; padding: 14px; font-size: 15px; overflow-x: auto; margin: 1em 0; font-family: monospace; white-space: pre-wrap; }
strong { font-weight: bold; }
img { display: block; margin: 18px auto; max-width: 95%; border-radius: 4px; box-shadow: 0 2px 8px #0001; }
ul,ol { margin: 1em 0 1em 2em; }
code { background: #f5f5f5; color: #c7254e; padding: 2px 4px; border-radius: 4px; }
</style>
`

// 修改 GenerateAIReportWithConfigAndSearch 实现，支持 hybridSearch
func GenerateAIReportWithConfigAndSearch(stock, prompt, apiKey, apiURL, model string, searchMode bool, hybridSearch bool) (string, error) {
	// 构造请求体
	body := map[string]interface{}{
		"model": model,
		"messages": []map[string]string{
			{"role": "system", "content": "你是一个智能股票分析助手。"},
			{"role": "user", "content": prompt},
		},
		"temperature": 0.7,
		"max_tokens":  2000,
	}
	if hybridSearch {
		body["search"] = true // 混合模式，自动融合
	} else if searchMode {
		body["search"] = true // 兼容原有联网搜索
	}
	data, _ := json.Marshal(body)
	client := &http.Client{}
	req, _ := http.NewRequest("POST", apiURL, strings.NewReader(string(data)))
	req.Header.Set("Authorization", "Bearer "+apiKey)
	req.Header.Set("Content-Type", "application/json")
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	respData, _ := ioutil.ReadAll(resp.Body)
	if resp.StatusCode != 200 {
		return "", fmt.Errorf("DeepSeek API 错误: %s", string(respData))
	}
	var result struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}
	err = json.Unmarshal(respData, &result)
	if err != nil {
		return "", err
	}
	if len(result.Choices) == 0 {
		return "", fmt.Errorf("DeepSeek API 无返回内容")
	}
	return result.Choices[0].Message.Content, nil
}

// 字符串宽度填充，兼容中英文
func pad(s string, width int) string {
	w := runewidth.StringWidth(s)
	if w < width {
		return s + strings.Repeat(" ", width-w)
	}
	if w > width {
		rs := []rune(s)
		cut := 0
		for i := range rs {
			if runewidth.StringWidth(string(rs[:i+1])) > width {
				break
			}
			cut = i + 1
		}
		return string(rs[:cut])
	}
	return s
}
