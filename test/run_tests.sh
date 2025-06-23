#!/bin/bash

# Quantix 测试运行脚本
# 使用方法: ./run_tests.sh [选项]

echo "=== Quantix 测试套件 ==="
echo ""

# 默认运行所有测试
RUN_ALL=true
RUN_FACTORS=false
RUN_RISK=false
RUN_ML=false
RUN_COMPARE=false
VERBOSE=false

# 解析命令行参数
while [[ $# -gt 0 ]]; do
    case $1 in
        --factors)
            RUN_FACTORS=true
            RUN_ALL=false
            shift
            ;;
        --risk)
            RUN_RISK=true
            RUN_ALL=false
            shift
            ;;
        --ml)
            RUN_ML=true
            RUN_ALL=false
            shift
            ;;
        --compare)
            RUN_COMPARE=true
            RUN_ALL=false
            shift
            ;;
        --verbose|-v)
            VERBOSE=true
            shift
            ;;
        --help|-h)
            echo "使用方法: $0 [选项]"
            echo ""
            echo "选项:"
            echo "  --factors     运行技术指标测试"
            echo "  --risk        运行风险管理测试"
            echo "  --ml          运行机器学习预测测试"
            echo "  --compare     运行股票对比测试"
            echo "  --verbose, -v 详细输出"
            echo "  --help, -h    显示帮助信息"
            echo ""
            echo "示例:"
            echo "  $0                    # 运行所有测试"
            echo "  $0 --factors --risk   # 运行技术指标和风险管理测试"
            echo "  $0 --verbose          # 详细输出所有测试"
            exit 0
            ;;
        *)
            echo "未知选项: $1"
            echo "使用 --help 查看帮助信息"
            exit 1
            ;;
    esac
done

# 设置测试参数
TEST_ARGS=""
if [ "$VERBOSE" = true ]; then
    TEST_ARGS="-v"
fi

# 检查是否在正确的目录
if [ ! -f "go.mod" ]; then
    echo "错误: 请在项目根目录运行此脚本"
    exit 1
fi

echo "开始运行测试..."
echo ""

# 运行所有测试
if [ "$RUN_ALL" = true ] || [ "$RUN_FACTORS" = true ]; then
    echo "=== 运行技术指标测试 ==="
    go test ./test -run "TestTechnicalIndicators|TestMovingAverages|TestVolumeIndicators|TestMomentumIndicators|TestVolatilityIndicators|TestDataConsistency" $TEST_ARGS
    echo ""
fi

if [ "$RUN_ALL" = true ] || [ "$RUN_RISK" = true ]; then
    echo "=== 运行风险管理测试 ==="
    go test ./test -run "TestRiskMetricsCalculation|TestRiskRating|TestRiskMetricsEdgeCases|TestRiskMetricsConsistency" $TEST_ARGS
    echo ""
fi

if [ "$RUN_ALL" = true ] || [ "$RUN_ML" = true ]; then
    echo "=== 运行机器学习预测测试 ==="
    go test ./test -run "TestMLPredictionMethods|TestPredictionConsistency|TestPredictionEdgeCases|TestPredictionAccuracy|TestPredictionMethodsComparison" $TEST_ARGS
    echo ""
fi

if [ "$RUN_ALL" = true ] || [ "$RUN_COMPARE" = true ]; then
    echo "=== 运行股票对比测试 ==="
    go test ./test -run "TestStockComparison|TestComparisonAccuracy|TestComparisonEdgeCases" $TEST_ARGS
    echo ""
fi

if [ "$RUN_ALL" = true ]; then
    echo "=== 运行所有测试 ==="
    go test ./test $TEST_ARGS
    echo ""
fi

echo "=== 测试完成 ==="
echo ""
echo "测试报告:"
echo "- 技术指标测试: 验证各种技术指标的计算准确性"
echo "- 风险管理测试: 验证风险指标的计算和评级"
echo "- 机器学习测试: 验证预测方法的有效性"
echo "- 股票对比测试: 验证多股票对比功能"
echo ""
echo "如需查看详细测试结果，请使用 --verbose 选项" 