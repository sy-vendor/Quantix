package analysis

import (
	"fmt"
	"io/ioutil"
	"math"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"encoding/json"
	"net/http"
	"strconv"

	"context"

	"google.golang.org/genai"

	"regexp"

	"github.com/chromedp/cdproto/page"
	"github.com/chromedp/chromedp"
	"github.com/russross/blackfriday/v2"
)

// 类型定义补充

// AnalysisParams 用于传递分析参数
// 可根据main.go和ai.go的实际用法调整字段
// StockData、TechnicalIndicator用于chart.go

type AnalysisParams struct {
	LLMType      string // 新增：大模型类型 deepseek/gmini
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

	// 新增：回测参数
	BacktestParams *BacktestParams // 回测参数，允许为nil
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

	// 新增：更多高级技术指标
	StochK       float64 // 随机指标K值
	StochD       float64 // 随机指标D值
	ADX          float64 // 平均趋向指数
	ParabolicSAR float64 // 抛物线转向指标
	Ichimoku     struct {
		TenkanSen   float64 // 转换线
		KijunSen    float64 // 基准线
		SenkouSpanA float64 // 先行带A
		SenkouSpanB float64 // 先行带B
		ChikouSpan  float64 // 滞后线
	}
	PivotPoints struct {
		PP float64 // 轴心点
		R1 float64 // 阻力位1
		R2 float64 // 阻力位2
		R3 float64 // 阻力位3
		S1 float64 // 支撑位1
		S2 float64 // 支撑位2
		S3 float64 // 支撑位3
	}
}

// 函数声明补充
func FetchStockHistory(stockCode, start, end, apiKey string) ([]StockData, []TechnicalIndicator, error) {
	// 尝试多个数据源，确保数据准确性
	var stockData []StockData
	var err error

	// 数据源优先级：1. 雪球API 2. 网易API 3. 腾讯API
	dataSources := []struct {
		name string
		fn   func(string) ([]StockData, error)
	}{
		{"雪球API", fetchFromXueqiu},
		{"网易API", fetchFromNetEase},
		{"腾讯API", fetchFromTencent},
	}

	for _, source := range dataSources {
		fmt.Printf("[数据源] 尝试从 %s 获取 %s 的历史数据...\n", source.name, stockCode)
		stockData, err = source.fn(stockCode)
		if err == nil && len(stockData) > 0 {
			fmt.Printf("[数据源] ✓ 成功从 %s 获取 %d 条数据\n", source.name, len(stockData))
			break
		}
		fmt.Printf("[数据源] ✗ %s 获取失败: %v\n", source.name, err)
	}

	if len(stockData) == 0 {
		return nil, nil, fmt.Errorf("所有数据源都获取失败")
	}

	// 数据验证：检查价格合理性
	stockData = validateAndFilterData(stockData, stockCode)

	// 按日期排序
	sort.Slice(stockData, func(i, j int) bool {
		return stockData[i].Date.Before(stockData[j].Date)
	})

	// 计算技术指标
	indicators := calculateTechnicalIndicators(stockData)

	return stockData, indicators, nil
}

// 腾讯API数据源
func fetchFromTencent(stockCode string) ([]StockData, error) {
	// 腾讯API symbol格式：sh600036、sz000001
	symbol := stockCode
	if len(stockCode) == 6 && stockCode[0] == '6' {
		symbol = "sh" + stockCode
	} else if len(stockCode) == 6 && (stockCode[0] == '0' || stockCode[0] == '3') {
		symbol = "sz" + stockCode
	}

	url := "https://web.ifzq.gtimg.cn/appstock/app/kline/kline?param=" + symbol + ",day,,,320"
	client := &http.Client{Timeout: 10 * time.Second}
	req, _ := http.NewRequest("GET", url, nil)
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36")
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
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
		return nil, err
	}

	var stockData []StockData
	for _, v := range data.Data {
		for _, item := range v.Day {
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
		}
	}
	return stockData, nil
}

// 网易API数据源
func fetchFromNetEase(stockCode string) ([]StockData, error) {
	// 网易API格式：0.000001（深市）、1.600036（沪市）
	symbol := stockCode
	if len(stockCode) == 6 && stockCode[0] == '6' {
		symbol = "1." + stockCode
	} else if len(stockCode) == 6 && (stockCode[0] == '0' || stockCode[0] == '3') {
		symbol = "0." + stockCode
	}

	url := fmt.Sprintf("http://api.money.126.net/data/feed/%s/history", symbol)
	client := &http.Client{Timeout: 10 * time.Second}
	req, _ := http.NewRequest("GET", url, nil)
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36")
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, _ := ioutil.ReadAll(resp.Body)
	var data map[string]struct {
		Data [][]float64 `json:"data"`
	}
	err = json.Unmarshal(body, &data)
	if err != nil {
		return nil, err
	}

	var stockData []StockData
	for _, v := range data {
		for _, item := range v.Data {
			if len(item) < 6 {
				continue
			}
			// 网易数据格式：[时间戳, 开盘, 最高, 最低, 收盘, 成交量]
			timestamp := int64(item[0])
			dt := time.Unix(timestamp/1000, 0)

			stockData = append(stockData, StockData{
				Date:   dt,
				Open:   item[1],
				High:   item[2],
				Low:    item[3],
				Close:  item[4],
				Volume: item[5],
			})
		}
	}
	return stockData, nil
}

// 雪球API数据源
func fetchFromXueqiu(stockCode string) ([]StockData, error) {
	// 雪球API格式：SZ000001、SH600036
	symbol := stockCode
	if len(stockCode) == 6 && stockCode[0] == '6' {
		symbol = "SH" + stockCode
	} else if len(stockCode) == 6 && (stockCode[0] == '0' || stockCode[0] == '3') {
		symbol = "SZ" + stockCode
	}

	// 获取当前时间戳（雪球API不需要时间参数，但保留注释说明）
	// now := time.Now()
	// endTime := now.UnixNano() / 1e6
	// startTime := now.AddDate(0, -1, 0).UnixNano() / 1e6 // 最近1个月

	url := fmt.Sprintf("https://stock.xueqiu.com/v5/stock/chart/kline.json?symbol=%s&period=day&type=before&count=320&indicator=kline", symbol)
	client := &http.Client{Timeout: 10 * time.Second}
	req, _ := http.NewRequest("GET", url, nil)
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36")
	req.Header.Set("Referer", "https://xueqiu.com")

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, _ := ioutil.ReadAll(resp.Body)
	var data struct {
		Data struct {
			Item [][]interface{} `json:"item"`
		} `json:"data"`
	}
	err = json.Unmarshal(body, &data)
	if err != nil {
		return nil, err
	}

	var stockData []StockData
	for _, item := range data.Data.Item {
		if len(item) < 6 {
			continue
		}
		// 雪球数据格式：[时间戳, 成交量, 开盘, 最高, 最低, 收盘, ...]
		timestamp := int64(item[0].(float64))
		dt := time.Unix(timestamp/1000, 0)
		volume := item[1].(float64)
		open := item[2].(float64)
		high := item[3].(float64)
		low := item[4].(float64)
		close := item[5].(float64)

		stockData = append(stockData, StockData{
			Date:   dt,
			Open:   open,
			High:   high,
			Low:    low,
			Close:  close,
			Volume: volume,
		})
	}
	return stockData, nil
}

// 数据验证和过滤
func validateAndFilterData(stockData []StockData, stockCode string) []StockData {
	var validData []StockData

	// 价格合理性检查
	for _, data := range stockData {
		// 基本价格检查
		if data.Open <= 0 || data.Close <= 0 || data.High <= 0 || data.Low <= 0 {
			continue
		}

		// 价格逻辑检查
		if data.High < data.Open || data.High < data.Close || data.High < data.Low {
			continue
		}
		if data.Low > data.Open || data.Low > data.Close || data.Low > data.High {
			continue
		}

		// 价格范围检查（防止异常值）
		if data.Close > 10000 || data.Close < 0.01 {
			continue
		}

		// 成交量检查
		if data.Volume < 0 {
			continue
		}

		validData = append(validData, data)
	}

	// 如果过滤后数据太少，记录警告
	if len(validData) < int(float64(len(stockData))*0.8) {
		fmt.Printf("[数据验证] ⚠️  %s 数据过滤较多：原始%d条，有效%d条\n",
			stockCode, len(stockData), len(validData))
	}

	return validData
}

// 计算技术指标
func calculateTechnicalIndicators(stockData []StockData) []TechnicalIndicator {
	if len(stockData) == 0 {
		return nil
	}

	var closes []float64
	var volumes []float64
	for _, d := range stockData {
		closes = append(closes, d.Close)
		volumes = append(volumes, d.Volume)
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
	return indicators
}

func BuildPrompt(params AnalysisParams) string {
	// 判断是否联网/混合模式
	isOnline := params.SearchMode || params.HybridSearch
	prompt := ""
	if isOnline {
		prompt += fmt.Sprintf(`请联网获取股票%s的最新股价、最新公告和新闻，分析时以最新联网数据为准。

【重要】数据验证要求：
1. 请联网查询该股票的最新收盘价，并与本地K线数据对比
2. 如果最新联网价格与本地数据差异超过5%，请以联网数据为准
3. 在报告开头明确标注：
   - 最新联网价格：XX.XX元（查询时间：YYYY-MM-DD HH:MM）
   - 本地数据最新价格：XX.XX元（日期：YYYY-MM-DD）
   - 数据差异：+/-X.XX元（X.XX%）
4. 如果发现价格异常（如超过1000元或低于0.01元），请重新查询并标注"数据异常，已重新验证"

请确保获取的是真实准确的股价数据，不要使用过时或错误的价格信息。`, strings.Join(params.StockCodes, ","))
	} else {
		prompt += fmt.Sprintf("请对股票代码 %s 进行智能分析。\n", strings.Join(params.StockCodes, ","))
	}
	prompt += fmt.Sprintf("分析时间范围：%s 至 %s\n", params.Start, params.End)
	if len(params.Periods) > 0 {
		prompt += fmt.Sprintf("预测周期：%s\n", strings.Join(params.Periods, ","))
	}
	if len(params.Dims) > 0 {
		prompt += fmt.Sprintf("分析维度：%s\n", strings.Join(params.Dims, "、"))
	}
	if params.Risk != "" {
		prompt += fmt.Sprintf("风险偏好：%s\n", params.Risk)
	}
	if params.Lang != "" {
		prompt += fmt.Sprintf("输出语言：%s\n", params.Lang)
	}
	prompt += "\n【预测要求】\n"
	if len(params.PredictionTypes) > 0 {
		prompt += fmt.Sprintf("预测类型：%s\n", strings.Join(params.PredictionTypes, "、"))
	}
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
	if params.Confidence {
		prompt += "每个预测结论都需要提供置信度/概率区间\n"
	}
	prompt += "\n请提供详细的技术分析和投资建议，包含上述所有预测项目。"
	prompt += "\n\n【格式要求】\n1. 多周期预测请用markdown表格输出，表头包含：周期、趋势判断、关键价位、置信度、主要驱动因素/理由。\n2. 综合预测结论请用markdown表格输出，表头包含：预测项目、预测值/区间、置信度、主要驱动因素/理由。\n3. 若某项预测不适用或数据不足，请在表格中注明'数据不足'或'-'。\n4. 结论部分请分为'主要结论'、'风险提示'、'操作建议'三块，分别用表格或要点输出。"
	// 智能异常检测与提示
	prompt += "\n5. 请对比最新股价与历史K线（如最近30日均价、最高价、最低价），如最新价与历史均值/区间差异超过10%，请在报告开头高亮提示'行情异动'，并简要分析可能原因。"
	prompt += "\n6. 如果多周期预测或综合结论中某项置信度低于60%，请在该行或结论部分自动加'风险提示'（如'预测不确定性较高，请谨慎参考'）。"
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

// 新增：回测结果 markdown 表格
func FormatBacktestTable(btParams BacktestParams, btResult BacktestResult) string {
	head := "\n【策略回测结果】\n| 策略类型 | 参数 | 总收益率 | 胜率 | 最大回撤 | 盈亏比 | 交易次数 |\n|---|---|---|---|---|---|---|\n"
	paramStr := fmt.Sprintf("%+v", btParams)
	row := fmt.Sprintf("| %s | %s | %.2f%% | %.2f%% | %.2f%% | %.2f | %d |\n",
		btParams.StrategyType, paramStr, btResult.TotalReturn*100, btResult.WinRate*100, btResult.MaxDrawdown*100, btResult.ProfitFactor, btResult.Trades)
	return head + row
}

// 新增：风险指标 markdown 表格
func FormatRiskTable(risk RiskMetrics) string {
	head := "\n【风险指标】\n| 波动率 | 最大回撤 | 夏普比率 | VaR(95%) | 风险等级 | 风险评分 |\n|---|---|---|---|---|---|\n"
	row := fmt.Sprintf("| %.4f | %.2f%% | %.2f | %.4f | %s | %.1f |\n",
		risk.Volatility, risk.MaxDrawdown*100, risk.SharpeRatio, risk.VaR95, risk.RiskLevel, risk.RiskScore)
	return head + row
}

// 新增：回测结果 HTML 表格
func FormatBacktestTableHTML(btParams BacktestParams, btResult BacktestResult) string {
	return fmt.Sprintf(`
<h3>【策略回测结果】</h3>
<table>
<tr><th>策略类型</th><th>参数</th><th>总收益率</th><th>胜率</th><th>最大回撤</th><th>盈亏比</th><th>交易次数</th></tr>
<tr>
<td>%s</td>
<td>%+v</td>
<td>%.2f%%</td>
<td>%.2f%%</td>
<td>%.2f%%</td>
<td>%.2f</td>
<td>%d</td>
</tr>
</table>
`, btParams.StrategyType, btParams, btResult.TotalReturn*100, btResult.WinRate*100, btResult.MaxDrawdown*100, btResult.ProfitFactor, btResult.Trades)
}

// 新增：风险指标 HTML 表格
func FormatRiskTableHTML(risk RiskMetrics) string {
	return fmt.Sprintf(`
<h3>【风险指标】</h3>
<table>
<tr><th>波动率</th><th>最大回撤</th><th>夏普比率</th><th>VaR(95%%)</th><th>风险等级</th><th>风险评分</th></tr>
<tr>
<td>%.4f</td>
<td>%.2f%%</td>
<td>%.2f</td>
<td>%.4f</td>
<td>%s</td>
<td>%.1f</td>
</tr>
</table>
`, risk.Volatility, risk.MaxDrawdown*100, risk.SharpeRatio, risk.VaR95, risk.RiskLevel, risk.RiskScore)
}

func AnalyzeOne(params AnalysisParams, genFunc func(string, string, string, string, string, bool, bool) (string, error)) AnalysisResult {
	prompt := params.Prompt
	if prompt == "" {
		prompt = BuildPrompt(params)
	}

	// 自动插入当前系统日期声明，防止AI用自身认知时间
	now := time.Now().Format("2006-01-02")
	dateNotice := fmt.Sprintf(
		"\n【重要提示】本系统优先使用 DeepSeek 联网模式获取最新行情和分析，只有在联网失败时才尝试本地数据源。请严格以当前分析时间 %s 为准，禁止引用AI自身认知的时间或任何与本地参数不符的时间信息。若分析区间超出数据范围，请直接说明“数据不足”，不要虚构或假设当前时间。\n",
		now)
	prompt = dateNotice + prompt + "\n请再次确认，所有分析均以当前分析时间为准，不要引用AI自身时间认知。\n"

	useHTML := false
	for _, o := range params.Output {
		if o == "html" || o == "pdf" {
			useHTML = true
			break
		}
	}

	var report string
	var err error
	var savedFile string
	var chartRefs, riskTable, backtestTable string

	var stockData []StockData
	var indicators []TechnicalIndicator
	var chartPaths []string

	if params.LLMType == "Gemini" {
		report, err = GenerateGeminiReportWithConfigAndSearch(params.Model, params.APIKey, prompt, params.SearchMode)
	} else if params.LLMType == "gmini" {
		// 伪实现：调用 gmini API
		report, err = GenerateGminiReportWithConfigAndSearch(params)
	} else if params.SearchMode || params.HybridSearch {
		// DeepSeek 联网/混合模式
		stockData, indicators, _ = FetchStockHistory(params.StockCodes[0], params.Start, params.End, params.APIKey)
		if len(stockData) > 0 {
			latest := stockData[len(stockData)-1].Date
			stockData, indicators = filterRecentDataToDate(stockData, indicators, latest, 12)
			chartPaths, _ = GenerateCharts(params.StockCodes[0], stockData, indicators, "charts")
		}
		report, err = genFunc(params.StockCodes[0], prompt, params.APIKey, "https://api.deepseek.com/v1/chat/completions", params.Model, params.SearchMode, params.HybridSearch)
	} else {
		// DeepSeek 本地数据模式
		stockData, indicators, fetchErr := FetchStockHistory(params.StockCodes[0], params.Start, params.End, params.APIKey)
		if len(stockData) > 0 {
			latest := stockData[len(stockData)-1].Date
			stockData, indicators = filterRecentDataToDate(stockData, indicators, latest, 12)
			chartPaths, _ = GenerateCharts(params.StockCodes[0], stockData, indicators, "charts")
		}
		if len(stockData) == 0 && fetchErr != nil {
			params.SearchMode = true
			params.HybridSearch = false
			prompt = "[提示] DeepSeek 联网模式优先，本地数据源全部获取失败，已自动继续使用 DeepSeek 联网分析。\n" + BuildPrompt(params)
			stockData, indicators, _ = FetchStockHistory(params.StockCodes[0], params.Start, params.End, params.APIKey)
			if len(stockData) > 0 {
				latest := stockData[len(stockData)-1].Date
				stockData, indicators = filterRecentDataToDate(stockData, indicators, latest, 12)
				chartPaths, _ = GenerateCharts(params.StockCodes[0], stockData, indicators, "charts")
			}
			report, err = genFunc(params.StockCodes[0], prompt, params.APIKey, "https://api.deepseek.com/v1/chat/completions", params.Model, true, false)
		} else {
			riskTable = ""
			if len(stockData) > 0 {
				risk := CalculateRiskMetrics(stockData)
				if useHTML {
					riskTable = FormatRiskTableHTML(risk)
				} else {
					riskTable = FormatRiskTable(risk)
				}
			}
			stockTable := FormatStockDataTable(stockData, indicators)
			prompt = stockTable + "\n" + prompt
			report, err = genFunc(params.StockCodes[0], prompt, params.APIKey, "https://api.deepseek.com/v1/chat/completions", params.Model, false, false)
		}
	}
	if err != nil {
		return AnalysisResult{StockCode: params.StockCodes[0], Err: err}
	}

	// ====== 图表引用、风险、回测表格统一拼接 ======
	if len(chartPaths) > 0 {
		for _, p := range chartPaths {
			chartRefs += fmt.Sprintf("![图表](%s)\n", p)
		}
	}
	if riskTable == "" && len(stockData) > 0 {
		risk := CalculateRiskMetrics(stockData)
		if useHTML {
			riskTable = FormatRiskTableHTML(risk)
		} else {
			riskTable = FormatRiskTable(risk)
		}
	}
	var btParams BacktestParams
	if params.BacktestParams != nil {
		btParams = *params.BacktestParams
	} else {
		btParams = BacktestParams{
			StrategyType:   "ma_cross",
			FastMAPeriod:   5,
			SlowMAPeriod:   20,
			BreakoutPeriod: 10,
			RSIPeriod:      14,
			RSIOverbought:  70,
			RSIOversold:    30,
			StopLoss:       0.05,
			TakeProfit:     0.10,
			InitialCash:    100000,
		}
	}
	btResult := BacktestStrategy(stockData, btParams)
	if useHTML {
		backtestTable = FormatBacktestTableHTML(btParams, btResult)
	} else {
		backtestTable = FormatBacktestTable(btParams, btResult)
	}

	finalReport := chartRefs + riskTable + backtestTable + report

	// ====== 恢复多格式导出逻辑 ======
	os.MkdirAll("history", 0755)
	exports := []string{"md"}
	if len(params.Output) > 0 {
		exports = params.Output
	}
	var writeErr error
	for _, ext := range exports {
		var fname string
		fbase := fmt.Sprintf("%s-%s-%s", params.StockCodes[0], params.End, time.Now().Format("150405"))
		fpath := ""
		reportHTML := replaceImagesWithAbsHTML(finalReport)
		if ext == "md" {
			fname = fbase + ".md"
			fpath = filepath.Join("history", fname)
			err := ioutil.WriteFile(fpath, []byte(finalReport), 0644)
			if err != nil {
				fmt.Fprintf(os.Stderr, "[错误] 写入Markdown文件失败: %s\n", err)
				writeErr = err
			} else {
				savedFile = fname
			}
		} else if ext == "html" {
			fname = fbase + ".html"
			fpath = filepath.Join("history", fname)
			html := "<meta charset=\"utf-8\">\n" + exportCSS + markdownToHTML(convertMarkdownTablesToHTML(reportHTML))
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
			htmlContent := "<meta charset=\"utf-8\">\n" + exportCSS + markdownToHTML(convertMarkdownTablesToHTML(reportHTML))
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
		return AnalysisResult{StockCode: params.StockCodes[0], Report: finalReport, SavedFile: savedFile, Err: writeErr}
	}
	return AnalysisResult{StockCode: params.StockCodes[0], Report: finalReport, SavedFile: savedFile}
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

// 新增：将 markdown 表格转换为 HTML 表格
func convertMarkdownTablesToHTML(md string) string {
	re := regexp.MustCompile(`(?ms)(\|.+\|\n\|[-| ]+\|\n(?:\|.*\|\n?)+)`)
	return re.ReplaceAllStringFunc(md, func(table string) string {
		lines := strings.Split(strings.TrimSpace(table), "\n")
		if len(lines) < 2 {
			return table
		}
		var html strings.Builder
		html.WriteString("<table>\n")
		// 表头
		cols := strings.Split(lines[0], "|")
		if len(cols) < 3 {
			return table // 防止越界
		}
		html.WriteString("<tr>")
		for _, c := range cols[1 : len(cols)-1] {
			html.WriteString("<th>" + strings.TrimSpace(c) + "</th>")
		}
		html.WriteString("</tr>\n")
		// 表体
		for _, l := range lines[2:] {
			cells := strings.Split(l, "|")
			if len(cells) < 3 {
				continue // 跳过异常行
			}
			html.WriteString("<tr>")
			for _, c := range cells[1 : len(cells)-1] {
				html.WriteString("<td>" + strings.TrimSpace(c) + "</td>")
			}
			html.WriteString("</tr>\n")
		}
		html.WriteString("</table>\n")
		return html.String()
	})
}

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

// 伪实现：gmini大模型API调用
func GenerateGminiReportWithConfigAndSearch(params AnalysisParams) (string, error) {
	// 这里写gmini的API调用逻辑，暂时返回伪内容
	return "[gmini大模型分析报告]（此处为gmini模型返回的内容）", nil
}

// Gemini大模型API调用，支持 deepSearch
func GenerateGeminiReportWithConfigAndSearch(model, apiKey, prompt string, deepSearch bool) (string, error) {
	ctx := context.Background()
	client, err := genai.NewClient(ctx, &genai.ClientConfig{
		APIKey:  apiKey,
		Backend: genai.BackendGeminiAPI,
	})
	if err != nil {
		return "", err
	}
	contents := []*genai.Content{
		{Parts: []*genai.Part{
			{Text: prompt},
		}},
	}
	var config *genai.GenerateContentConfig
	if deepSearch {
		config = &genai.GenerateContentConfig{
			Tools: []*genai.Tool{
				{Retrieval: &genai.Retrieval{}},
			},
		}
	}
	resp, err := client.Models.GenerateContent(ctx, model, contents, config)
	if err != nil {
		return "", err
	}
	if len(resp.Candidates) == 0 || resp.Candidates[0].Content == nil || len(resp.Candidates[0].Content.Parts) == 0 {
		return "", nil
	}
	for _, part := range resp.Candidates[0].Content.Parts {
		if part.Text != "" {
			return part.Text, nil
		}
	}
	return "", nil
}
