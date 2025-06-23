package analysis

import (
	"Quantix/data"
	"math"
)

// Factors 技术指标结构
type Factors struct {
	Code       string
	Date       string
	Close      float64
	Volume     float64
	MA5        float64
	MA10       float64
	MA20       float64
	MA30       float64
	Momentum   float64
	Volatility float64
	Turnover   float64
	RSI        float64
	MACD       float64
	MACDSignal float64
	MACDHist   float64
	// 新增指标
	BBUpper    float64 // 布林带上轨
	BBMiddle   float64 // 布林带中轨
	BBLower    float64 // 布林带下轨
	BBWidth    float64 // 布林带宽度
	BBPosition float64 // 布林带位置
	KDJ_K      float64 // KDJ指标K值
	KDJ_D      float64 // KDJ指标D值
	KDJ_J      float64 // KDJ指标J值
	WR         float64 // 威廉指标
	CCI        float64 // 顺势指标
	ATR        float64 // 平均真实波幅
	OBV        float64 // 能量潮指标
}

// CalcFactors 计算技术指标
func CalcFactors(klines []data.Kline) []Factors {
	if len(klines) < 30 {
		return nil
	}

	factors := make([]Factors, len(klines))

	for i, kline := range klines {
		factors[i] = Factors{
			Code:   "", // 暂时留空，后续可以从参数传入
			Date:   kline.Date.Format("2006-01-02"),
			Close:  kline.Close,
			Volume: float64(kline.Volume),
		}

		// 移动平均线
		if i >= 4 {
			factors[i].MA5 = calcMA(klines, i, 5)
		}
		if i >= 9 {
			factors[i].MA10 = calcMA(klines, i, 10)
		}
		if i >= 19 {
			factors[i].MA20 = calcMA(klines, i, 20)
		}
		if i >= 29 {
			factors[i].MA30 = calcMA(klines, i, 30)
		}

		// 动量指标
		if i >= 10 {
			factors[i].Momentum = calcMomentum(klines, i, 10)
		}

		// 波动率
		if i >= 20 {
			factors[i].Volatility = calcVolatility(klines, i, 20)
		}

		// 换手率
		if i >= 20 {
			factors[i].Turnover = calcTurnover(klines, i, 20)
		}

		// RSI
		if i >= 14 {
			factors[i].RSI = calcRSI(klines, i, 14)
		}

		// MACD
		if i >= 26 {
			factors[i].MACD, factors[i].MACDSignal, factors[i].MACDHist = calcMACD(klines, i)
		}

		// 布林带
		if i >= 20 {
			factors[i].BBUpper, factors[i].BBMiddle, factors[i].BBLower, factors[i].BBWidth, factors[i].BBPosition = calcBollingerBands(klines, i, 20)
		}

		// KDJ
		if i >= 9 {
			factors[i].KDJ_K, factors[i].KDJ_D, factors[i].KDJ_J = calcKDJ(klines, i, 9)
		}

		// 威廉指标
		if i >= 14 {
			factors[i].WR = calcWilliamsR(klines, i, 14)
		}

		// CCI
		if i >= 20 {
			factors[i].CCI = calcCCI(klines, i, 20)
		}

		// ATR
		if i >= 14 {
			factors[i].ATR = calcATR(klines, i, 14)
		}

		// OBV
		if i >= 1 {
			factors[i].OBV = calcOBV(klines, i)
		}
	}

	return factors
}

// 计算移动平均线
func calcMA(klines []data.Kline, index, period int) float64 {
	sum := 0.0
	for i := index - period + 1; i <= index; i++ {
		sum += klines[i].Close
	}
	return sum / float64(period)
}

// 计算动量
func calcMomentum(klines []data.Kline, index, period int) float64 {
	if index < period {
		return 0
	}
	return (klines[index].Close - klines[index-period].Close) / klines[index-period].Close * 100
}

// 计算波动率
func calcVolatility(klines []data.Kline, index, period int) float64 {
	if index < period {
		return 0
	}

	var returns []float64
	for i := index - period + 1; i <= index; i++ {
		if i > 0 {
			returns = append(returns, (klines[i].Close-klines[i-1].Close)/klines[i-1].Close)
		}
	}

	if len(returns) == 0 {
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

	return math.Sqrt(variance) * 100
}

// 计算换手率
func calcTurnover(klines []data.Kline, index, period int) float64 {
	if index < period {
		return 0
	}

	totalVolume := 0.0
	for i := index - period + 1; i <= index; i++ {
		totalVolume += float64(klines[i].Volume)
	}

	avgVolume := totalVolume / float64(period)
	if avgVolume == 0 {
		return 0
	}

	return float64(klines[index].Volume) / avgVolume
}

// 计算RSI
func calcRSI(klines []data.Kline, index, period int) float64 {
	if index < period {
		return 50
	}

	var gains, losses []float64
	for i := index - period + 1; i <= index; i++ {
		if i > 0 {
			change := klines[i].Close - klines[i-1].Close
			if change > 0 {
				gains = append(gains, change)
				losses = append(losses, 0)
			} else {
				gains = append(gains, 0)
				losses = append(losses, -change)
			}
		}
	}

	avgGain := 0.0
	avgLoss := 0.0
	for _, gain := range gains {
		avgGain += gain
	}
	for _, loss := range losses {
		avgLoss += loss
	}
	avgGain /= float64(len(gains))
	avgLoss /= float64(len(losses))

	if avgLoss == 0 {
		return 100
	}

	rs := avgGain / avgLoss
	return 100 - (100 / (1 + rs))
}

// 计算MACD
func calcMACD(klines []data.Kline, index int) (float64, float64, float64) {
	if index < 26 {
		return 0, 0, 0
	}

	ema12 := calcEMA(klines, index, 12)
	ema26 := calcEMA(klines, index, 26)
	macd := ema12 - ema26

	// 计算MACD信号线（9日EMA）
	signal := calcEMASignal(klines, index, macd)
	histogram := macd - signal

	return macd, signal, histogram
}

// 计算EMA
func calcEMA(klines []data.Kline, index, period int) float64 {
	if index < period-1 {
		return 0
	}

	multiplier := 2.0 / float64(period+1)
	ema := klines[index-period+1].Close

	for i := index - period + 2; i <= index; i++ {
		ema = (klines[i].Close * multiplier) + (ema * (1 - multiplier))
	}

	return ema
}

// 计算MACD信号线
func calcEMASignal(klines []data.Kline, index int, currentMACD float64) float64 {
	// 简化实现，实际应该维护MACD历史数据
	multiplier := 2.0 / 10.0
	return currentMACD * multiplier
}

// 计算布林带
func calcBollingerBands(klines []data.Kline, index, period int) (float64, float64, float64, float64, float64) {
	if index < period-1 {
		return 0, 0, 0, 0, 0
	}

	ma := calcMA(klines, index, period)

	// 计算标准差
	sum := 0.0
	for i := index - period + 1; i <= index; i++ {
		sum += (klines[i].Close - ma) * (klines[i].Close - ma)
	}
	stdDev := math.Sqrt(sum / float64(period))

	upper := ma + (2 * stdDev)
	lower := ma - (2 * stdDev)
	width := (upper - lower) / ma * 100
	position := (klines[index].Close - lower) / (upper - lower) * 100

	return upper, ma, lower, width, position
}

// 计算KDJ
func calcKDJ(klines []data.Kline, index, period int) (float64, float64, float64) {
	if index < period-1 {
		return 50, 50, 50
	}

	// 计算RSV
	high := klines[index].High
	low := klines[index].Low
	for i := index - period + 1; i <= index; i++ {
		if klines[i].High > high {
			high = klines[i].High
		}
		if klines[i].Low < low {
			low = klines[i].Low
		}
	}

	rsv := 0.0
	if high != low {
		rsv = (klines[index].Close - low) / (high - low) * 100
	}

	// 计算K值（简化实现）
	k := 0.67*50 + 0.33*rsv // 假设前一日K值为50
	d := 0.67*50 + 0.33*k   // 假设前一日D值为50
	j := 3*k - 2*d

	return k, d, j
}

// 计算威廉指标
func calcWilliamsR(klines []data.Kline, index, period int) float64 {
	if index < period-1 {
		return -50
	}

	high := klines[index].High
	low := klines[index].Low
	for i := index - period + 1; i <= index; i++ {
		if klines[i].High > high {
			high = klines[i].High
		}
		if klines[i].Low < low {
			low = klines[i].Low
		}
	}

	if high == low {
		return -50
	}

	return (high - klines[index].Close) / (high - low) * -100
}

// 计算CCI
func calcCCI(klines []data.Kline, index, period int) float64 {
	if index < period-1 {
		return 0
	}

	// 计算典型价格
	tp := (klines[index].High + klines[index].Low + klines[index].Close) / 3

	// 计算平均典型价格
	sum := 0.0
	for i := index - period + 1; i <= index; i++ {
		sum += (klines[i].High + klines[i].Low + klines[i].Close) / 3
	}
	avgTP := sum / float64(period)

	// 计算平均偏差
	sumDev := 0.0
	for i := index - period + 1; i <= index; i++ {
		tp_i := (klines[i].High + klines[i].Low + klines[i].Close) / 3
		sumDev += math.Abs(tp_i - avgTP)
	}
	avgDev := sumDev / float64(period)

	if avgDev == 0 {
		return 0
	}

	return (tp - avgTP) / (0.015 * avgDev)
}

// 计算ATR
func calcATR(klines []data.Kline, index, period int) float64 {
	if index < period-1 {
		return 0
	}

	var trs []float64
	for i := index - period + 1; i <= index; i++ {
		tr := klines[i].High - klines[i].Low
		if i > 0 {
			hl := klines[i].High - klines[i-1].Close
			lh := klines[i-1].Close - klines[i].Low
			if hl > tr {
				tr = hl
			}
			if lh > tr {
				tr = lh
			}
		}
		trs = append(trs, tr)
	}

	sum := 0.0
	for _, tr := range trs {
		sum += tr
	}

	return sum / float64(len(trs))
}

// 计算OBV
func calcOBV(klines []data.Kline, index int) float64 {
	if index == 0 {
		return float64(klines[0].Volume)
	}

	obv := calcOBV(klines, index-1)
	if klines[index].Close > klines[index-1].Close {
		obv += float64(klines[index].Volume)
	} else if klines[index].Close < klines[index-1].Close {
		obv -= float64(klines[index].Volume)
	}

	return obv
}
