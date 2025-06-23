package analysis

import (
	"Quantix/data"
	"fmt"
	"html/template"
	"os"
	"path/filepath"
	"time"
)

// 图表数据结构
type ChartData struct {
	StockCode      string
	StartDate      string
	EndDate        string
	Klines         []KlineData
	BacktestData   []BacktestPoint
	Factors        Factors
	Prediction     Prediction
	BacktestResult BacktestResult
}

type KlineData struct {
	Date   string  `json:"date"`
	Open   float64 `json:"open"`
	High   float64 `json:"high"`
	Low    float64 `json:"low"`
	Close  float64 `json:"close"`
	Volume int64   `json:"volume"`
}

type BacktestPoint struct {
	Date    string  `json:"date"`
	Capital float64 `json:"capital"`
	Shares  int     `json:"shares"`
	Price   float64 `json:"price"`
	Action  string  `json:"action"`
}

// 生成K线图HTML
func GenerateKlineChart(klines []data.Kline, stockCode, startDate, endDate string) error {
	// 转换数据格式
	var klineData []KlineData
	for _, k := range klines {
		klineData = append(klineData, KlineData{
			Date:   k.Date.Format("2006-01-02"),
			Open:   k.Open,
			High:   k.High,
			Low:    k.Low,
			Close:  k.Close,
			Volume: k.Volume,
		})
	}

	// 创建图表数据
	chartData := ChartData{
		StockCode: stockCode,
		StartDate: startDate,
		EndDate:   endDate,
		Klines:    klineData,
	}

	// 生成HTML文件
	return generateHTMLChart(chartData, "kline")
}

// 生成回测曲线图HTML
func GenerateBacktestChart(klines []data.Kline, backtestResult BacktestResult, stockCode, startDate, endDate string) error {
	// 转换回测数据
	var backtestPoints []BacktestPoint
	initialCapital := backtestResult.InitialCapital

	// 添加初始点
	backtestPoints = append(backtestPoints, BacktestPoint{
		Date:    klines[0].Date.Format("2006-01-02"),
		Capital: initialCapital,
		Shares:  0,
		Price:   klines[0].Close,
		Action:  "初始资金",
	})

	// 添加交易点
	for _, trade := range backtestResult.TradeHistory {
		backtestPoints = append(backtestPoints, BacktestPoint{
			Date:    trade.Date,
			Capital: trade.Capital,
			Shares:  trade.Shares,
			Price:   trade.Price,
			Action:  trade.Type,
		})
	}

	// 创建图表数据
	chartData := ChartData{
		StockCode:      stockCode,
		StartDate:      startDate,
		EndDate:        endDate,
		Klines:         convertKlines(klines),
		BacktestData:   backtestPoints,
		BacktestResult: backtestResult,
	}

	// 生成HTML文件
	return generateHTMLChart(chartData, "backtest")
}

// 生成综合分析图表
func GenerateAnalysisChart(klines []data.Kline, factors Factors, prediction Prediction, backtestResult BacktestResult, stockCode, startDate, endDate string) error {
	// 转换回测数据
	var backtestPoints []BacktestPoint
	if len(klines) > 0 {
		initialCapital := backtestResult.InitialCapital
		backtestPoints = append(backtestPoints, BacktestPoint{
			Date:    klines[0].Date.Format("2006-01-02"),
			Capital: initialCapital,
		})
		for _, trade := range backtestResult.TradeHistory {
			backtestPoints = append(backtestPoints, BacktestPoint{
				Date:    trade.Date,
				Capital: trade.Capital,
			})
		}
	}

	chartData := ChartData{
		StockCode:      stockCode,
		StartDate:      startDate,
		EndDate:        endDate,
		Klines:         convertKlines(klines),
		BacktestData:   backtestPoints,
		Factors:        factors,
		Prediction:     prediction,
		BacktestResult: backtestResult,
	}

	return generateHTMLChart(chartData, "analysis")
}

// 转换K线数据格式
func convertKlines(klines []data.Kline) []KlineData {
	var result []KlineData
	for _, k := range klines {
		result = append(result, KlineData{
			Date:   k.Date.Format("2006-01-02"),
			Open:   k.Open,
			High:   k.High,
			Low:    k.Low,
			Close:  k.Close,
			Volume: k.Volume,
		})
	}
	return result
}

// 生成HTML图表文件
func generateHTMLChart(data ChartData, chartType string) error {
	// 创建charts目录
	chartsDir := "charts"
	if err := os.MkdirAll(chartsDir, 0755); err != nil {
		return err
	}

	// 生成文件名
	timestamp := time.Now().Format("20060102_150405")
	filename := fmt.Sprintf("%s_%s_%s.html", data.StockCode, chartType, timestamp)
	filepath := filepath.Join(chartsDir, filename)

	// 根据图表类型选择模板
	var templateStr string
	switch chartType {
	case "kline":
		templateStr = klineChartTemplate
	case "backtest":
		templateStr = backtestChartTemplate
	case "analysis":
		templateStr = analysisChartTemplate
	default:
		return fmt.Errorf("未知的图表类型: %s", chartType)
	}

	// 创建模板函数映射
	funcMap := template.FuncMap{
		"getMaxPrice": getMaxPrice,
		"getMinPrice": getMinPrice,
		"getAvgPrice": getAvgPrice,
		"mul":         mul,
	}

	// 解析模板
	tmpl, err := template.New("chart").Funcs(funcMap).Parse(templateStr)
	if err != nil {
		return err
	}

	// 创建文件
	file, err := os.Create(filepath)
	if err != nil {
		return err
	}
	defer file.Close()

	// 执行模板
	if err := tmpl.Execute(file, data); err != nil {
		return err
	}

	fmt.Printf("图表已生成: %s\n", filepath)
	return nil
}

// 模板辅助函数
func getMaxPrice(klines []KlineData) float64 {
	if len(klines) == 0 {
		return 0
	}
	max := klines[0].High
	for _, k := range klines {
		if k.High > max {
			max = k.High
		}
	}
	return max
}

func getMinPrice(klines []KlineData) float64 {
	if len(klines) == 0 {
		return 0
	}
	min := klines[0].Low
	for _, k := range klines {
		if k.Low < min {
			min = k.Low
		}
	}
	return min
}

func getAvgPrice(klines []KlineData) float64 {
	if len(klines) == 0 {
		return 0
	}
	sum := 0.0
	for _, k := range klines {
		sum += k.Close
	}
	return sum / float64(len(klines))
}

func mul(a, b float64) float64 {
	return a * b
}

// K线图HTML模板
const klineChartTemplate = `
<!DOCTYPE html>
<html>
<head>
    <title>{{.StockCode}} K线图</title>
    <script src="https://cdn.jsdelivr.net/npm/chart.js"></script>
    <style>
        body { font-family: Arial, sans-serif; margin: 20px; }
        .container { max-width: 1200px; margin: 0 auto; }
        .header { text-align: center; margin-bottom: 20px; }
        .chart-container { position: relative; height: 500px; margin-bottom: 30px; }
        .info-panel { background: #f5f5f5; padding: 15px; border-radius: 5px; }
        .info-grid { display: grid; grid-template-columns: repeat(auto-fit, minmax(200px, 1fr)); gap: 10px; }
    </style>
</head>
<body>
    <div class="container">
        <div class="header">
            <h1>{{.StockCode}} K线图</h1>
            <p>期间: {{.StartDate}} 至 {{.EndDate}}</p>
        </div>
        
        <div class="chart-container">
            <canvas id="klineChart"></canvas>
        </div>
        
        <div class="info-panel">
            <h3>数据统计</h3>
            <div class="info-grid">
                <div>数据点数: {{len .Klines}}</div>
                <div>最高价: {{getMaxPrice .Klines}}</div>
                <div>最低价: {{getMinPrice .Klines}}</div>
                <div>平均价: {{getAvgPrice .Klines}}</div>
            </div>
        </div>
    </div>

    <script>
        const ctx = document.getElementById('klineChart').getContext('2d');
        
        const data = {
            labels: [{{range .Klines}}'{{.Date}}',{{end}}],
            datasets: [{
                label: '收盘价',
                data: [{{range .Klines}}{{.Close}},{{end}}],
                borderColor: 'rgb(75, 192, 192)',
                backgroundColor: 'rgba(75, 192, 192, 0.2)',
                tension: 0.1
            }]
        };
        
        new Chart(ctx, {
            type: 'line',
            data: data,
            options: {
                responsive: true,
                maintainAspectRatio: false,
                scales: {
                    y: {
                        beginAtZero: false
                    }
                },
                plugins: {
                    title: {
                        display: true,
                        text: '{{.StockCode}} 价格走势'
                    }
                }
            }
        });
    </script>
</body>
</html>
`

// 回测图表HTML模板
const backtestChartTemplate = `
<!DOCTYPE html>
<html>
<head>
    <title>{{.StockCode}} 回测结果</title>
    <script src="https://cdn.jsdelivr.net/npm/chart.js"></script>
    <style>
        body { font-family: Arial, sans-serif; margin: 20px; }
        .container { max-width: 1200px; margin: 0 auto; }
        .header { text-align: center; margin-bottom: 20px; }
        .chart-container { position: relative; height: 400px; margin-bottom: 30px; }
        .results { display: grid; grid-template-columns: repeat(auto-fit, minmax(250px, 1fr)); gap: 20px; margin-bottom: 30px; }
        .result-card { background: #f8f9fa; padding: 15px; border-radius: 8px; border-left: 4px solid #007bff; }
        .result-card h3 { margin-top: 0; color: #333; }
        .trade-history { background: #fff; border: 1px solid #ddd; border-radius: 5px; padding: 15px; }
        .trade-item { padding: 8px 0; border-bottom: 1px solid #eee; }
        .trade-item:last-child { border-bottom: none; }
        .buy { color: #28a745; }
        .sell { color: #dc3545; }
    </style>
</head>
<body>
    <div class="container">
        <div class="header">
            <h1>{{.StockCode}} 回测结果</h1>
            <p>期间: {{.StartDate}} 至 {{.EndDate}}</p>
        </div>
        
        <div class="results">
            <div class="result-card">
                <h3>收益率</h3>
                <p style="font-size: 24px; font-weight: bold; color: {{if gt .BacktestResult.TotalReturn 0.0}}#28a745{{else}}#dc3545{{end}};">
                    {{printf "%.2f" (mul .BacktestResult.TotalReturn 100)}}%
                </p>
            </div>
            <div class="result-card">
                <h3>最大回撤</h3>
                <p style="font-size: 24px; font-weight: bold; color: #dc3545;">
                    {{printf "%.2f" (mul .BacktestResult.MaxDrawdown 100)}}%
                </p>
            </div>
            <div class="result-card">
                <h3>夏普比率</h3>
                <p style="font-size: 24px; font-weight: bold; color: #007bff;">
                    {{printf "%.2f" .BacktestResult.SharpeRatio}}
                </p>
            </div>
            <div class="result-card">
                <h3>胜率</h3>
                <p style="font-size: 24px; font-weight: bold; color: #007bff;">
                    {{printf "%.1f" (mul .BacktestResult.WinRate 100)}}%
                </p>
            </div>
        </div>
        
        <div class="chart-container">
            <canvas id="backtestChart"></canvas>
        </div>
        
        <div class="trade-history">
            <h3>交易历史</h3>
            {{range .BacktestResult.TradeHistory}}
            <div class="trade-item">
                <span class="{{if eq .Type "买入"}}buy{{else}}sell{{end}}">
                    {{.Date}} - {{.Type}} {{printf "%.2f" .Price}}元 ({{.Shares}}股) - {{.Reason}}
                </span>
            </div>
            {{end}}
        </div>
    </div>

    <script>
        const ctx = document.getElementById('backtestChart').getContext('2d');
        
        const data = {
            labels: [{{range .BacktestData}}'{{.Date}}',{{end}}],
            datasets: [{
                label: '资金曲线',
                data: [{{range .BacktestData}}{{.Capital}},{{end}}],
                borderColor: 'rgb(54, 162, 235)',
                backgroundColor: 'rgba(54, 162, 235, 0.2)',
                tension: 0.1
            }]
        };
        
        new Chart(ctx, {
            type: 'line',
            data: data,
            options: {
                responsive: true,
                maintainAspectRatio: false,
                scales: {
                    y: {
                        beginAtZero: false
                    }
                },
                plugins: {
                    title: {
                        display: true,
                        text: '资金曲线'
                    }
                }
            }
        });
    </script>
</body>
</html>
`

// 综合分析图表HTML模板
const analysisChartTemplate = `
<!DOCTYPE html>
<html>
<head>
    <title>{{.StockCode}} 综合分析</title>
    <script src="https://cdn.jsdelivr.net/npm/chart.js"></script>
    <style>
        body { font-family: Arial, sans-serif; margin: 20px; }
        .container { max-width: 1400px; margin: 0 auto; }
        .header { text-align: center; margin-bottom: 20px; }
        .charts-grid { display: grid; grid-template-columns: 1fr 1fr; gap: 20px; margin-bottom: 30px; }
        .chart-container { position: relative; height: 400px; }
        .factors-panel { background: #f8f9fa; padding: 20px; border-radius: 8px; margin-bottom: 20px; }
        .factors-grid { display: grid; grid-template-columns: repeat(auto-fit, minmax(200px, 1fr)); gap: 15px; }
        .factor-item { background: white; padding: 10px; border-radius: 5px; text-align: center; }
        .prediction-panel { background: #e3f2fd; padding: 20px; border-radius: 8px; margin-bottom: 20px; }
    </style>
</head>
<body>
    <div class="container">
        <div class="header">
            <h1>{{.StockCode}} 综合分析报告</h1>
            <p>期间: {{.StartDate}} 至 {{.EndDate}}</p>
        </div>
        
        <div class="charts-grid">
            <div class="chart-container">
                <canvas id="priceChart"></canvas>
            </div>
            <div class="chart-container">
                <canvas id="backtestChart"></canvas>
            </div>
        </div>
        
        <div class="factors-panel">
            <h3>技术指标</h3>
            <div class="factors-grid">
                <div class="factor-item">
                    <strong>MA5</strong><br>{{printf "%.2f" .Factors.MA5}}
                </div>
                <div class="factor-item">
                    <strong>MA10</strong><br>{{printf "%.2f" .Factors.MA10}}
                </div>
                <div class="factor-item">
                    <strong>RSI</strong><br>{{printf "%.1f" .Factors.RSI}}
                </div>
                <div class="factor-item">
                    <strong>MACD</strong><br>{{printf "%.4f" .Factors.MACD}}
                </div>
                <div class="factor-item">
                    <strong>动量</strong><br>{{printf "%.2f" .Factors.Momentum}}
                </div>
                <div class="factor-item">
                    <strong>波动率</strong><br>{{printf "%.4f" .Factors.Volatility}}
                </div>
            </div>
        </div>
        
        <div class="prediction-panel">
            <h3>趋势预测</h3>
            <p><strong>方向:</strong> {{.Prediction.Trend}}</p>
            <p><strong>置信度:</strong> {{printf "%.1f" .Prediction.Confidence}}%</p>
            <p><strong>风险等级:</strong> {{.Prediction.RiskLevel}}</p>
            <p><strong>操作建议:</strong> {{.Prediction.Recommendation}}</p>
        </div>
    </div>

    <script>
        // 价格图表
        const priceCtx = document.getElementById('priceChart').getContext('2d');
        new Chart(priceCtx, {
            type: 'line',
            data: {
                labels: [{{range .Klines}}'{{.Date}}',{{end}}],
                datasets: [{
                    label: '收盘价',
                    data: [{{range .Klines}}{{.Close}},{{end}}],
                    borderColor: 'rgb(75, 192, 192)',
                    backgroundColor: 'rgba(75, 192, 192, 0.2)',
                    tension: 0.1
                }]
            },
            options: {
                responsive: true,
                maintainAspectRatio: false,
                plugins: {
                    title: { display: true, text: '价格走势' }
                }
            }
        });
        
        // 回测图表
        const backtestCtx = document.getElementById('backtestChart').getContext('2d');
        new Chart(backtestCtx, {
            type: 'line',
            data: {
                labels: [{{range .BacktestData}}'{{.Date}}',{{end}}],
                datasets: [{
                    label: '资金曲线',
                    data: [{{range .BacktestData}}{{.Capital}},{{end}}],
                    borderColor: 'rgb(54, 162, 235)',
                    backgroundColor: 'rgba(54, 162, 235, 0.2)',
                    tension: 0.1
                }]
            },
            options: {
                responsive: true,
                maintainAspectRatio: false,
                plugins: {
                    title: { display: true, text: '回测资金曲线' }
                }
            }
        });
    </script>
</body>
</html>
`
