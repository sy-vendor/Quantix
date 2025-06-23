package analysis_test

import (
	"Quantix/analysis"
	"Quantix/data"
	"testing"
	"time"
)

func TestNewFeatures(t *testing.T) {
	t.Log("=== Quantix 新功能测试 ===")

	testKlines := createTestData()
	t.Logf("创建了 %d 条测试K线数据", len(testKlines))

	// 技术指标计算
	factors := analysis.CalcFactors(testKlines)
	if len(factors) == 0 {
		t.Fatal("技术指标计算失败")
	}
	latest := factors[len(factors)-1]
	t.Logf("RSI: %.2f, MACD: %.4f, BBUpper: %.2f, KDJ-K: %.2f, WR: %.2f, CCI: %.2f", latest.RSI, latest.MACD, latest.BBUpper, latest.KDJ_K, latest.WR, latest.CCI)

	// 风险管理
	riskMetrics := analysis.CalculateRiskMetrics(testKlines, 0.03)
	t.Logf("VaR(95%%): %.2f%%, 最大回撤: %.2f%%, 夏普比率: %.3f, 年化波动率: %.2f%%", riskMetrics.VaR95*100, riskMetrics.MaxDrawdown*100, riskMetrics.SharpeRatio, riskMetrics.Volatility*100)

	// 机器学习预测
	predictor := analysis.NewMLPredictor(testKlines, factors)
	mlPredictions := predictor.PredictAll()
	if len(mlPredictions) == 0 {
		t.Error("机器学习预测失败")
	}
	for _, pred := range mlPredictions {
		if pred.Confidence > 0 {
			t.Logf("%s: 下一天价格 %.2f (置信度 %.1f%%)", pred.Method, pred.NextDayPrice, pred.Confidence*100)
		}
	}

	// 传统趋势预测
	prediction := analysis.PredictTrend(testKlines)
	t.Logf("趋势: %s, 置信度: %.1f%%, 信号强度: %s", prediction.Trend, prediction.Confidence, prediction.SignalStrength)

	// 多股票对比
	stockData := map[string][]data.Kline{
		"000001.SZ": testKlines,
		"600519.SH": createTestData2(),
	}
	comparison := analysis.AnalyzeMultipleStocks(stockData)
	if len(comparison.Stocks) == 0 {
		t.Error("多股票对比失败")
	}
	for _, stock := range comparison.Stocks {
		t.Logf("%s: 评分 %.1f, 建议: %s", stock.Code, stock.Score, stock.Recommend)
	}
}

func TestRiskManagement(t *testing.T) {
	t.Log("=== 风险管理模块测试 ===")
	testCases := []struct {
		name     string
		klines   []data.Kline
		expected string
	}{
		{"低波动性股票", createLowVolatilityData(), "低风险"},
		{"高波动性股票", createHighVolatilityData(), "高风险"},
		{"趋势性股票", createTrendingData(), "中风险"},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			if len(tc.klines) < 30 {
				t.Skip("数据不足，跳过测试")
			}
			riskMetrics := analysis.CalculateRiskMetrics(tc.klines, 0.03)
			t.Logf("VaR(95%%): %.2f%%, VaR(99%%): %.2f%%, 最大回撤: %.2f%%, 夏普比率: %.3f, 索提诺比率: %.3f, 卡玛比率: %.3f, 年化波动率: %.2f%%, 偏度: %.3f, 峰度: %.3f, 下行偏差: %.2f%%, 上行捕获率: %.2f%%, 下行捕获率: %.2f%%",
				riskMetrics.VaR95*100, riskMetrics.VaR99*100, riskMetrics.MaxDrawdown*100, riskMetrics.SharpeRatio, riskMetrics.SortinoRatio, riskMetrics.CalmarRatio, riskMetrics.Volatility*100, riskMetrics.Skewness, riskMetrics.Kurtosis, riskMetrics.DownsideDeviation*100, riskMetrics.UpsideCapture, riskMetrics.DownsideCapture)
			if riskMetrics.MaxDrawdown < 0 {
				t.Errorf("MaxDrawdown should not be negative")
			}
		})
	}
}

func TestMLPrediction(t *testing.T) {
	t.Log("=== 机器学习预测模块测试 ===")
	testCases := []struct {
		name   string
		klines []data.Kline
	}{
		{"上涨趋势数据", createUptrendData()},
		{"下跌趋势数据", createDowntrendData()},
		{"震荡数据", createSidewaysData()},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			if len(tc.klines) < 30 {
				t.Skip("数据不足，跳过测试")
			}
			factors := analysis.CalcFactors(tc.klines)
			if len(factors) == 0 {
				t.Fatal("技术指标计算失败")
			}
			predictor := analysis.NewMLPredictor(tc.klines, factors)
			predictions := predictor.PredictAll()
			if len(predictions) == 0 {
				t.Error("预测失败")
			}
			for _, pred := range predictions {
				if pred.Confidence > 0 {
					t.Logf("%s预测: 下一天价格: %.2f, 下一周价格: %.2f, 下一月价格: %.2f, 置信度: %.1f%%, 趋势: %s, 历史准确率: %.1f%%", pred.Method, pred.NextDayPrice, pred.NextWeekPrice, pred.NextMonthPrice, pred.Confidence*100, pred.Trend, pred.Accuracy*100)
				}
			}
		})
	}
}

// 以下为数据生成函数，直接复制即可
// 创建测试数据1
func createTestData() []data.Kline {
	var klines []data.Kline
	basePrice := 10.0
	baseDate := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	for i := 0; i < 60; i++ {
		date := baseDate.AddDate(0, 0, i)
		price := basePrice + float64(i)*0.1 + float64(i%10)*0.05
		high := price * 1.02
		low := price * 0.98
		open := price * 0.99
		volume := int64(1000000 + i*10000)
		klines = append(klines, data.Kline{
			Date:   date,
			Open:   open,
			High:   high,
			Low:    low,
			Close:  price,
			Volume: volume,
		})
	}
	return klines
}

func createTestData2() []data.Kline {
	var klines []data.Kline
	basePrice := 1500.0
	baseDate := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	for i := 0; i < 60; i++ {
		date := baseDate.AddDate(0, 0, i)
		price := basePrice + float64(i)*5.0 + float64(i%15)*2.0
		high := price * 1.015
		low := price * 0.985
		open := price * 0.995
		volume := int64(500000 + i*5000)
		klines = append(klines, data.Kline{
			Date:   date,
			Open:   open,
			High:   high,
			Low:    low,
			Close:  price,
			Volume: volume,
		})
	}
	return klines
}

func createLowVolatilityData() []data.Kline {
	var klines []data.Kline
	basePrice := 100.0
	baseDate := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	for i := 0; i < 60; i++ {
		date := baseDate.AddDate(0, 0, i)
		price := basePrice + float64(i%10)*0.1
		high := price * 1.005
		low := price * 0.995
		open := price * 0.998
		volume := int64(1000000 + i*1000)
		klines = append(klines, data.Kline{
			Date:   date,
			Open:   open,
			High:   high,
			Low:    low,
			Close:  price,
			Volume: volume,
		})
	}
	return klines
}

func createHighVolatilityData() []data.Kline {
	var klines []data.Kline
	basePrice := 50.0
	baseDate := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	for i := 0; i < 60; i++ {
		date := baseDate.AddDate(0, 0, i)
		volatility := 0.05 + float64(i%5)*0.02
		price := basePrice + float64(i)*0.5 + float64(i%10)*volatility*basePrice
		high := price * 1.05
		low := price * 0.95
		open := price * 0.98
		volume := int64(2000000 + i*50000)
		klines = append(klines, data.Kline{
			Date:   date,
			Open:   open,
			High:   high,
			Low:    low,
			Close:  price,
			Volume: volume,
		})
	}
	return klines
}

func createTrendingData() []data.Kline {
	var klines []data.Kline
	basePrice := 20.0
	baseDate := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	for i := 0; i < 60; i++ {
		date := baseDate.AddDate(0, 0, i)
		price := basePrice + float64(i)*0.3
		high := price * 1.02
		low := price * 0.98
		open := price * 0.99
		volume := int64(1500000 + i*20000)
		klines = append(klines, data.Kline{
			Date:   date,
			Open:   open,
			High:   high,
			Low:    low,
			Close:  price,
			Volume: volume,
		})
	}
	return klines
}

func createUptrendData() []data.Kline {
	var klines []data.Kline
	basePrice := 50.0
	baseDate := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	for i := 0; i < 60; i++ {
		date := baseDate.AddDate(0, 0, i)
		price := basePrice + float64(i)*1.0
		high := price * 1.03
		low := price * 0.97
		open := price * 0.99
		volume := int64(1000000 + i*20000)
		klines = append(klines, data.Kline{
			Date:   date,
			Open:   open,
			High:   high,
			Low:    low,
			Close:  price,
			Volume: volume,
		})
	}
	return klines
}

func createDowntrendData() []data.Kline {
	var klines []data.Kline
	basePrice := 100.0
	baseDate := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	for i := 0; i < 60; i++ {
		date := baseDate.AddDate(0, 0, i)
		price := basePrice - float64(i)*0.8
		high := price * 1.02
		low := price * 0.98
		open := price * 1.01
		volume := int64(1200000 + i*15000)
		klines = append(klines, data.Kline{
			Date:   date,
			Open:   open,
			High:   high,
			Low:    low,
			Close:  price,
			Volume: volume,
		})
	}
	return klines
}

func createSidewaysData() []data.Kline {
	var klines []data.Kline
	basePrice := 30.0
	baseDate := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	for i := 0; i < 60; i++ {
		date := baseDate.AddDate(0, 0, i)
		price := basePrice + float64(i%20)*0.5 - 5.0
		high := price * 1.02
		low := price * 0.98
		open := price * 1.0
		volume := int64(800000 + i*8000)
		klines = append(klines, data.Kline{
			Date:   date,
			Open:   open,
			High:   high,
			Low:    low,
			Close:  price,
			Volume: volume,
		})
	}
	return klines
}
