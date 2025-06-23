package analysis_test

import (
	"Quantix/analysis"
	"Quantix/data"
	"math"
	"testing"
	"time"
)

func TestMLPredictionMethods(t *testing.T) {
	t.Log("=== 机器学习预测方法测试 ===")

	testCases := []struct {
		name   string
		klines []data.Kline
	}{
		{"上涨趋势数据", createUptrendData()},
		{"下跌趋势数据", createDowntrendData()},
		{"震荡数据", createSidewaysData()},
		{"正常数据", createTestData()},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			if len(tc.klines) < 30 {
				t.Skip("数据不足，跳过测试")
			}

			if len(tc.klines) <= 1 {
				t.Skip("K线数据不足，跳过ML建模测试")
			}
			factors := analysis.CalcFactors(tc.klines)
			if len(factors) <= 1 {
				t.Skip("因子数据不足，跳过ML建模测试")
			}

			predictor := analysis.NewMLPredictor(tc.klines, factors)
			predictions := predictor.PredictAll()

			if len(predictions) == 0 {
				t.Fatal("预测失败，没有返回任何预测结果")
			}

			// 测试每种预测方法
			for _, pred := range predictions {
				t.Run(pred.Method, func(t *testing.T) {
					// 测试预测价格
					t.Run("价格预测测试", func(t *testing.T) {
						if pred.NextDayPrice < 0 {
							t.Errorf("下一天价格不应为负: %.2f", pred.NextDayPrice)
						}
						if pred.NextWeekPrice < 0 {
							t.Errorf("下一周价格不应为负: %.2f", pred.NextWeekPrice)
						}
						if pred.NextMonthPrice < 0 {
							t.Errorf("下一月价格不应为负: %.2f", pred.NextMonthPrice)
						}
					})

					// 测试置信度
					t.Run("置信度测试", func(t *testing.T) {
						if pred.Confidence < 0 || pred.Confidence > 1 {
							t.Errorf("置信度应该在[0,1]范围内: %.3f", pred.Confidence)
						}
					})

					// 测试趋势
					t.Run("趋势测试", func(t *testing.T) {
						validTrends := []string{"上涨", "下跌", "震荡", "横盘"}
						found := false
						for _, trend := range validTrends {
							if pred.Trend == trend {
								found = true
								break
							}
						}
						if !found {
							t.Errorf("趋势值无效: %s", pred.Trend)
						}
					})

					// 测试准确率
					t.Run("准确率测试", func(t *testing.T) {
						if pred.Accuracy < 0 || pred.Accuracy > 1 {
							t.Errorf("准确率应该在[0,1]范围内: %.3f", pred.Accuracy)
						}
					})
				})
			}
		})
	}
}

func TestPredictionConsistency(t *testing.T) {
	t.Log("=== 预测一致性测试 ===")

	klines := createTestData()
	if len(klines) <= 1 {
		t.Skip("K线数据不足，跳过ML建模测试")
	}
	factors := analysis.CalcFactors(klines)
	if len(factors) <= 1 {
		t.Skip("因子数据不足，跳过ML建模测试")
	}

	predictor := analysis.NewMLPredictor(klines, factors)

	// 多次预测应该得到相似结果
	predictions1 := predictor.PredictAll()
	predictions2 := predictor.PredictAll()

	if len(predictions1) != len(predictions2) {
		t.Error("预测方法数量不一致")
	}

	for i, pred1 := range predictions1 {
		pred2 := predictions2[i]

		if pred1.Method != pred2.Method {
			t.Errorf("预测方法名称不一致: %s vs %s", pred1.Method, pred2.Method)
		}

		// 价格预测应该相近（允许小误差）
		if math.Abs(pred1.NextDayPrice-pred2.NextDayPrice) > 0.01 {
			t.Errorf("下一天价格预测不一致: %.2f vs %.2f", pred1.NextDayPrice, pred2.NextDayPrice)
		}

		// 置信度应该相近
		if math.Abs(pred1.Confidence-pred2.Confidence) > 0.01 {
			t.Errorf("置信度不一致: %.3f vs %.3f", pred1.Confidence, pred2.Confidence)
		}
	}

	t.Log("预测结果一致")
}

func TestPredictionEdgeCases(t *testing.T) {
	t.Log("=== 预测边界情况测试 ===")

	t.Run("数据不足测试", func(t *testing.T) {
		klines := createTestData()[:15] // 只有15条数据
		if len(klines) <= 1 {
			t.Skip("K线数据不足，跳过ML建模测试")
		}
		factors := analysis.CalcFactors(klines)
		if len(factors) <= 1 {
			t.Skip("因子数据不足，跳过ML建模测试")
		}

		predictor := analysis.NewMLPredictor(klines, factors)
		predictions := predictor.PredictAll()

		// 数据不足时可能返回空预测或低置信度预测
		for _, pred := range predictions {
			if pred.Confidence > 0.8 {
				t.Errorf("数据不足时置信度不应该过高: %.3f", pred.Confidence)
			}
		}
	})

	t.Run("极值数据测试", func(t *testing.T) {
		// 创建包含极值的数据
		klines := createExtremeData()
		if len(klines) <= 1 {
			t.Skip("K线数据不足，跳过ML建模测试")
		}
		factors := analysis.CalcFactors(klines)
		if len(factors) <= 1 {
			t.Skip("因子数据不足，跳过ML建模测试")
		}

		predictor := analysis.NewMLPredictor(klines, factors)
		predictions := predictor.PredictAll()

		for _, pred := range predictions {
			if math.IsNaN(pred.NextDayPrice) || math.IsInf(pred.NextDayPrice, 0) {
				t.Error("预测价格不应该是NaN或无穷大")
			}
			if math.IsNaN(pred.Confidence) || math.IsInf(pred.Confidence, 0) {
				t.Error("置信度不应该是NaN或无穷大")
			}
		}
	})
}

func TestPredictionAccuracy(t *testing.T) {
	t.Log("=== 预测准确性测试 ===")

	klines := createTestData()
	if len(klines) <= 1 {
		t.Skip("K线数据不足，跳过ML建模测试")
	}
	factors := analysis.CalcFactors(klines)
	if len(factors) <= 1 {
		t.Skip("因子数据不足，跳过ML建模测试")
	}

	predictor := analysis.NewMLPredictor(klines, factors)
	predictions := predictor.PredictAll()

	for _, pred := range predictions {
		t.Run(pred.Method+"准确性测试", func(t *testing.T) {
			// 检查准确率是否合理
			if pred.Accuracy > 0.95 {
				t.Logf("警告: %s方法准确率过高(%.1f%%)，可能存在过拟合", pred.Method, pred.Accuracy*100)
			}

			if pred.Accuracy < 0.3 {
				t.Logf("警告: %s方法准确率过低(%.1f%%)，可能需要优化", pred.Method, pred.Accuracy*100)
			}
		})
	}
}

func TestPredictionMethodsComparison(t *testing.T) {
	t.Log("=== 预测方法对比测试 ===")

	klines := createTestData()
	if len(klines) <= 1 {
		t.Skip("K线数据不足，跳过ML建模测试")
	}
	factors := analysis.CalcFactors(klines)
	if len(factors) <= 1 {
		t.Skip("因子数据不足，跳过ML建模测试")
	}

	predictor := analysis.NewMLPredictor(klines, factors)
	predictions := predictor.PredictAll()

	// 统计各种方法的性能
	methodStats := make(map[string]struct {
		count    int
		avgConf  float64
		avgAcc   float64
		avgPrice float64
	})

	for _, pred := range predictions {
		stats := methodStats[pred.Method]
		stats.count++
		stats.avgConf += pred.Confidence
		stats.avgAcc += pred.Accuracy
		stats.avgPrice += pred.NextDayPrice
		methodStats[pred.Method] = stats
	}

	// 计算平均值
	for method, stats := range methodStats {
		if stats.count > 0 {
			stats.avgConf /= float64(stats.count)
			stats.avgAcc /= float64(stats.count)
			stats.avgPrice /= float64(stats.count)
			methodStats[method] = stats
		}
	}
}

// 创建极值数据用于测试
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
