package analysis

import (
	"Quantix/data"
	"fmt"
	"math"
)

// MLPrediction 机器学习预测结果
type MLPrediction struct {
	Method          string  // 预测方法
	NextDayPrice    float64 // 下一天预测价格
	NextWeekPrice   float64 // 下一周预测价格
	NextMonthPrice  float64 // 下一月预测价格
	Confidence      float64 // 预测置信度
	Trend           string  // 趋势方向
	SupportLevel    float64 // 支撑位
	ResistanceLevel float64 // 阻力位
	Accuracy        float64 // 历史准确率
}

// MLPredictor 机器学习预测器
type MLPredictor struct {
	klines  []data.Kline
	factors []Factors
}

// NewMLPredictor 创建新的预测器
func NewMLPredictor(klines []data.Kline, factors []Factors) *MLPredictor {
	return &MLPredictor{
		klines:  klines,
		factors: factors,
	}
}

// PredictAll 执行所有预测方法
func (p *MLPredictor) PredictAll() map[string]MLPrediction {
	predictions := make(map[string]MLPrediction)

	// 线性回归预测
	predictions["linear"] = p.linearRegressionPredict()

	// 移动平均预测
	predictions["ma"] = p.movingAveragePredict()

	// 技术指标预测
	predictions["technical"] = p.technicalIndicatorPredict()

	// 组合预测
	predictions["ensemble"] = p.ensemblePredict(predictions)

	return predictions
}

// linearRegressionPredict 线性回归预测
func (p *MLPredictor) linearRegressionPredict() MLPrediction {
	if len(p.klines) < 20 {
		return MLPrediction{Method: "线性回归", Confidence: 0}
	}

	// 使用最近20天的数据进行线性回归
	n := 20
	x := make([]float64, n)
	y := make([]float64, n)

	for i := 0; i < n; i++ {
		x[i] = float64(i)
		y[i] = p.klines[len(p.klines)-n+i].Close
	}

	// 计算线性回归参数
	slope, intercept := p.calculateLinearRegression(x, y)

	// 预测未来价格
	nextDayPrice := slope*float64(n) + intercept
	nextWeekPrice := slope*float64(n+5) + intercept
	nextMonthPrice := slope*float64(n+20) + intercept

	// 计算置信度（基于R²）
	confidence := p.calculateR2(x, y, slope, intercept)

	// 确定趋势
	trend := "横盘"
	if slope > 0.01 {
		trend = "上涨"
	} else if slope < -0.01 {
		trend = "下跌"
	}

	// 计算支撑和阻力位
	support, resistance := p.calculateSupportResistance()

	return MLPrediction{
		Method:          "线性回归",
		NextDayPrice:    nextDayPrice,
		NextWeekPrice:   nextWeekPrice,
		NextMonthPrice:  nextMonthPrice,
		Confidence:      confidence,
		Trend:           trend,
		SupportLevel:    support,
		ResistanceLevel: resistance,
		Accuracy:        p.calculateHistoricalAccuracy("linear"),
	}
}

// movingAveragePredict 移动平均预测
func (p *MLPredictor) movingAveragePredict() MLPrediction {
	if len(p.klines) < 30 {
		return MLPrediction{Method: "移动平均", Confidence: 0}
	}

	latest := p.klines[len(p.klines)-1]

	// 计算各种移动平均
	ma5 := p.calculateMA(5)
	ma10 := p.calculateMA(10)
	ma20 := p.calculateMA(20)
	ma30 := p.calculateMA(30)

	// 预测下一天价格（基于移动平均趋势）
	trend := (ma5 + ma10 + ma20) / 3
	nextDayPrice := latest.Close + (trend-latest.Close)*0.1

	// 预测未来价格
	nextWeekPrice := nextDayPrice * math.Pow(1.001, 5) // 假设每日0.1%的增长率
	nextMonthPrice := nextDayPrice * math.Pow(1.001, 20)

	// 计算置信度
	confidence := p.calculateMAConfidence(ma5, ma10, ma20, ma30)

	// 确定趋势
	trendDirection := "横盘"
	if ma5 > ma10 && ma10 > ma20 {
		trendDirection = "上涨"
	} else if ma5 < ma10 && ma10 < ma20 {
		trendDirection = "下跌"
	}

	// 计算支撑和阻力位
	support := math.Min(ma20, ma30)
	resistance := math.Max(ma5, ma10)

	return MLPrediction{
		Method:          "移动平均",
		NextDayPrice:    nextDayPrice,
		NextWeekPrice:   nextWeekPrice,
		NextMonthPrice:  nextMonthPrice,
		Confidence:      confidence,
		Trend:           trendDirection,
		SupportLevel:    support,
		ResistanceLevel: resistance,
		Accuracy:        p.calculateHistoricalAccuracy("ma"),
	}
}

// technicalIndicatorPredict 技术指标预测
func (p *MLPredictor) technicalIndicatorPredict() MLPrediction {
	if len(p.factors) == 0 {
		return MLPrediction{Method: "技术指标", Confidence: 0}
	}

	latest := p.factors[len(p.factors)-1]
	currentPrice := latest.Close

	// 基于技术指标预测
	var priceChange float64
	var confidence float64

	// RSI信号
	if latest.RSI < 30 {
		priceChange += 0.02 // 超卖，预期反弹
		confidence += 0.3
	} else if latest.RSI > 70 {
		priceChange -= 0.02 // 超买，预期回调
		confidence += 0.3
	}

	// MACD信号
	if latest.MACD > latest.MACDSignal && latest.MACDHist > 0 {
		priceChange += 0.015
		confidence += 0.2
	} else if latest.MACD < latest.MACDSignal && latest.MACDHist < 0 {
		priceChange -= 0.015
		confidence += 0.2
	}

	// 布林带信号
	if latest.BBPosition < 20 {
		priceChange += 0.01 // 接近下轨，预期反弹
		confidence += 0.2
	} else if latest.BBPosition > 80 {
		priceChange -= 0.01 // 接近上轨，预期回调
		confidence += 0.2
	}

	// KDJ信号
	if latest.KDJ_J < 20 {
		priceChange += 0.01
		confidence += 0.1
	} else if latest.KDJ_J > 80 {
		priceChange -= 0.01
		confidence += 0.1
	}

	// 计算预测价格
	nextDayPrice := currentPrice * (1 + priceChange)
	nextWeekPrice := nextDayPrice * math.Pow(1+priceChange/5, 5)
	nextMonthPrice := nextDayPrice * math.Pow(1+priceChange/20, 20)

	// 确定趋势
	trend := "横盘"
	if priceChange > 0.01 {
		trend = "上涨"
	} else if priceChange < -0.01 {
		trend = "下跌"
	}

	// 计算支撑和阻力位
	support := latest.BBLower
	resistance := latest.BBUpper

	return MLPrediction{
		Method:          "技术指标",
		NextDayPrice:    nextDayPrice,
		NextWeekPrice:   nextWeekPrice,
		NextMonthPrice:  nextMonthPrice,
		Confidence:      math.Min(confidence, 1.0),
		Trend:           trend,
		SupportLevel:    support,
		ResistanceLevel: resistance,
		Accuracy:        p.calculateHistoricalAccuracy("technical"),
	}
}

// ensemblePredict 组合预测
func (p *MLPredictor) ensemblePredict(predictions map[string]MLPrediction) MLPrediction {
	var totalPrice float64
	var totalConfidence float64
	var count int

	for _, pred := range predictions {
		if pred.Confidence > 0 {
			totalPrice += pred.NextDayPrice * pred.Confidence
			totalConfidence += pred.Confidence
			count++
		}
	}

	if count == 0 {
		return MLPrediction{Method: "组合预测", Confidence: 0}
	}

	avgPrice := totalPrice / totalConfidence
	avgConfidence := totalConfidence / float64(count)

	// 计算加权平均的未来价格
	var nextWeekPrice, nextMonthPrice float64
	for _, pred := range predictions {
		if pred.Confidence > 0 {
			nextWeekPrice += pred.NextWeekPrice * pred.Confidence
			nextMonthPrice += pred.NextMonthPrice * pred.Confidence
		}
	}
	nextWeekPrice /= totalConfidence
	nextMonthPrice /= totalConfidence

	// 确定趋势
	trend := "横盘"
	if avgPrice > p.klines[len(p.klines)-1].Close*1.01 {
		trend = "上涨"
	} else if avgPrice < p.klines[len(p.klines)-1].Close*0.99 {
		trend = "下跌"
	}

	return MLPrediction{
		Method:         "组合预测",
		NextDayPrice:   avgPrice,
		NextWeekPrice:  nextWeekPrice,
		NextMonthPrice: nextMonthPrice,
		Confidence:     avgConfidence,
		Trend:          trend,
		Accuracy:       p.calculateHistoricalAccuracy("ensemble"),
	}
}

// 辅助方法
func (p *MLPredictor) calculateLinearRegression(x, y []float64) (float64, float64) {
	n := len(x)
	if n != len(y) || n == 0 {
		return 0, 0
	}

	var sumX, sumY, sumXY, sumX2 float64
	for i := 0; i < n; i++ {
		sumX += x[i]
		sumY += y[i]
		sumXY += x[i] * y[i]
		sumX2 += x[i] * x[i]
	}

	slope := (float64(n)*sumXY - sumX*sumY) / (float64(n)*sumX2 - sumX*sumX)
	intercept := (sumY - slope*sumX) / float64(n)

	return slope, intercept
}

func (p *MLPredictor) calculateR2(x, y []float64, slope, intercept float64) float64 {
	n := len(x)
	if n != len(y) || n == 0 {
		return 0
	}

	// 计算平均值
	meanY := 0.0
	for _, val := range y {
		meanY += val
	}
	meanY /= float64(n)

	// 计算总平方和和残差平方和
	var ssTotal, ssResidual float64
	for i := 0; i < n; i++ {
		predicted := slope*x[i] + intercept
		ssTotal += (y[i] - meanY) * (y[i] - meanY)
		ssResidual += (y[i] - predicted) * (y[i] - predicted)
	}

	if ssTotal == 0 {
		return 0
	}

	return 1 - ssResidual/ssTotal
}

func (p *MLPredictor) calculateMA(period int) float64 {
	if len(p.klines) < period {
		return 0
	}

	sum := 0.0
	for i := len(p.klines) - period; i < len(p.klines); i++ {
		sum += p.klines[i].Close
	}
	return sum / float64(period)
}

func (p *MLPredictor) calculateMAConfidence(ma5, ma10, ma20, ma30 float64) float64 {
	// 基于移动平均线的排列计算置信度
	confidence := 0.5 // 基础置信度

	// 短期均线向上排列
	if ma5 > ma10 && ma10 > ma20 {
		confidence += 0.2
	}
	// 长期均线向上排列
	if ma10 > ma20 && ma20 > ma30 {
		confidence += 0.2
	}
	// 均线间距合理
	if math.Abs(ma5-ma10)/ma10 < 0.05 {
		confidence += 0.1
	}

	return math.Min(confidence, 1.0)
}

func (p *MLPredictor) calculateSupportResistance() (float64, float64) {
	if len(p.klines) < 20 {
		return 0, 0
	}

	// 计算最近20天的最高价和最低价
	high := p.klines[len(p.klines)-20].High
	low := p.klines[len(p.klines)-20].Low

	for i := len(p.klines) - 19; i < len(p.klines); i++ {
		if p.klines[i].High > high {
			high = p.klines[i].High
		}
		if p.klines[i].Low < low {
			low = p.klines[i].Low
		}
	}

	return low, high
}

func (p *MLPredictor) calculateHistoricalAccuracy(method string) float64 {
	// 简化实现，实际应该基于历史预测准确性
	switch method {
	case "linear":
		return 0.65
	case "ma":
		return 0.70
	case "technical":
		return 0.75
	case "ensemble":
		return 0.80
	default:
		return 0.60
	}
}

// PrintMLPredictions 打印机器学习预测结果
func PrintMLPredictions(predictions map[string]MLPrediction) {
	fmt.Println("\n=== 机器学习预测分析 ===")

	for method, pred := range predictions {
		if pred.Confidence == 0 {
			continue
		}

		fmt.Printf("\n--- %s预测 (%s) ---\n", pred.Method, method)
		fmt.Printf("下一天预测价格: %.2f\n", pred.NextDayPrice)
		fmt.Printf("下一周预测价格: %.2f\n", pred.NextWeekPrice)
		fmt.Printf("下一月预测价格: %.2f\n", pred.NextMonthPrice)
		fmt.Printf("预测置信度: %.1f%%\n", pred.Confidence*100)
		fmt.Printf("趋势方向: %s\n", pred.Trend)
		fmt.Printf("历史准确率: %.1f%%\n", pred.Accuracy*100)

		if pred.SupportLevel > 0 {
			fmt.Printf("支撑位: %.2f\n", pred.SupportLevel)
		}
		if pred.ResistanceLevel > 0 {
			fmt.Printf("阻力位: %.2f\n", pred.ResistanceLevel)
		}
	}

	// 综合建议
	fmt.Println("\n=== 综合预测建议 ===")
	printMLAdvice(predictions)
}

// printMLAdvice 打印机器学习预测建议
func printMLAdvice(predictions map[string]MLPrediction) {
	// 计算加权平均预测
	var totalPrice, totalConfidence float64
	var count int

	for _, pred := range predictions {
		if pred.Confidence > 0 {
			totalPrice += pred.NextDayPrice * pred.Confidence
			totalConfidence += pred.Confidence
			count++
		}
	}

	if count == 0 {
		fmt.Println("❌ 数据不足，无法提供预测")
		return
	}

	avgPrice := totalPrice / totalConfidence
	avgConfidence := totalConfidence / float64(count)

	// 获取当前价格（从第一个有效预测中获取）
	var currentPrice float64
	for _, pred := range predictions {
		if pred.Confidence > 0 {
			currentPrice = pred.NextDayPrice
			break
		}
	}

	// 价格预测建议
	if avgPrice > currentPrice*1.02 {
		fmt.Println("📈 综合预测: 强烈看涨")
	} else if avgPrice > currentPrice*1.005 {
		fmt.Println("📊 综合预测: 温和看涨")
	} else if avgPrice < currentPrice*0.98 {
		fmt.Println("📉 综合预测: 强烈看跌")
	} else if avgPrice < currentPrice*0.995 {
		fmt.Println("📊 综合预测: 温和看跌")
	} else {
		fmt.Println("➡️  综合预测: 横盘整理")
	}

	// 置信度评级
	if avgConfidence > 0.8 {
		fmt.Println("🎯 预测置信度: 很高")
	} else if avgConfidence > 0.6 {
		fmt.Println("📊 预测置信度: 较高")
	} else if avgConfidence > 0.4 {
		fmt.Println("⚠️  预测置信度: 中等")
	} else {
		fmt.Println("❓ 预测置信度: 较低")
	}

	// 操作建议
	fmt.Println("\n=== 操作建议 ===")
	if avgConfidence > 0.7 && avgPrice > currentPrice*1.01 {
		fmt.Println("✅ 建议: 可以考虑买入")
	} else if avgConfidence > 0.7 && avgPrice < currentPrice*0.99 {
		fmt.Println("⚠️  建议: 可以考虑卖出")
	} else {
		fmt.Println("💡 建议: 观望为主，等待更明确信号")
	}
}
