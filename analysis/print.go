package analysis

import (
	"fmt"
)

// PrintAnalysis 打印技术分析结果
func PrintAnalysis(factors []Factors) {
	if len(factors) == 0 {
		fmt.Println("没有足够的数据进行分析")
		return
	}

	latest := factors[len(factors)-1]
	fmt.Println("\n=== 技术指标分析 ===")
	fmt.Printf("日期: %s\n", latest.Date)
	fmt.Printf("收盘价: %.2f\n", latest.Close)
	fmt.Printf("成交量: %.0f\n", latest.Volume)

	fmt.Println("\n--- 移动平均线 ---")
	fmt.Printf("MA5:  %.2f\n", latest.MA5)
	fmt.Printf("MA10: %.2f\n", latest.MA10)
	fmt.Printf("MA20: %.2f\n", latest.MA20)
	fmt.Printf("MA30: %.2f\n", latest.MA30)

	fmt.Println("\n--- 动量指标 ---")
	fmt.Printf("动量(10日): %.2f%%\n", latest.Momentum)
	fmt.Printf("波动率(20日): %.2f%%\n", latest.Volatility)
	fmt.Printf("换手率: %.2f\n", latest.Turnover)

	fmt.Println("\n--- 超买超卖指标 ---")
	fmt.Printf("RSI(14): %.2f\n", latest.RSI)
	fmt.Printf("威廉指标(14): %.2f\n", latest.WR)
	fmt.Printf("KDJ-K: %.2f\n", latest.KDJ_K)
	fmt.Printf("KDJ-D: %.2f\n", latest.KDJ_D)
	fmt.Printf("KDJ-J: %.2f\n", latest.KDJ_J)

	fmt.Println("\n--- 趋势指标 ---")
	fmt.Printf("MACD: %.4f\n", latest.MACD)
	fmt.Printf("MACD信号线: %.4f\n", latest.MACDSignal)
	fmt.Printf("MACD柱状图: %.4f\n", latest.MACDHist)
	fmt.Printf("CCI(20): %.2f\n", latest.CCI)

	fmt.Println("\n--- 布林带 ---")
	fmt.Printf("上轨: %.2f\n", latest.BBUpper)
	fmt.Printf("中轨: %.2f\n", latest.BBMiddle)
	fmt.Printf("下轨: %.2f\n", latest.BBLower)
	fmt.Printf("带宽: %.2f%%\n", latest.BBWidth)
	fmt.Printf("位置: %.2f%%\n", latest.BBPosition)

	fmt.Println("\n--- 其他指标 ---")
	fmt.Printf("ATR(14): %.2f\n", latest.ATR)
	fmt.Printf("OBV: %.0f\n", latest.OBV)

	// 技术分析建议
	fmt.Println("\n=== 技术分析建议 ===")
	printTechnicalAdvice(latest)
}

// printTechnicalAdvice 打印技术分析建议
func printTechnicalAdvice(factor Factors) {
	// RSI分析
	if factor.RSI > 70 {
		fmt.Println("⚠️  RSI超买区域，注意回调风险")
	} else if factor.RSI < 30 {
		fmt.Println("📈 RSI超卖区域，可能存在反弹机会")
	} else {
		fmt.Println("✅ RSI处于正常区间")
	}

	// MACD分析
	if factor.MACD > factor.MACDSignal && factor.MACDHist > 0 {
		fmt.Println("📈 MACD金叉，趋势向上")
	} else if factor.MACD < factor.MACDSignal && factor.MACDHist < 0 {
		fmt.Println("📉 MACD死叉，趋势向下")
	} else {
		fmt.Println("➡️  MACD趋势不明显")
	}

	// 布林带分析
	if factor.BBPosition > 80 {
		fmt.Println("⚠️  接近布林带上轨，注意回调")
	} else if factor.BBPosition < 20 {
		fmt.Println("📈 接近布林带下轨，可能存在支撑")
	} else {
		fmt.Println("✅ 价格在布林带中轨附近")
	}

	// KDJ分析
	if factor.KDJ_J > 80 {
		fmt.Println("⚠️  KDJ超买，注意风险")
	} else if factor.KDJ_J < 20 {
		fmt.Println("📈 KDJ超卖，关注反弹")
	} else {
		fmt.Println("✅ KDJ处于正常区间")
	}

	// 威廉指标分析
	if factor.WR > -20 {
		fmt.Println("⚠️  威廉指标超买")
	} else if factor.WR < -80 {
		fmt.Println("📈 威廉指标超卖")
	} else {
		fmt.Println("✅ 威廉指标正常")
	}

	// CCI分析
	if factor.CCI > 100 {
		fmt.Println("📈 CCI显示强势")
	} else if factor.CCI < -100 {
		fmt.Println("📉 CCI显示弱势")
	} else {
		fmt.Println("➡️  CCI中性")
	}

	// 移动平均线分析
	if factor.Close > factor.MA5 && factor.MA5 > factor.MA10 && factor.MA10 > factor.MA20 {
		fmt.Println("📈 短期均线多头排列，趋势向上")
	} else if factor.Close < factor.MA5 && factor.MA5 < factor.MA10 && factor.MA10 < factor.MA20 {
		fmt.Println("📉 短期均线空头排列，趋势向下")
	} else {
		fmt.Println("➡️  均线趋势不明显")
	}
}
