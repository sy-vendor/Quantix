package api

import (
	"Quantix/analysis"
	"Quantix/config"
	"Quantix/data"
	"Quantix/logger"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

// Server API服务器
type Server struct {
	config   *config.Config
	router   *gin.Engine
	upgrader websocket.Upgrader
}

// NewServer 创建新的API服务器
func NewServer(cfg *config.Config) *Server {
	gin.SetMode(gin.ReleaseMode)
	router := gin.New()
	router.Use(gin.Recovery())
	router.Use(corsMiddleware())
	router.Use(loggingMiddleware())

	server := &Server{
		config: cfg,
		router: router,
		upgrader: websocket.Upgrader{
			CheckOrigin: func(r *http.Request) bool {
				return true // 允许所有来源
			},
		},
	}

	server.setupRoutes()
	return server
}

// setupRoutes 设置路由
func (server *Server) setupRoutes() {
	// 健康检查
	server.router.GET("/health", server.healthCheck)

	// API v1
	v1 := server.router.Group("/api/v1")
	{
		// 股票分析
		v1.GET("/stock/:code", server.getStockAnalysis)
		v1.POST("/stock/compare", server.compareStocks)
		v1.GET("/stock/:code/predict", server.getStockPrediction)
		v1.GET("/stock/:code/risk", server.getRiskMetrics)
		v1.GET("/stock/:code/backtest", server.getBacktest)

		// 数据获取
		v1.GET("/data/:code", server.getStockData)
		v1.POST("/data/upload", server.uploadCSV)

		// 技术指标
		v1.GET("/indicators/:code", server.getTechnicalIndicators)

		// 机器学习
		v1.POST("/ml/train", server.trainModel)
		v1.GET("/ml/models", server.listModels)

		// WebSocket实时数据
		v1.GET("/ws/stock/:code", server.handleWebSocket)
	}
}

// Start 启动服务器
func (server *Server) Start() error {
	addr := ":" + server.config.Server.Port
	logger.Infof("启动API服务器，监听端口: %s", server.config.Server.Port)
	return server.router.Run(addr)
}

// healthCheck 健康检查
func (server *Server) healthCheck(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status":    "ok",
		"timestamp": time.Now().Unix(),
		"version":   "1.0.0",
	})
}

// getStockAnalysis 获取股票分析
func (server *Server) getStockAnalysis(c *gin.Context) {
	code := c.Param("code")
	start := c.DefaultQuery("start", "2023-01-01")
	end := c.DefaultQuery("end", time.Now().Format("2006-01-02"))

	// 获取K线数据
	var klines []data.Kline
	var err error

	if len(code) > 3 && (code[len(code)-3:] == ".SZ" || code[len(code)-3:] == ".SH") {
		klines, err = data.FetchTencentKlines(code, start, end)
	} else {
		klines, err = data.FetchYahooKlines(code, start, end)
	}

	if err != nil {
		logger.Errorf("获取股票数据失败: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "数据获取失败"})
		return
	}

	if len(klines) < 30 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "数据不足30天"})
		return
	}

	// 计算技术指标
	factors := analysis.CalcFactors(klines)
	for i := range factors {
		factors[i].Code = code
	}

	// 计算风险指标
	riskMetrics := analysis.CalculateRiskMetrics(klines, 0.03)

	// 机器学习预测
	var mlPredictions map[string]analysis.MLPrediction
	if len(factors) > 0 {
		predictor := analysis.NewMLPredictor(klines, factors)
		mlPredictions = predictor.PredictAll()
	}

	// 趋势预测
	var prediction analysis.Prediction
	if len(klines) >= 30 {
		prediction = analysis.PredictTrend(klines)
	}

	response := gin.H{
		"code":           code,
		"period":         gin.H{"start": start, "end": end},
		"data_count":     len(klines),
		"factors":        factors[len(factors)-1], // 最新指标
		"risk_metrics":   riskMetrics,
		"ml_predictions": mlPredictions,
		"prediction":     prediction,
	}

	c.JSON(http.StatusOK, response)
}

// compareStocks 股票对比
func (server *Server) compareStocks(c *gin.Context) {
	var request struct {
		Codes   []string  `json:"codes" binding:"required"`
		Start   string    `json:"start"`
		End     string    `json:"end"`
		Factors []string  `json:"factors"`
		Weights []float64 `json:"weights"`
	}

	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请求参数错误"})
		return
	}

	if request.Start == "" {
		request.Start = "2023-01-01"
	}
	if request.End == "" {
		request.End = time.Now().Format("2006-01-02")
	}

	// 执行股票对比分析
	results := analysis.CompareStocks(request.Codes, request.Start, request.End, request.Factors, request.Weights)

	c.JSON(http.StatusOK, gin.H{
		"stocks":  request.Codes,
		"period":  gin.H{"start": request.Start, "end": request.End},
		"results": results,
	})
}

// getStockPrediction 获取股票预测
func (server *Server) getStockPrediction(c *gin.Context) {
	code := c.Param("code")
	start := c.DefaultQuery("start", "2023-01-01")
	end := c.DefaultQuery("end", time.Now().Format("2006-01-02"))

	// 获取数据
	var klines []data.Kline
	var err error

	if len(code) > 3 && (code[len(code)-3:] == ".SZ" || code[len(code)-3:] == ".SH") {
		klines, err = data.FetchTencentKlines(code, start, end)
	} else {
		klines, err = data.FetchYahooKlines(code, start, end)
	}

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "数据获取失败"})
		return
	}

	factors := analysis.CalcFactors(klines)
	if len(factors) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无法计算技术指标"})
		return
	}

	// 机器学习预测
	predictor := analysis.NewMLPredictor(klines, factors)
	predictions := predictor.PredictAll()

	// 传统预测
	var trendPrediction analysis.Prediction
	if len(klines) >= 30 {
		trendPrediction = analysis.PredictTrend(klines)
	}

	c.JSON(http.StatusOK, gin.H{
		"code":             code,
		"ml_predictions":   predictions,
		"trend_prediction": trendPrediction,
	})
}

// getRiskMetrics 获取风险指标
func (server *Server) getRiskMetrics(c *gin.Context) {
	code := c.Param("code")
	start := c.DefaultQuery("start", "2023-01-01")
	end := c.DefaultQuery("end", time.Now().Format("2006-01-02"))
	riskFreeRate, _ := strconv.ParseFloat(c.DefaultQuery("risk_free_rate", "0.03"), 64)

	// 获取数据
	var klines []data.Kline
	var err error

	if len(code) > 3 && (code[len(code)-3:] == ".SZ" || code[len(code)-3:] == ".SH") {
		klines, err = data.FetchTencentKlines(code, start, end)
	} else {
		klines, err = data.FetchYahooKlines(code, start, end)
	}

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "数据获取失败"})
		return
	}

	riskMetrics := analysis.CalculateRiskMetrics(klines, riskFreeRate)

	c.JSON(http.StatusOK, gin.H{
		"code":         code,
		"risk_metrics": riskMetrics,
	})
}

// getBacktest 获取回测结果
func (server *Server) getBacktest(c *gin.Context) {
	code := c.Param("code")
	start := c.DefaultQuery("start", "2023-01-01")
	end := c.DefaultQuery("end", time.Now().Format("2006-01-02"))

	// 获取数据
	var klines []data.Kline
	var err error

	if len(code) > 3 && (code[len(code)-3:] == ".SZ" || code[len(code)-3:] == ".SH") {
		klines, err = data.FetchTencentKlines(code, start, end)
	} else {
		klines, err = data.FetchYahooKlines(code, start, end)
	}

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "数据获取失败"})
		return
	}

	// 执行多策略回测
	analysis.RunMultiStrategyBacktest(klines)

	c.JSON(http.StatusOK, gin.H{
		"code":    code,
		"results": "回测结果已输出到控制台",
	})
}

// getStockData 获取股票数据
func (server *Server) getStockData(c *gin.Context) {
	code := c.Param("code")
	start := c.DefaultQuery("start", "2023-01-01")
	end := c.DefaultQuery("end", time.Now().Format("2006-01-02"))

	var klines []data.Kline
	var err error

	if len(code) > 3 && (code[len(code)-3:] == ".SZ" || code[len(code)-3:] == ".SH") {
		klines, err = data.FetchTencentKlines(code, start, end)
	} else {
		klines, err = data.FetchYahooKlines(code, start, end)
	}

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "数据获取失败"})
		return
	}

	// 转换为JSON格式
	var data []gin.H
	for _, kline := range klines {
		data = append(data, gin.H{
			"date":   kline.Date.Format("2006-01-02"),
			"open":   kline.Open,
			"high":   kline.High,
			"low":    kline.Low,
			"close":  kline.Close,
			"volume": kline.Volume,
		})
	}

	c.JSON(http.StatusOK, gin.H{
		"code": code,
		"data": data,
	})
}

// uploadCSV 上传CSV文件
func (server *Server) uploadCSV(c *gin.Context) {
	file, err := c.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "文件上传失败"})
		return
	}

	// 保存文件
	filename := fmt.Sprintf("uploads/%s", file.Filename)
	if err := c.SaveUploadedFile(file, filename); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "文件保存失败"})
		return
	}

	// 读取CSV数据
	klines, err := data.FetchCSVKlines(filename)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "CSV解析失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":    "文件上传成功",
		"data_count": len(klines),
		"filename":   filename,
	})
}

// getTechnicalIndicators 获取技术指标
func (server *Server) getTechnicalIndicators(c *gin.Context) {
	code := c.Param("code")
	start := c.DefaultQuery("start", "2023-01-01")
	end := c.DefaultQuery("end", time.Now().Format("2006-01-02"))

	// 获取数据
	var klines []data.Kline
	var err error

	if len(code) > 3 && (code[len(code)-3:] == ".SZ" || code[len(code)-3:] == ".SH") {
		klines, err = data.FetchTencentKlines(code, start, end)
	} else {
		klines, err = data.FetchYahooKlines(code, start, end)
	}

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "数据获取失败"})
		return
	}

	factors := analysis.CalcFactors(klines)
	for i := range factors {
		factors[i].Code = code
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    code,
		"factors": factors,
	})
}

// trainModel 训练模型
func (server *Server) trainModel(c *gin.Context) {
	var request struct {
		Code     string   `json:"code" binding:"required"`
		Start    string   `json:"start"`
		End      string   `json:"end"`
		Features []string `json:"features"`
	}

	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请求参数错误"})
		return
	}

	// 这里可以添加模型训练逻辑
	c.JSON(http.StatusOK, gin.H{
		"message": "模型训练功能开发中",
		"code":    request.Code,
	})
}

// listModels 列出模型
func (server *Server) listModels(c *gin.Context) {
	// 这里可以返回已训练的模型列表
	c.JSON(http.StatusOK, gin.H{
		"models":  []string{},
		"message": "模型管理功能开发中",
	})
}

// handleWebSocket WebSocket处理
func (server *Server) handleWebSocket(c *gin.Context) {
	code := c.Param("code")

	conn, err := server.upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		logger.Errorf("WebSocket升级失败: %v", err)
		return
	}
	defer conn.Close()

	logger.Infof("WebSocket连接建立: %s", code)

	// 发送实时数据
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			// 获取最新数据
			end := time.Now().Format("2006-01-02")
			start := time.Now().AddDate(0, 0, -7).Format("2006-01-02")

			var klines []data.Kline
			var err error

			if len(code) > 3 && (code[len(code)-3:] == ".SZ" || code[len(code)-3:] == ".SH") {
				klines, err = data.FetchTencentKlines(code, start, end)
			} else {
				klines, err = data.FetchYahooKlines(code, start, end)
			}

			if err != nil {
				logger.Errorf("获取实时数据失败: %v", err)
				continue
			}

			if len(klines) > 0 {
				latest := klines[len(klines)-1]
				data := gin.H{
					"code":   code,
					"time":   time.Now().Unix(),
					"price":  latest.Close,
					"volume": latest.Volume,
				}

				if err := conn.WriteJSON(data); err != nil {
					logger.Errorf("发送WebSocket数据失败: %v", err)
					return
				}
			}
		}
	}
}

// corsMiddleware CORS中间件
func corsMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Header("Access-Control-Allow-Origin", "*")
		c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Origin, Content-Type, Authorization")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}

		c.Next()
	}
}

// loggingMiddleware 日志中间件
func loggingMiddleware() gin.HandlerFunc {
	return gin.LoggerWithFormatter(func(param gin.LogFormatterParams) string {
		logger.Infof("API请求: %s %s %d %v",
			param.Method,
			param.Path,
			param.StatusCode,
			param.Latency,
		)
		return ""
	})
}
