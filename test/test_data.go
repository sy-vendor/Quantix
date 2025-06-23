package analysis

import (
	"Quantix/data"
	"math"
	"time"
)

// 创建测试数据1 - 正常上涨趋势
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

// 创建测试数据2 - 高价股票
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

// 创建低波动性数据 - 价格变化很小
func createLowVolatilityData() []data.Kline {
	var klines []data.Kline
	basePrice := 100.0
	baseDate := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)

	for i := 0; i < 60; i++ {
		date := baseDate.AddDate(0, 0, i)
		// 低波动性：价格变化很小
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

// 创建高波动性数据 - 价格变化很大
func createHighVolatilityData() []data.Kline {
	var klines []data.Kline
	basePrice := 50.0
	baseDate := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)

	for i := 0; i < 60; i++ {
		date := baseDate.AddDate(0, 0, i)
		// 高波动性：价格变化很大
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

// 创建趋势性数据 - 价格持续上涨
func createTrendingData() []data.Kline {
	var klines []data.Kline
	basePrice := 20.0
	baseDate := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)

	for i := 0; i < 60; i++ {
		date := baseDate.AddDate(0, 0, i)
		// 趋势性：价格持续上涨
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

// 创建上涨趋势数据 - 持续上涨
func createUptrendData() []data.Kline {
	var klines []data.Kline
	basePrice := 50.0
	baseDate := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)

	for i := 0; i < 60; i++ {
		date := baseDate.AddDate(0, 0, i)
		// 持续上涨趋势
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

// 创建下跌趋势数据 - 持续下跌
func createDowntrendData() []data.Kline {
	var klines []data.Kline
	basePrice := 100.0
	baseDate := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)

	for i := 0; i < 60; i++ {
		date := baseDate.AddDate(0, 0, i)
		// 持续下跌趋势
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

// 创建震荡数据 - 价格在一定范围内波动
func createSidewaysData() []data.Kline {
	var klines []data.Kline
	basePrice := 30.0
	baseDate := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)

	for i := 0; i < 60; i++ {
		date := baseDate.AddDate(0, 0, i)
		// 震荡趋势：价格在一定范围内波动
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

// 创建极值数据 - 包含极值价格
func createExtremeData() []data.Kline {
	var klines []data.Kline
	basePrice := 100.0
	baseDate := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)

	for i := 0; i < 60; i++ {
		date := baseDate.AddDate(0, 0, i)

		// 创建极值价格
		var price float64
		if i%10 == 0 {
			price = basePrice * 2 // 极值高点
		} else if i%10 == 5 {
			price = basePrice * 0.5 // 极值低点
		} else {
			price = basePrice + float64(i)*0.1
		}

		high := price * 1.1
		low := price * 0.9
		open := price * 0.95
		volume := int64(1000000 + i*50000)

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

// 创建周期性数据 - 价格呈现周期性变化
func createCyclicalData() []data.Kline {
	var klines []data.Kline
	basePrice := 50.0
	baseDate := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)

	for i := 0; i < 60; i++ {
		date := baseDate.AddDate(0, 0, i)
		// 周期性变化：使用正弦函数
		cycle := math.Sin(float64(i) * 2 * math.Pi / 20) // 20天一个周期
		price := basePrice + cycle*10
		high := price * 1.02
		low := price * 0.98
		open := price * 1.0
		volume := int64(1000000 + int64(cycle*100000))

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

// 创建跳跃性数据 - 价格有突然的跳跃
func createJumpData() []data.Kline {
	var klines []data.Kline
	basePrice := 25.0
	baseDate := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)

	for i := 0; i < 60; i++ {
		date := baseDate.AddDate(0, 0, i)

		// 跳跃性变化
		var price float64
		if i == 15 {
			price = basePrice * 1.5 // 突然上涨50%
		} else if i == 35 {
			price = basePrice * 0.7 // 突然下跌30%
		} else {
			price = basePrice + float64(i)*0.1
		}

		high := price * 1.03
		low := price * 0.97
		open := price * 1.0
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
