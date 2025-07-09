package analysis

// 回测参数
type BacktestParams struct {
	StrategyType   string  // 策略类型：ma_cross, breakout, rsi
	FastMAPeriod   int     // 快速均线周期
	SlowMAPeriod   int     // 慢速均线周期
	BreakoutPeriod int     // 突破周期
	RSIPeriod      int     // RSI周期
	RSIOverbought  float64 // RSI超买阈值
	RSIOversold    float64 // RSI超卖阈值
	StopLoss       float64 // 止损百分比
	TakeProfit     float64 // 止盈百分比
	InitialCash    float64 // 初始资金
}

// 回测结果
type BacktestResult struct {
	TotalReturn  float64   // 总收益率
	WinRate      float64   // 胜率
	MaxDrawdown  float64   // 最大回撤
	Trades       int       // 交易次数
	ProfitFactor float64   // 盈亏比
	EquityCurve  []float64 // 资金曲线
}

// 均线计算
func ma(prices []float64, period int, idx int) float64 {
	if idx+1 < period {
		return 0
	}
	sum := 0.0
	for i := idx + 1 - period; i <= idx; i++ {
		sum += prices[i]
	}
	return sum / float64(period)
}

// RSI计算
func rsi(prices []float64, period int, idx int) float64 {
	if idx < period {
		return 0
	}
	gain, loss := 0.0, 0.0
	for i := idx - period + 1; i <= idx; i++ {
		chg := prices[i] - prices[i-1]
		if chg > 0 {
			gain += chg
		} else {
			loss -= chg
		}
	}
	if gain+loss == 0 {
		return 50
	}
	avgGain := gain / float64(period)
	avgLoss := loss / float64(period)
	if avgLoss == 0 {
		return 100
	}
	RS := avgGain / avgLoss
	return 100 - 100/(1+RS)
}

// 主回测入口
func BacktestStrategy(stockData []StockData, params BacktestParams) BacktestResult {
	switch params.StrategyType {
	case "breakout":
		return backtestBreakout(stockData, params)
	case "rsi":
		return backtestRSI(stockData, params)
	default:
		return backtestMACross(stockData, params)
	}
}

// 均线交叉策略
func backtestMACross(stockData []StockData, params BacktestParams) BacktestResult {
	if len(stockData) == 0 {
		return BacktestResult{}
	}
	cash := params.InitialCash
	position := 0.0
	entryPrice := 0.0
	trades := 0
	wins := 0
	losses := 0
	profitSum := 0.0
	lossSum := 0.0
	maxEquity := cash
	equityCurve := []float64{cash}

	var closes []float64
	for _, d := range stockData {
		closes = append(closes, d.Close)
	}

	for i := params.SlowMAPeriod; i < len(stockData); i++ {
		fastMA := ma(closes, params.FastMAPeriod, i)
		slowMA := ma(closes, params.SlowMAPeriod, i)
		price := closes[i]

		if fastMA > slowMA && ma(closes, params.FastMAPeriod, i-1) <= ma(closes, params.SlowMAPeriod, i-1) && position == 0 {
			position = cash / price
			entryPrice = price
			cash = 0
			trades++
		}
		if fastMA < slowMA && ma(closes, params.FastMAPeriod, i-1) >= ma(closes, params.SlowMAPeriod, i-1) && position > 0 {
			profit := (price - entryPrice) * position
			cash = position * price
			if profit > 0 {
				wins++
				profitSum += profit
			} else {
				losses++
				lossSum += -profit
			}
			position = 0
			entryPrice = 0
		}
		if position > 0 {
			if price <= entryPrice*(1-params.StopLoss) {
				profit := (price - entryPrice) * position
				cash = position * price
				losses++
				lossSum += -profit
				position = 0
				entryPrice = 0
			}
			if price >= entryPrice*(1+params.TakeProfit) {
				profit := (price - entryPrice) * position
				cash = position * price
				wins++
				profitSum += profit
				position = 0
				entryPrice = 0
			}
		}
		equity := cash
		if position > 0 {
			equity += position * price
		}
		if equity > maxEquity {
			maxEquity = equity
		}
		curDrawdown := (maxEquity - equity) / maxEquity
		if curDrawdown > 0.5 {
			curDrawdown = 0.5
		}
		equityCurve = append(equityCurve, equity)
	}
	if position > 0 {
		cash += position * closes[len(closes)-1]
		profit := (closes[len(closes)-1] - entryPrice) * position
		if profit > 0 {
			wins++
			profitSum += profit
		} else {
			losses++
			lossSum += -profit
		}
		position = 0
	}
	finalEquity := cash
	if finalEquity < 0.01 {
		finalEquity = 0.01
	}
	maxDrawdown := 0.0
	peak := equityCurve[0]
	for _, eq := range equityCurve {
		if eq > peak {
			peak = eq
		}
		drawdown := (peak - eq) / peak
		if drawdown > maxDrawdown {
			maxDrawdown = drawdown
		}
	}
	winRate := 0.0
	if trades > 0 {
		winRate = float64(wins) / float64(trades)
	}
	profitFactor := 0.0
	if lossSum > 0 {
		profitFactor = profitSum / lossSum
	}
	return BacktestResult{
		TotalReturn:  (finalEquity - params.InitialCash) / params.InitialCash,
		WinRate:      winRate,
		MaxDrawdown:  maxDrawdown,
		Trades:       trades,
		ProfitFactor: profitFactor,
		EquityCurve:  equityCurve,
	}
}

// 突破策略
func backtestBreakout(stockData []StockData, params BacktestParams) BacktestResult {
	if len(stockData) == 0 || params.BreakoutPeriod < 2 {
		return BacktestResult{}
	}
	cash := params.InitialCash
	position := 0.0
	entryPrice := 0.0
	trades := 0
	wins := 0
	losses := 0
	profitSum := 0.0
	lossSum := 0.0
	maxEquity := cash
	equityCurve := []float64{cash}

	var closes []float64
	for _, d := range stockData {
		closes = append(closes, d.Close)
	}

	for i := params.BreakoutPeriod; i < len(stockData); i++ {
		price := closes[i]
		maxHigh := closes[i-params.BreakoutPeriod]
		for j := i - params.BreakoutPeriod + 1; j <= i; j++ {
			if closes[j] > maxHigh {
				maxHigh = closes[j]
			}
		}
		// 突破买入
		if price > maxHigh && position == 0 {
			position = cash / price
			entryPrice = price
			cash = 0
			trades++
		}
		// 跌破最低卖出
		minLow := closes[i-params.BreakoutPeriod]
		for j := i - params.BreakoutPeriod + 1; j <= i; j++ {
			if closes[j] < minLow {
				minLow = closes[j]
			}
		}
		if price < minLow && position > 0 {
			profit := (price - entryPrice) * position
			cash = position * price
			if profit > 0 {
				wins++
				profitSum += profit
			} else {
				losses++
				lossSum += -profit
			}
			position = 0
			entryPrice = 0
		}
		// 止损止盈
		if position > 0 {
			if price <= entryPrice*(1-params.StopLoss) {
				profit := (price - entryPrice) * position
				cash = position * price
				losses++
				lossSum += -profit
				position = 0
				entryPrice = 0
			}
			if price >= entryPrice*(1+params.TakeProfit) {
				profit := (price - entryPrice) * position
				cash = position * price
				wins++
				profitSum += profit
				position = 0
				entryPrice = 0
			}
		}
		equity := cash
		if position > 0 {
			equity += position * price
		}
		if equity > maxEquity {
			maxEquity = equity
		}
		curDrawdown := (maxEquity - equity) / maxEquity
		if curDrawdown > 0.5 {
			curDrawdown = 0.5
		}
		equityCurve = append(equityCurve, equity)
	}
	if position > 0 {
		cash += position * closes[len(closes)-1]
		profit := (closes[len(closes)-1] - entryPrice) * position
		if profit > 0 {
			wins++
			profitSum += profit
		} else {
			losses++
			lossSum += -profit
		}
		position = 0
	}
	finalEquity := cash
	if finalEquity < 0.01 {
		finalEquity = 0.01
	}
	maxDrawdown := 0.0
	peak := equityCurve[0]
	for _, eq := range equityCurve {
		if eq > peak {
			peak = eq
		}
		drawdown := (peak - eq) / peak
		if drawdown > maxDrawdown {
			maxDrawdown = drawdown
		}
	}
	winRate := 0.0
	if trades > 0 {
		winRate = float64(wins) / float64(trades)
	}
	profitFactor := 0.0
	if lossSum > 0 {
		profitFactor = profitSum / lossSum
	}
	return BacktestResult{
		TotalReturn:  (finalEquity - params.InitialCash) / params.InitialCash,
		WinRate:      winRate,
		MaxDrawdown:  maxDrawdown,
		Trades:       trades,
		ProfitFactor: profitFactor,
		EquityCurve:  equityCurve,
	}
}

// RSI策略
func backtestRSI(stockData []StockData, params BacktestParams) BacktestResult {
	if len(stockData) == 0 || params.RSIPeriod < 2 {
		return BacktestResult{}
	}
	cash := params.InitialCash
	position := 0.0
	entryPrice := 0.0
	trades := 0
	wins := 0
	losses := 0
	profitSum := 0.0
	lossSum := 0.0
	maxEquity := cash
	equityCurve := []float64{cash}

	var closes []float64
	for _, d := range stockData {
		closes = append(closes, d.Close)
	}

	for i := params.RSIPeriod; i < len(stockData); i++ {
		price := closes[i]
		rsiVal := rsi(closes, params.RSIPeriod, i)
		// 超卖买入
		if rsiVal < params.RSIOversold && position == 0 {
			position = cash / price
			entryPrice = price
			cash = 0
			trades++
		}
		// 超买卖出
		if rsiVal > params.RSIOverbought && position > 0 {
			profit := (price - entryPrice) * position
			cash = position * price
			if profit > 0 {
				wins++
				profitSum += profit
			} else {
				losses++
				lossSum += -profit
			}
			position = 0
			entryPrice = 0
		}
		// 止损止盈
		if position > 0 {
			if price <= entryPrice*(1-params.StopLoss) {
				profit := (price - entryPrice) * position
				cash = position * price
				losses++
				lossSum += -profit
				position = 0
				entryPrice = 0
			}
			if price >= entryPrice*(1+params.TakeProfit) {
				profit := (price - entryPrice) * position
				cash = position * price
				wins++
				profitSum += profit
				position = 0
				entryPrice = 0
			}
		}
		equity := cash
		if position > 0 {
			equity += position * price
		}
		if equity > maxEquity {
			maxEquity = equity
		}
		curDrawdown := (maxEquity - equity) / maxEquity
		if curDrawdown > 0.5 {
			curDrawdown = 0.5
		}
		equityCurve = append(equityCurve, equity)
	}
	if position > 0 {
		cash += position * closes[len(closes)-1]
		profit := (closes[len(closes)-1] - entryPrice) * position
		if profit > 0 {
			wins++
			profitSum += profit
		} else {
			losses++
			lossSum += -profit
		}
		position = 0
	}
	finalEquity := cash
	if finalEquity < 0.01 {
		finalEquity = 0.01
	}
	maxDrawdown := 0.0
	peak := equityCurve[0]
	for _, eq := range equityCurve {
		if eq > peak {
			peak = eq
		}
		drawdown := (peak - eq) / peak
		if drawdown > maxDrawdown {
			maxDrawdown = drawdown
		}
	}
	winRate := 0.0
	if trades > 0 {
		winRate = float64(wins) / float64(trades)
	}
	profitFactor := 0.0
	if lossSum > 0 {
		profitFactor = profitSum / lossSum
	}
	return BacktestResult{
		TotalReturn:  (finalEquity - params.InitialCash) / params.InitialCash,
		WinRate:      winRate,
		MaxDrawdown:  maxDrawdown,
		Trades:       trades,
		ProfitFactor: profitFactor,
		EquityCurve:  equityCurve,
	}
}
