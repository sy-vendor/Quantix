package analysis

import (
	"Quantix/data"
	"fmt"
	"math"
	"sort"
)

// RiskMetrics 风险指标结构
type RiskMetrics struct {
	VaR95             float64 // 95%置信度下的VaR
	VaR99             float64 // 99%置信度下的VaR
	MaxDrawdown       float64 // 最大回撤
	SharpeRatio       float64 // 夏普比率
	SortinoRatio      float64 // 索提诺比率
	CalmarRatio       float64 // 卡玛比率
	Volatility        float64 // 年化波动率
	Skewness          float64 // 偏度
	Kurtosis          float64 // 峰度
	Beta              float64 // 贝塔系数（相对于市场）
	Alpha             float64 // 阿尔法系数
	InformationRatio  float64 // 信息比率
	DownsideDeviation float64 // 下行偏差
	UpsideCapture     float64 // 上行捕获率
	DownsideCapture   float64 // 下行捕获率
}

// CalculateRiskMetrics 计算风险指标
func CalculateRiskMetrics(klines []data.Kline, riskFreeRate float64) RiskMetrics {
	if len(klines) < 30 {
		return RiskMetrics{}
	}

	// 计算日收益率
	returns := calculateReturns(klines)

	// 计算年化收益率
	annualReturn := calculateAnnualReturn(klines)

	// 计算年化波动率
	volatility := calculateAnnualVolatility(returns)

	// 计算VaR
	var95 := calculateVaR(returns, 0.95)
	var99 := calculateVaR(returns, 0.99)

	// 计算最大回撤
	maxDrawdown := calculateMaxDrawdown(klines)

	// 计算夏普比率
	sharpeRatio := calculateSharpeRatio(annualReturn, volatility, riskFreeRate)

	// 计算索提诺比率
	sortinoRatio := calculateSortinoRatio(returns, riskFreeRate)

	// 计算卡玛比率
	calmarRatio := calculateCalmarRatio(annualReturn, maxDrawdown)

	// 计算偏度和峰度
	skewness := calculateSkewness(returns)
	kurtosis := calculateKurtosis(returns)

	// 计算下行偏差
	downsideDeviation := calculateDownsideDeviation(returns, riskFreeRate)

	// 计算上行/下行捕获率
	upsideCapture, downsideCapture := calculateCaptureRatios(returns)

	return RiskMetrics{
		VaR95:             var95,
		VaR99:             var99,
		MaxDrawdown:       maxDrawdown,
		SharpeRatio:       sharpeRatio,
		SortinoRatio:      sortinoRatio,
		CalmarRatio:       calmarRatio,
		Volatility:        volatility,
		Skewness:          skewness,
		Kurtosis:          kurtosis,
		DownsideDeviation: downsideDeviation,
		UpsideCapture:     upsideCapture,
		DownsideCapture:   downsideCapture,
	}
}

// calculateReturns 计算日收益率
func calculateReturns(klines []data.Kline) []float64 {
	var returns []float64
	for i := 1; i < len(klines); i++ {
		return_ := (klines[i].Close - klines[i-1].Close) / klines[i-1].Close
		returns = append(returns, return_)
	}
	return returns
}

// calculateAnnualReturn 计算年化收益率
func calculateAnnualReturn(klines []data.Kline) float64 {
	if len(klines) < 2 {
		return 0
	}

	totalReturn := (klines[len(klines)-1].Close - klines[0].Close) / klines[0].Close
	days := float64(len(klines))
	annualReturn := math.Pow(1+totalReturn, 365/days) - 1

	return annualReturn
}

// calculateAnnualVolatility 计算年化波动率
func calculateAnnualVolatility(returns []float64) float64 {
	if len(returns) < 2 {
		return 0
	}

	mean := 0.0
	for _, r := range returns {
		mean += r
	}
	mean /= float64(len(returns))

	variance := 0.0
	for _, r := range returns {
		variance += (r - mean) * (r - mean)
	}
	variance /= float64(len(returns) - 1)

	dailyVolatility := math.Sqrt(variance)
	annualVolatility := dailyVolatility * math.Sqrt(252) // 假设252个交易日

	return annualVolatility
}

// calculateVaR 计算VaR（Value at Risk）
func calculateVaR(returns []float64, confidence float64) float64 {
	if len(returns) == 0 {
		return 0
	}

	// 复制returns并排序
	sortedReturns := make([]float64, len(returns))
	copy(sortedReturns, returns)
	sort.Float64s(sortedReturns)

	// 计算分位数
	index := int(float64(len(sortedReturns)) * (1 - confidence))
	if index >= len(sortedReturns) {
		index = len(sortedReturns) - 1
	}

	return -sortedReturns[index] // 返回负值，表示损失
}

// calculateMaxDrawdown 计算最大回撤
func calculateMaxDrawdown(klines []data.Kline) float64 {
	if len(klines) < 2 {
		return 0
	}

	maxDrawdown := 0.0
	peak := klines[0].Close

	for _, kline := range klines {
		if kline.Close > peak {
			peak = kline.Close
		}

		drawdown := (peak - kline.Close) / peak
		if drawdown > maxDrawdown {
			maxDrawdown = drawdown
		}
	}

	return maxDrawdown
}

// calculateSharpeRatio 计算夏普比率
func calculateSharpeRatio(annualReturn, volatility, riskFreeRate float64) float64 {
	if volatility == 0 {
		return 0
	}
	return (annualReturn - riskFreeRate) / volatility
}

// calculateSortinoRatio 计算索提诺比率
func calculateSortinoRatio(returns []float64, riskFreeRate float64) float64 {
	if len(returns) == 0 {
		return 0
	}

	// 计算平均收益率
	meanReturn := 0.0
	for _, r := range returns {
		meanReturn += r
	}
	meanReturn /= float64(len(returns))

	// 年化收益率
	annualReturn := meanReturn * 252

	// 计算下行偏差
	downsideDeviation := calculateDownsideDeviation(returns, riskFreeRate)

	if downsideDeviation == 0 {
		return 0
	}

	return (annualReturn - riskFreeRate) / downsideDeviation
}

// calculateCalmarRatio 计算卡玛比率
func calculateCalmarRatio(annualReturn, maxDrawdown float64) float64 {
	if maxDrawdown == 0 {
		return 0
	}
	return annualReturn / maxDrawdown
}

// calculateSkewness 计算偏度
func calculateSkewness(returns []float64) float64 {
	if len(returns) < 3 {
		return 0
	}

	mean := 0.0
	for _, r := range returns {
		mean += r
	}
	mean /= float64(len(returns))

	variance := 0.0
	for _, r := range returns {
		variance += (r - mean) * (r - mean)
	}
	variance /= float64(len(returns))

	stdDev := math.Sqrt(variance)
	if stdDev == 0 {
		return 0
	}

	skewness := 0.0
	for _, r := range returns {
		skewness += math.Pow((r-mean)/stdDev, 3)
	}
	skewness /= float64(len(returns))

	return skewness
}

// calculateKurtosis 计算峰度
func calculateKurtosis(returns []float64) float64 {
	if len(returns) < 4 {
		return 0
	}

	mean := 0.0
	for _, r := range returns {
		mean += r
	}
	mean /= float64(len(returns))

	variance := 0.0
	for _, r := range returns {
		variance += (r - mean) * (r - mean)
	}
	variance /= float64(len(returns))

	stdDev := math.Sqrt(variance)
	if stdDev == 0 {
		return 0
	}

	kurtosis := 0.0
	for _, r := range returns {
		kurtosis += math.Pow((r-mean)/stdDev, 4)
	}
	kurtosis /= float64(len(returns))

	return kurtosis - 3 // 减去正态分布的峰度3
}

// calculateDownsideDeviation 计算下行偏差
func calculateDownsideDeviation(returns []float64, riskFreeRate float64) float64 {
	if len(returns) == 0 {
		return 0
	}

	dailyRiskFreeRate := riskFreeRate / 252
	sum := 0.0
	count := 0

	for _, r := range returns {
		if r < dailyRiskFreeRate {
			sum += (dailyRiskFreeRate - r) * (dailyRiskFreeRate - r)
			count++
		}
	}

	if count == 0 {
		return 0
	}

	downsideVariance := sum / float64(count)
	return math.Sqrt(downsideVariance * 252) // 年化
}

// calculateCaptureRatios 计算上行/下行捕获率
func calculateCaptureRatios(returns []float64) (float64, float64) {
	if len(returns) == 0 {
		return 0, 0
	}

	// 计算基准收益率（这里用简单平均作为基准）
	benchmarkReturn := 0.0
	for _, r := range returns {
		benchmarkReturn += r
	}
	benchmarkReturn /= float64(len(returns))

	upsideSum := 0.0
	downsideSum := 0.0
	upsideCount := 0
	downsideCount := 0

	for _, r := range returns {
		if r > benchmarkReturn {
			upsideSum += r
			upsideCount++
		} else if r < benchmarkReturn {
			downsideSum += r
			downsideCount++
		}
	}

	upsideCapture := 0.0
	downsideCapture := 0.0

	if upsideCount > 0 {
		upsideCapture = upsideSum / float64(upsideCount) / benchmarkReturn * 100
	}

	if downsideCount > 0 {
		downsideCapture = downsideSum / float64(downsideCount) / benchmarkReturn * 100
	}

	return upsideCapture, downsideCapture
}

// PrintRiskMetrics 打印风险指标
func PrintRiskMetrics(metrics RiskMetrics) {
	fmt.Println("\n=== 风险指标分析 ===")

	fmt.Println("--- 风险度量 ---")
	fmt.Printf("VaR(95%%): %.2f%%\n", metrics.VaR95*100)
	fmt.Printf("VaR(99%%): %.2f%%\n", metrics.VaR99*100)
	fmt.Printf("最大回撤: %.2f%%\n", metrics.MaxDrawdown*100)
	fmt.Printf("年化波动率: %.2f%%\n", metrics.Volatility*100)

	fmt.Println("\n--- 风险调整收益 ---")
	fmt.Printf("夏普比率: %.3f\n", metrics.SharpeRatio)
	fmt.Printf("索提诺比率: %.3f\n", metrics.SortinoRatio)
	fmt.Printf("卡玛比率: %.3f\n", metrics.CalmarRatio)
	fmt.Printf("信息比率: %.3f\n", metrics.InformationRatio)

	fmt.Println("\n--- 分布特征 ---")
	fmt.Printf("偏度: %.3f\n", metrics.Skewness)
	fmt.Printf("峰度: %.3f\n", metrics.Kurtosis)
	fmt.Printf("下行偏差: %.2f%%\n", metrics.DownsideDeviation*100)

	fmt.Println("\n--- 捕获率 ---")
	fmt.Printf("上行捕获率: %.2f%%\n", metrics.UpsideCapture)
	fmt.Printf("下行捕获率: %.2f%%\n", metrics.DownsideCapture)

	// 风险评级
	fmt.Println("\n=== 风险评级 ===")
	printRiskRating(metrics)
}

// printRiskRating 打印风险评级
func printRiskRating(metrics RiskMetrics) {
	// 基于最大回撤的风险评级
	if metrics.MaxDrawdown < 0.05 {
		fmt.Println("🟢 风险等级: 低风险 (最大回撤 < 5%)")
	} else if metrics.MaxDrawdown < 0.15 {
		fmt.Println("🟡 风险等级: 中低风险 (最大回撤 5-15%)")
	} else if metrics.MaxDrawdown < 0.25 {
		fmt.Println("🟠 风险等级: 中风险 (最大回撤 15-25%)")
	} else if metrics.MaxDrawdown < 0.35 {
		fmt.Println("🔴 风险等级: 中高风险 (最大回撤 25-35%)")
	} else {
		fmt.Println("⚫ 风险等级: 高风险 (最大回撤 > 35%)")
	}

	// 基于夏普比率的收益评级
	if metrics.SharpeRatio > 1.0 {
		fmt.Println("📈 收益评级: 优秀 (夏普比率 > 1.0)")
	} else if metrics.SharpeRatio > 0.5 {
		fmt.Println("📊 收益评级: 良好 (夏普比率 0.5-1.0)")
	} else if metrics.SharpeRatio > 0.0 {
		fmt.Println("➡️  收益评级: 一般 (夏普比率 0-0.5)")
	} else {
		fmt.Println("📉 收益评级: 较差 (夏普比率 < 0)")
	}

	// 风险建议
	fmt.Println("\n=== 风险建议 ===")
	if metrics.MaxDrawdown > 0.25 {
		fmt.Println("⚠️  建议: 风险较高，建议降低仓位或增加止损")
	} else if metrics.SharpeRatio < 0.3 {
		fmt.Println("💡 建议: 风险调整收益较低，考虑优化策略")
	} else {
		fmt.Println("✅ 建议: 风险收益比合理，可继续持有")
	}
}
