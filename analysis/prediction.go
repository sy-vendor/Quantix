package analysis

import (
	"Quantix/data"
	"fmt"
	"math"
)

type Prediction struct {
	Trend      string  // 趋势方向：上涨/下跌/震荡
	Confidence float64 // 预测置信度 0-100
	PriceRange struct {
		Min float64
		Max float64
	}
	SignalStrength string  // 信号强度：强/中/弱
	NextDayPred    float64 // 下一天预测价格
	RiskLevel      string  // 风险等级：高/中/低
	Recommendation string  // 操作建议
}

// 基于技术指标的趋势预测
func PredictTrend(klines []data.Kline) Prediction {
	if len(klines) < 30 {
		return Prediction{Trend: "数据不足", Confidence: 0}
	}

	factors := CalcFactors(klines)
	prices := extractPrices(klines)

	var pred Prediction

	// 1. 趋势判断
	pred.Trend = determineTrend(factors, prices)

	// 2. 置信度计算
	pred.Confidence = calculateConfidence(factors, prices)

	// 3. 价格区间预测
	pred.PriceRange = predictPriceRange(klines, factors)

	// 4. 信号强度
	pred.SignalStrength = calculateSignalStrength(factors)

	// 5. 下一天预测价格
	pred.NextDayPred = predictNextDayPrice(klines, factors)

	// 6. 风险等级
	pred.RiskLevel = assessRisk(factors, prices)

	// 7. 操作建议
	pred.Recommendation = generatePredictionRecommendation(pred)

	return pred
}

// 提取价格数据
func extractPrices(klines []data.Kline) []float64 {
	prices := make([]float64, len(klines))
	for i, kline := range klines {
		prices[i] = kline.Close
	}
	return prices
}

// 趋势判断
func determineTrend(factors Factors, prices []float64) string {
	// 计算短期和长期趋势
	shortTrend := 0
	longTrend := 0

	// 短期趋势（最近5天）
	if len(prices) >= 5 {
		recent5 := prices[len(prices)-5:]
		for i := 1; i < len(recent5); i++ {
			if recent5[i] > recent5[i-1] {
				shortTrend++
			} else if recent5[i] < recent5[i-1] {
				shortTrend--
			}
		}
	}

	// 长期趋势（最近20天）
	if len(prices) >= 20 {
		recent20 := prices[len(prices)-20:]
		for i := 1; i < len(recent20); i++ {
			if recent20[i] > recent20[i-1] {
				longTrend++
			} else if recent20[i] < recent20[i-1] {
				longTrend--
			}
		}
	}

	// 技术指标趋势
	maTrend := 0
	if factors.MA5 > factors.MA10 && factors.MA10 > factors.MA20 {
		maTrend = 1
	} else if factors.MA5 < factors.MA10 && factors.MA10 < factors.MA20 {
		maTrend = -1
	}

	macdTrend := 0
	if factors.MACD > factors.MACDSignal {
		macdTrend = 1
	} else {
		macdTrend = -1
	}

	// 综合判断
	totalScore := shortTrend + longTrend + maTrend*2 + macdTrend*2

	if totalScore >= 3 {
		return "上涨"
	} else if totalScore <= -3 {
		return "下跌"
	} else {
		return "震荡"
	}
}

// 计算预测置信度
func calculateConfidence(factors Factors, prices []float64) float64 {
	confidence := 50.0 // 基础置信度

	// RSI置信度
	if factors.RSI > 30 && factors.RSI < 70 {
		confidence += 15 // RSI在正常区间，置信度较高
	} else if factors.RSI < 20 || factors.RSI > 80 {
		confidence -= 10 // RSI极值，置信度降低
	}

	// MACD置信度
	if math.Abs(factors.MACD-factors.MACDSignal) > 0.1 {
		confidence += 10 // MACD信号明确
	} else {
		confidence -= 5 // MACD信号模糊
	}

	// 均线置信度
	if (factors.MA5 > factors.MA10 && factors.MA10 > factors.MA20) ||
		(factors.MA5 < factors.MA10 && factors.MA10 < factors.MA20) {
		confidence += 10 // 均线排列明确
	} else {
		confidence -= 5 // 均线混乱
	}

	// 波动率置信度
	if factors.Volatility > 0 {
		confidence += 5 // 有波动，预测有意义
	}

	// 限制在0-100范围内
	confidence = math.Max(0, math.Min(100, confidence))
	return confidence
}

// 预测价格区间
func predictPriceRange(klines []data.Kline, factors Factors) struct {
	Min float64
	Max float64
} {
	currentPrice := klines[len(klines)-1].Close
	volatility := factors.Volatility

	// 基于波动率预测区间
	rangeMultiplier := 2.0 // 2倍标准差
	minPrice := currentPrice - volatility*rangeMultiplier
	maxPrice := currentPrice + volatility*rangeMultiplier

	// 考虑趋势调整
	if factors.MACD > factors.MACDSignal {
		// 上涨趋势，向上调整
		minPrice = currentPrice - volatility*1.5
		maxPrice = currentPrice + volatility*2.5
	} else {
		// 下跌趋势，向下调整
		minPrice = currentPrice - volatility*2.5
		maxPrice = currentPrice + volatility*1.5
	}

	return struct {
		Min float64
		Max float64
	}{
		Min: math.Max(0, minPrice),
		Max: maxPrice,
	}
}

// 计算信号强度
func calculateSignalStrength(factors Factors) string {
	strength := 0

	// RSI信号
	if factors.RSI < 30 || factors.RSI > 70 {
		strength += 2 // 超买超卖信号强
	} else if factors.RSI < 40 || factors.RSI > 60 {
		strength += 1 // 中等信号
	}

	// MACD信号
	if math.Abs(factors.MACD-factors.MACDSignal) > 0.2 {
		strength += 2 // MACD信号强
	} else if math.Abs(factors.MACD-factors.MACDSignal) > 0.1 {
		strength += 1 // MACD信号中等
	}

	// 均线信号
	if (factors.MA5 > factors.MA10 && factors.MA10 > factors.MA20) ||
		(factors.MA5 < factors.MA10 && factors.MA10 < factors.MA20) {
		strength += 1 // 均线排列明确
	}

	if strength >= 4 {
		return "强"
	} else if strength >= 2 {
		return "中"
	} else {
		return "弱"
	}
}

// 预测下一天价格
func predictNextDayPrice(klines []data.Kline, factors Factors) float64 {
	currentPrice := klines[len(klines)-1].Close

	// 基于动量预测
	momentum := factors.Momentum

	// 基于MACD预测
	macdSignal := 0.0
	if factors.MACD > factors.MACDSignal {
		macdSignal = 0.01 // 上涨信号
	} else {
		macdSignal = -0.01 // 下跌信号
	}

	// 基于RSI预测
	rsiSignal := 0.0
	if factors.RSI < 30 {
		rsiSignal = 0.005 // 超卖反弹
	} else if factors.RSI > 70 {
		rsiSignal = -0.005 // 超买回调
	}

	// 综合预测
	predictedChange := momentum + currentPrice*(macdSignal+rsiSignal)
	predictedPrice := currentPrice + predictedChange

	return math.Max(0, predictedPrice)
}

// 评估风险等级
func assessRisk(factors Factors, prices []float64) string {
	riskScore := 0

	// 波动率风险
	if factors.Volatility > 1.0 {
		riskScore += 3 // 高波动率
	} else if factors.Volatility > 0.5 {
		riskScore += 2 // 中等波动率
	} else {
		riskScore += 1 // 低波动率
	}

	// RSI风险
	if factors.RSI > 80 || factors.RSI < 20 {
		riskScore += 2 // 极值风险
	} else if factors.RSI > 70 || factors.RSI < 30 {
		riskScore += 1 // 边界风险
	}

	// 换手率风险
	if factors.Turnover > 2.0 {
		riskScore += 2 // 高换手风险
	} else if factors.Turnover > 1.5 {
		riskScore += 1 // 中等换手风险
	}

	if riskScore >= 5 {
		return "高"
	} else if riskScore >= 3 {
		return "中"
	} else {
		return "低"
	}
}

// 生成操作建议
func generatePredictionRecommendation(pred Prediction) string {
	switch pred.Trend {
	case "上涨":
		if pred.Confidence >= 70 {
			return "建议买入，趋势明确"
		} else if pred.Confidence >= 50 {
			return "可考虑买入，注意风险"
		} else {
			return "观望为主，等待更明确信号"
		}
	case "下跌":
		if pred.Confidence >= 70 {
			return "建议卖出或减仓"
		} else if pred.Confidence >= 50 {
			return "谨慎持有，注意止损"
		} else {
			return "观望为主，避免追跌"
		}
	case "震荡":
		return "区间震荡，可做波段操作"
	default:
		return "数据不足，无法给出建议"
	}
}

// 打印预测结果
func PrintPrediction(pred Prediction, stockCode string) {
	fmt.Printf("=== %s 趋势预测 ===\n", stockCode)
	fmt.Printf("趋势方向:     %s\n", pred.Trend)
	fmt.Printf("预测置信度:   %.1f%%\n", pred.Confidence)
	fmt.Printf("信号强度:     %s\n", pred.SignalStrength)
	fmt.Printf("风险等级:     %s\n", pred.RiskLevel)
	fmt.Printf("价格区间:     %.2f - %.2f\n", pred.PriceRange.Min, pred.PriceRange.Max)
	fmt.Printf("下日预测:     %.2f\n", pred.NextDayPred)
	fmt.Printf("操作建议:     %s\n", pred.Recommendation)
}
