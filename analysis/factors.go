package analysis

import (
	"Quantix/data"
	"math"
)

type Factors struct {
	MA5        float64
	MA10       float64
	MA20       float64
	Momentum   float64
	Volatility float64
	Turnover   float64 // 换手率
	RSI        float64 // RSI指标
	MACD       float64 // MACD指标
	MACDSignal float64 // MACD信号线
	MACDHist   float64 // MACD柱状图
}

// 计算移动平均线
func calculateMA(prices []float64, period int) float64 {
	if len(prices) < period {
		return 0
	}
	sum := 0.0
	for i := len(prices) - period; i < len(prices); i++ {
		sum += prices[i]
	}
	return sum / float64(period)
}

// 计算RSI指标
func calculateRSI(prices []float64, period int) float64 {
	if len(prices) < period+1 {
		return 0
	}
	var gains, losses float64
	for i := 1; i <= period; i++ {
		change := prices[len(prices)-i] - prices[len(prices)-i-1]
		if change > 0 {
			gains += change
		} else {
			losses -= change
		}
	}
	if losses == 0 {
		return 100
	}
	rs := gains / losses
	return 100 - (100 / (1 + rs))
}

// 计算MACD指标
func calculateMACD(prices []float64) (float64, float64, float64) {
	if len(prices) < 26 {
		return 0, 0, 0
	}
	ema12 := calculateEMA(prices, 12)
	ema26 := calculateEMA(prices, 26)
	macd := ema12 - ema26

	// 计算MACD的EMA作为信号线
	macdValues := make([]float64, 0)
	for i := 26; i < len(prices); i++ {
		ema12 := calculateEMA(prices[:i+1], 12)
		ema26 := calculateEMA(prices[:i+1], 26)
		macdValues = append(macdValues, ema12-ema26)
	}
	signal := calculateEMA(macdValues, 9)
	histogram := macd - signal

	return macd, signal, histogram
}

// 计算指数移动平均线
func calculateEMA(prices []float64, period int) float64 {
	if len(prices) < period {
		return 0
	}
	multiplier := 2.0 / float64(period+1)
	ema := prices[len(prices)-period]
	for i := len(prices) - period + 1; i < len(prices); i++ {
		ema = (prices[i] * multiplier) + (ema * (1 - multiplier))
	}
	return ema
}

// 计算换手率（简化版，基于成交量）
func calculateTurnover(klines []data.Kline) float64 {
	if len(klines) < 20 {
		return 0
	}
	// 计算最近20日的平均成交量
	var avgVolume float64
	for i := len(klines) - 20; i < len(klines); i++ {
		avgVolume += float64(klines[i].Volume)
	}
	avgVolume /= 20

	// 安全检查：避免除零
	if avgVolume == 0 {
		return 0
	}

	// 最新成交量与平均成交量的比值
	latestVolume := float64(klines[len(klines)-1].Volume)
	return latestVolume / avgVolume
}

// CalcFactors 计算量化因子
func CalcFactors(klines []data.Kline) Factors {
	n := len(klines)
	if n == 0 {
		return Factors{}
	}

	// 提取收盘价
	prices := make([]float64, n)
	for i, kline := range klines {
		prices[i] = kline.Close
	}

	// 计算基础指标
	ma5 := calculateMA(prices, 5)
	ma10 := calculateMA(prices, 10)
	ma20 := calculateMA(prices, 20)

	var momentum float64
	if n >= 2 {
		momentum = prices[n-1] - prices[n-2]
	}

	// 计算波动率
	var volatility float64
	if n >= 2 {
		mean := 0.0
		for _, price := range prices {
			mean += price
		}
		mean /= float64(n)
		for _, price := range prices {
			volatility += (price - mean) * (price - mean)
		}
		volatility = math.Sqrt(volatility / float64(n))
	}

	// 计算技术指标
	turnover := calculateTurnover(klines)
	rsi := calculateRSI(prices, 14)
	macd, macdSignal, macdHist := calculateMACD(prices)

	return Factors{
		MA5:        ma5,
		MA10:       ma10,
		MA20:       ma20,
		Momentum:   momentum,
		Volatility: volatility,
		Turnover:   turnover,
		RSI:        rsi,
		MACD:       macd,
		MACDSignal: macdSignal,
		MACDHist:   macdHist,
	}
}
