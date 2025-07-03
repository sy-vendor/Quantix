package analysis

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"time"

	"encoding/json"
	"net/http"
	"strconv"

	"github.com/SebastiaanKlippert/go-wkhtmltopdf"
	"github.com/russross/blackfriday/v2"
	"golang.org/x/term"
)

// 类型定义补充

// AnalysisParams 用于传递分析参数
// 可根据main.go和ai.go的实际用法调整字段
// StockData、TechnicalIndicator用于chart.go

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
	Prompt     string // 可选，手动传递prompt
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
}

// ================== 回测模块 =====================

// BacktestTrade 记录一次买卖操作
// Type: "buy" or "sell"
type BacktestTrade struct {
	Date   time.Time
	Type   string
	Price  float64
	Volume float64
	Reason string // 触发原因，如"金叉买入"、"止损卖出"
}

// BacktestResult 汇总回测结果
// 包含收益率、最大回撤、胜率、交易次数、收益曲线等
// 可扩展更多统计项
type BacktestResult struct {
	Trades       []BacktestTrade
	EquityCurve  []float64 // 每日资金曲线
	FinalReturn  float64   // 总收益率
	MaxDrawdown  float64   // 最大回撤
	WinRate      float64   // 胜率
	TotalTrades  int
	TotalWins    int
	TotalLosses  int
	StrategyDesc string                 // 策略描述
	Params       map[string]interface{} // 策略参数
}

// BacktestStrategy 策略接口，可扩展多种买卖逻辑
type BacktestStrategy interface {
	ShouldBuy(idx int, data []StockData, ind []TechnicalIndicator, pos bool, cash float64) (bool, string)
	ShouldSell(idx int, data []StockData, ind []TechnicalIndicator, pos bool, cash float64, buyPrice float64) (bool, string)
	Desc() string
	Params() map[string]interface{}
}

// ================== 策略实现示例 =====================

// MAStrategy 多均线金叉死叉策略+止盈止损
// 支持参数化
type MAStrategy struct {
	ShortMA    int
	LongMA     int
	StopLoss   float64 // 止损百分比，如0.05表示5%
	TakeProfit float64 // 止盈百分比
}

func (s MAStrategy) ShouldBuy(idx int, data []StockData, ind []TechnicalIndicator, pos bool, cash float64) (bool, string) {
	if pos {
		return false, "已持仓"
	}
	if idx < s.LongMA-1 {
		return false, "数据不足"
	}
	shortMA := getMA(ind, idx, s.ShortMA)
	longMA := getMA(ind, idx, s.LongMA)
	if shortMA > 0 && longMA > 0 && shortMA > longMA && getMA(ind, idx-1, s.ShortMA) <= getMA(ind, idx-1, s.LongMA) {
		return true, "金叉买入"
	}
	return false, "无买入信号"
}
func (s MAStrategy) ShouldSell(idx int, data []StockData, ind []TechnicalIndicator, pos bool, cash float64, buyPrice float64) (bool, string) {
	if !pos {
		return false, "未持仓"
	}
	price := data[idx].Close
	if buyPrice > 0 {
		if price <= buyPrice*(1-s.StopLoss) {
			return true, "止损卖出"
		}
		if price >= buyPrice*(1+s.TakeProfit) {
			return true, "止盈卖出"
		}
	}
	shortMA := getMA(ind, idx, s.ShortMA)
	longMA := getMA(ind, idx, s.LongMA)
	if shortMA < longMA && getMA(ind, idx-1, s.ShortMA) >= getMA(ind, idx-1, s.LongMA) {
		return true, "死叉卖出"
	}
	return false, "无卖出信号"
}
func (s MAStrategy) Desc() string {
	return fmt.Sprintf("%d日均线金叉死叉+止盈%.1f%%+止损%.1f%%", s.ShortMA, s.TakeProfit*100, s.StopLoss*100)
}
func (s MAStrategy) Params() map[string]interface{} {
	return map[string]interface{}{"ShortMA": s.ShortMA, "LongMA": s.LongMA, "StopLoss": s.StopLoss, "TakeProfit": s.TakeProfit}
}

func getMA(ind []TechnicalIndicator, idx, n int) float64 {
	if idx < 0 || idx >= len(ind) {
		return 0
	}
	switch n {
	case 5:
		return ind[idx].MA5
	case 10:
		return ind[idx].MA10
	case 20:
		return ind[idx].MA20
	case 60:
		return ind[idx].MA60
	default:
		return 0
	}
}

// ================== 回测引擎 =====================

// RunBacktest 执行回测，支持可扩展策略
func RunBacktest(data []StockData, ind []TechnicalIndicator, strategy BacktestStrategy, initialCash float64) BacktestResult {
	var (
		cash         = initialCash
		pos          = false
		buyPrice     = 0.0
		volume       = 0.0
		trades       []BacktestTrade
		equityCurve  []float64
		wins, losses int
	)
	for i := 0; i < len(data); i++ {
		price := data[i].Close
		if !pos {
			if ok, reason := strategy.ShouldBuy(i, data, ind, pos, cash); ok {
				volume = cash / price
				buyPrice = price
				pos = true
				trades = append(trades, BacktestTrade{Date: data[i].Date, Type: "buy", Price: price, Volume: volume, Reason: reason})
				cash = 0
			}
		} else {
			if ok, reason := strategy.ShouldSell(i, data, ind, pos, cash, buyPrice); ok {
				cash = volume * price
				if price > buyPrice {
					wins++
				} else {
					losses++
				}
				trades = append(trades, BacktestTrade{Date: data[i].Date, Type: "sell", Price: price, Volume: volume, Reason: reason})
				pos = false
				buyPrice = 0
				volume = 0
			}
		}
		// 资金曲线
		if pos {
			equityCurve = append(equityCurve, volume*price)
		} else {
			equityCurve = append(equityCurve, cash)
		}
	}
	// 收盘强平
	if pos {
		price := data[len(data)-1].Close
		cash = volume * price
		trades = append(trades, BacktestTrade{Date: data[len(data)-1].Date, Type: "sell", Price: price, Volume: volume, Reason: "收盘强平"})
		if price > buyPrice {
			wins++
		} else {
			losses++
		}
	}
	finalReturn := (cash - initialCash) / initialCash
	maxDD := calcMaxDrawdown(equityCurve)
	winRate := 0.0
	if wins+losses > 0 {
		winRate = float64(wins) / float64(wins+losses)
	}
	return BacktestResult{
		Trades:       trades,
		EquityCurve:  equityCurve,
		FinalReturn:  finalReturn,
		MaxDrawdown:  maxDD,
		WinRate:      winRate,
		TotalTrades:  len(trades) / 2,
		TotalWins:    wins,
		TotalLosses:  losses,
		StrategyDesc: strategy.Desc(),
		Params:       strategy.Params(),
	}
}

// 计算最大回撤
func calcMaxDrawdown(equity []float64) float64 {
	maxDD := 0.0
	peak := equity[0]
	for _, v := range equity {
		if v > peak {
			peak = v
		}
		drawdown := (peak - v) / peak
		if drawdown > maxDD {
			maxDD = drawdown
		}
	}
	return maxDD
}

// ================== 回测集成入口（供主流程调用） =====================

// RunDefaultBacktest 提供主流程一键调用，策略和参数可后续扩展
func RunDefaultBacktest(data []StockData, ind []TechnicalIndicator) BacktestResult {
	strategy := MAStrategy{ShortMA: 5, LongMA: 20, StopLoss: 0.05, TakeProfit: 0.1}
	return RunBacktest(data, ind, strategy, 100000)
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
	// 本地计算MA5/10/20/60
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
	var indicators []TechnicalIndicator
	for i := range closes {
		indicators = append(indicators, TechnicalIndicator{
			MA5:  ma(closes, 5, i),
			MA10: ma(closes, 10, i),
			MA20: ma(closes, 20, i),
			MA60: ma(closes, 60, i),
		})
	}
	return stockData, indicators, nil
}

func BuildPrompt(params AnalysisParams) string {
	langMap := map[string]string{"zh": "中文", "en": "英文"}
	lang := langMap[params.Lang]
	if lang == "" {
		lang = params.Lang
	}
	prompt := fmt.Sprintf(
		"请用%s分析股票【%s】在区间【%s ~ %s】的走势。\n",
		lang,
		params.StockCodes[0],
		params.Start, params.End,
	)
	if len(params.Periods) > 0 {
		prompt += fmt.Sprintf("分析周期包括：%s。\n", strings.Join(params.Periods, "、"))
	}
	if len(params.Dims) > 0 {
		prompt += fmt.Sprintf("分析维度包括：%s。\n", strings.Join(params.Dims, "、"))
	}
	prompt += "请结合上方K线图、均线图、成交量图，对当前股票的走势、支撑阻力、均线形态、量价关系等进行详细分析，给出趋势判断、操作建议和风险提示。分析内容需包括：\n"
	prompt += "1. K线形态与趋势解读\n"
	prompt += "2. 均线系统排列与金叉死叉分析\n"
	prompt += "3. 成交量变化与量价关系\n"
	prompt += "4. 支撑位与阻力位判断\n"
	prompt += "5. 操作建议（如持有/加仓/减仓/观望等）\n"
	prompt += "6. 风险提示与注意事项\n"
	prompt += "请用专业、客观、结构化的语言输出，适合投资者决策参考。"
	return prompt
}

func markdownToHTML(md string) string {
	html := blackfriday.Run([]byte(md))
	return string(html)
}

func AnalyzeOne(params AnalysisParams, genFunc func(string, string, string, string, string, bool) (string, error)) AnalysisResult {
	prompt := params.Prompt
	if prompt == "" {
		prompt = BuildPrompt(params)
	}

	// 生成图表
	stockData, indicators, _ := FetchStockHistory(params.StockCodes[0], params.Start, params.End, params.APIKey)
	chartPaths, _ := GenerateCharts(params.StockCodes[0], stockData, indicators, "charts")

	// ====== 新增：回测并插入结果到 prompt ======
	backtest := RunDefaultBacktest(stockData, indicators)
	backtestSummary := formatBacktestSummary(backtest)
	prompt = backtestSummary + "\n" + prompt
	// =========================================

	report, err := genFunc(params.StockCodes[0], prompt, params.APIKey, "https://api.deepseek.com/v1/chat/completions", params.Model, params.SearchMode)
	if err != nil {
		return AnalysisResult{StockCode: params.StockCodes[0], Err: err}
	}

	// 在报告前插入图表引用和回测摘要
	var chartRefs string
	for _, p := range chartPaths {
		chartRefs += fmt.Sprintf("![图表](%s)\n", p)
	}
	// 回测摘要美观包裹
	btBox := formatBacktestBox(backtest)
	report = chartRefs + btBox + "\n" + report

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
			html := "<meta charset=\"utf-8\">\n" + markdownToHTML(report)
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
			html := "<meta charset=\"utf-8\">\n" + markdownToHTML(report)
			pdfg, _ := wkhtmltopdf.NewPDFGenerator()
			pdfg.AddPage(wkhtmltopdf.NewPageReader(strings.NewReader(html)))
			pdfg.Dpi.Set(300)
			pdfg.Orientation.Set(wkhtmltopdf.OrientationPortrait)
			pdfg.PageSize.Set(wkhtmltopdf.PageSizeA4)
			pdfg.NoCollate.Set(false)
			pdfg.Grayscale.Set(false)
			pdfg.MarginLeft.Set(10)
			pdfg.MarginRight.Set(10)
			pdfg.MarginTop.Set(10)
			pdfg.MarginBottom.Set(10)
			err := pdfg.Create()
			if err != nil {
				fmt.Fprintf(os.Stderr, "[错误] 生成PDF失败: %s\n", err)
				writeErr = err
				continue
			}
			err = pdfg.WriteFile(fpath)
			if err != nil {
				fmt.Fprintf(os.Stderr, "[错误] 写入PDF文件失败: %s\n", err)
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

// formatBacktestSummary 生成回测摘要，供 prompt 注入
func formatBacktestSummary(bt BacktestResult) string {
	return fmt.Sprintf("【回测结果】\n策略：%s\n总收益率：%.2f%%\n最大回撤：%.2f%%\n胜率：%.1f%%\n交易次数：%d\n", bt.StrategyDesc, bt.FinalReturn*100, bt.MaxDrawdown*100, bt.WinRate*100, bt.TotalTrades)
}

// formatBacktestBox 美观输出回测结果（终端框包裹）
func formatBacktestBox(bt BacktestResult) string {
	lines := []string{
		fmt.Sprintf("策略：%s", bt.StrategyDesc),
		fmt.Sprintf("总收益率：%.2f%%", bt.FinalReturn*100),
		fmt.Sprintf("最大回撤：%.2f%%", bt.MaxDrawdown*100),
		fmt.Sprintf("胜率：%.1f%%", bt.WinRate*100),
		fmt.Sprintf("交易次数：%d", bt.TotalTrades),
	}
	if len(bt.Trades) > 0 {
		lines = append(lines, "--- 交易明细 ---")
		for _, t := range bt.Trades {
			lines = append(lines, fmt.Sprintf("%s %s %.2f x %.2f [%s]", t.Date.Format("2006-01-02"), t.Type, t.Price, t.Volume, t.Reason))
		}
	}
	return printStepBoxStr("回测结果", lines...)
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
