package data

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"
)

type Kline struct {
	Date   time.Time
	Open   float64
	High   float64
	Low    float64
	Close  float64
	Volume int64
}

// Yahoo Finance 获取美股/部分A股
func FetchYahooKlines(symbol, start, end string) ([]Kline, error) {
	defaultLayout := "2006-01-02"
	startTime, _ := time.Parse(defaultLayout, start)
	endTime, _ := time.Parse(defaultLayout, end)
	url := fmt.Sprintf("https://query1.finance.yahoo.com/v7/finance/download/%s?period1=%d&period2=%d&interval=1d&events=history", symbol, startTime.Unix(), endTime.Unix())
	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("http status: %d", resp.StatusCode)
	}
	reader := csv.NewReader(resp.Body)
	records, err := reader.ReadAll()
	if err != nil {
		return nil, err
	}
	var klines []Kline
	for i, rec := range records {
		if i == 0 { // 跳过表头
			continue
		}
		if len(rec) < 6 {
			continue
		}
		date, _ := time.Parse(defaultLayout, rec[0])
		open, _ := strconv.ParseFloat(rec[1], 64)
		high, _ := strconv.ParseFloat(rec[2], 64)
		low, _ := strconv.ParseFloat(rec[3], 64)
		closep, _ := strconv.ParseFloat(rec[4], 64)
		volume, _ := strconv.ParseInt(rec[6], 10, 64)
		klines = append(klines, Kline{
			Date:   date,
			Open:   open,
			High:   high,
			Low:    low,
			Close:  closep,
			Volume: volume,
		})
	}
	return klines, nil
}

// 腾讯财经A股历史K线
type TencentResponse struct {
	Data map[string]struct {
		Qfqday [][]string `json:"qfqday"`
	} `json:"data"`
}

// 转换A股代码格式：000001.SZ -> sz000001, 600519.SH -> sh600519
func convertStockCode(code string) string {
	if strings.HasSuffix(code, ".SZ") {
		return "sz" + strings.TrimSuffix(code, ".SZ")
	} else if strings.HasSuffix(code, ".SH") {
		return "sh" + strings.TrimSuffix(code, ".SH")
	}
	return code
}

func FetchTencentKlines(code, start, end string) ([]Kline, error) {
	tencentCode := convertStockCode(code)
	url := fmt.Sprintf("https://web.ifzq.gtimg.cn/appstock/app/fqkline/get?param=%s,day,%s,%s,320,qfq", tencentCode, start, end)
	fmt.Printf("请求URL: %s\n", url)
	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("http status: %d", resp.StatusCode)
	}
	data, _ := ioutil.ReadAll(resp.Body)
	if len(data) > 500 {
		fmt.Printf("返回数据前500字符: %s\n", string(data[:500]))
	} else {
		fmt.Printf("返回数据: %s\n", string(data))
	}
	var tr TencentResponse
	err = json.Unmarshal(data, &tr)
	if err != nil {
		return nil, fmt.Errorf("JSON解析失败: %v", err)
	}
	var klines []Kline
	if stockData, exists := tr.Data[tencentCode]; exists {
		for _, item := range stockData.Qfqday {
			if len(item) < 6 {
				continue
			}
			date, _ := time.Parse("2006-01-02", item[0])
			open, _ := strconv.ParseFloat(item[1], 64)
			close, _ := strconv.ParseFloat(item[2], 64)
			high, _ := strconv.ParseFloat(item[3], 64)
			low, _ := strconv.ParseFloat(item[4], 64)
			volume, _ := strconv.ParseInt(item[5], 10, 64)
			klines = append(klines, Kline{
				Date:   date,
				Open:   open,
				High:   high,
				Low:    low,
				Close:  close,
				Volume: volume,
			})
		}
	} else {
		for k := range tr.Data {
			fmt.Printf("%s ", k)
		}
		fmt.Println()
	}
	return klines, nil
}

// 从本地CSV文件读取K线数据，兼容Yahoo等标准格式
func FetchCSVKlines(filename string) ([]Kline, error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	reader := csv.NewReader(file)
	records, err := reader.ReadAll()
	if err != nil {
		return nil, err
	}
	var klines []Kline
	for i, rec := range records {
		if i == 0 { // 跳过表头
			continue
		}
		if len(rec) < 6 {
			continue
		}
		date, _ := time.Parse("2006-01-02", rec[0])
		open, _ := strconv.ParseFloat(rec[1], 64)
		high, _ := strconv.ParseFloat(rec[2], 64)
		low, _ := strconv.ParseFloat(rec[3], 64)
		closep, _ := strconv.ParseFloat(rec[4], 64)
		volume, _ := strconv.ParseInt(rec[6], 10, 64)
		klines = append(klines, Kline{
			Date:   date,
			Open:   open,
			High:   high,
			Low:    low,
			Close:  closep,
			Volume: volume,
		})
	}
	return klines, nil
}
