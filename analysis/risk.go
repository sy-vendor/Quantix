package analysis

import (
	"math"
	"sort"
)

// RiskMetrics 风险指标结构
type RiskMetrics struct {
	Volatility  float64 // 历史波动率
	VaR95       float64 // 95%置信度下的风险价值
	MaxDrawdown float64 // 最大回撤
	SharpeRatio float64 // 夏普比率
	Beta        float64 // 贝塔系数
	RiskLevel   string  // 风险等级
	RiskScore   float64 // 风险评分（0-100）
}

// CalculateRiskMetrics 计算风险指标
func CalculateRiskMetrics(stockData []StockData) RiskMetrics {
	if len(stockData) < 30 {
		return RiskMetrics{RiskLevel: "数据不足", RiskScore: 0}
	}

	// 计算日收益率
	returns := calculateReturns(stockData)

	// 计算各项指标
	volatility := calculateVolatility(returns)
	var95, _ := calculateVaR(returns)
	maxDrawdown, _ := calculateMaxDrawdown(stockData)
	sharpeRatio := calculateSharpeRatio(returns)
	riskScore := calculateRiskScore(volatility, maxDrawdown)
	riskLevel := determineRiskLevel(riskScore)

	return RiskMetrics{
		Volatility:  volatility,
		VaR95:       var95,
		MaxDrawdown: maxDrawdown,
		SharpeRatio: sharpeRatio,
		Beta:        1.0, // 默认值
		RiskLevel:   riskLevel,
		RiskScore:   riskScore,
	}
}

// calculateReturns 计算日收益率
func calculateReturns(data []StockData) []float64 {
	var returns []float64
	for i := 1; i < len(data); i++ {
		ret := (data[i].Close - data[i-1].Close) / data[i-1].Close
		returns = append(returns, ret)
	}
	return returns
}

// calculateVolatility 计算历史波动率
func calculateVolatility(returns []float64) float64 {
	if len(returns) == 0 {
		return 0
	}

	var sum float64
	for _, r := range returns {
		sum += r
	}
	mean := sum / float64(len(returns))

	var variance float64
	for _, r := range returns {
		variance += math.Pow(r-mean, 2)
	}
	variance /= float64(len(returns) - 1)

	return math.Sqrt(variance) * math.Sqrt(252)
}

// calculateVaR 计算风险价值
func calculateVaR(returns []float64) (float95, float99 float64) {
	if len(returns) == 0 {
		return 0, 0
	}

	sortedReturns := make([]float64, len(returns))
	copy(sortedReturns, returns)
	sort.Float64s(sortedReturns)

	idx95 := int(float64(len(sortedReturns)) * 0.05)
	if idx95 < len(sortedReturns) {
		float95 = sortedReturns[idx95]
	}

	return float95, 0
}

// calculateMaxDrawdown 计算最大回撤
func calculateMaxDrawdown(data []StockData) (maxDrawdown float64, duration int) {
	if len(data) == 0 {
		return 0, 0
	}

	peak := data[0].Close
	peakIdx := 0
	maxDrawdown = 0
	duration = 0

	for i, d := range data {
		if d.Close > peak {
			peak = d.Close
			peakIdx = i
		}

		drawdown := (peak - d.Close) / peak
		if drawdown > maxDrawdown {
			maxDrawdown = drawdown
			duration = i - peakIdx
		}
	}

	return maxDrawdown, duration
}

// calculateSharpeRatio 计算夏普比率
func calculateSharpeRatio(returns []float64) float64 {
	if len(returns) == 0 {
		return 0
	}

	var sum float64
	for _, r := range returns {
		sum += r
	}
	mean := sum / float64(len(returns))

	var variance float64
	for _, r := range returns {
		variance += math.Pow(r-mean, 2)
	}
	stdDev := math.Sqrt(variance / float64(len(returns)-1))

	if stdDev == 0 {
		return 0
	}

	riskFreeRate := 0.03 / 252
	return (mean - riskFreeRate) / stdDev * math.Sqrt(252)
}

// calculateRiskScore 计算风险评分
func calculateRiskScore(volatility, maxDrawdown float64) float64 {
	volScore := math.Min(volatility*100, 40)
	drawdownScore := maxDrawdown * 30
	return volScore + drawdownScore
}

// determineRiskLevel 确定风险等级
func determineRiskLevel(riskScore float64) string {
	switch {
	case riskScore < 20:
		return "低风险"
	case riskScore < 40:
		return "中低风险"
	case riskScore < 60:
		return "中风险"
	case riskScore < 80:
		return "高风险"
	default:
		return "极高风险"
	}
}
