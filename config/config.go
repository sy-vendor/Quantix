package config

import (
	"fmt"
	"log"
	"os"
	"strconv"
	"time"

	"github.com/spf13/viper"
)

var GlobalConfig *Config

// Config 应用配置结构
type Config struct {
	Server   ServerConfig   `mapstructure:"server"`
	Database DatabaseConfig `mapstructure:"database"`
	Redis    RedisConfig    `mapstructure:"redis"`
	Data     DataConfig     `mapstructure:"data"`
	ML       MLConfig       `mapstructure:"ml"`
	Log      LogConfig      `mapstructure:"log"`
	DeepSeek DeepSeekConfig `mapstructure:"deepseek"`
}

// ServerConfig 服务器配置
type ServerConfig struct {
	Port         string        `mapstructure:"port"`
	ReadTimeout  time.Duration `mapstructure:"read_timeout"`
	WriteTimeout time.Duration `mapstructure:"write_timeout"`
	IdleTimeout  time.Duration `mapstructure:"idle_timeout"`
}

// DatabaseConfig 数据库配置
type DatabaseConfig struct {
	Host     string `mapstructure:"host"`
	Port     int    `mapstructure:"port"`
	User     string `mapstructure:"user"`
	Password string `mapstructure:"password"`
	DBName   string `mapstructure:"dbname"`
	SSLMode  string `mapstructure:"sslmode"`
}

// RedisConfig Redis配置
type RedisConfig struct {
	Host     string `mapstructure:"host"`
	Port     int    `mapstructure:"port"`
	Password string `mapstructure:"password"`
	DB       int    `mapstructure:"db"`
}

// DataConfig 数据源配置
type DataConfig struct {
	YahooAPIKey     string `mapstructure:"yahoo_api_key"`
	TencentAPIKey   string `mapstructure:"tencent_api_key"`
	CacheExpiration int    `mapstructure:"cache_expiration"`
	MaxRetries      int    `mapstructure:"max_retries"`
}

// MLConfig 机器学习配置
type MLConfig struct {
	ModelPath           string  `mapstructure:"model_path"`
	ConfidenceThreshold float64 `mapstructure:"confidence_threshold"`
	MaxFeatures         int     `mapstructure:"max_features"`
	CrossValidation     bool    `mapstructure:"cross_validation"`
}

// LogConfig 日志配置
type LogConfig struct {
	Level  string `mapstructure:"level"`
	Format string `mapstructure:"format"`
	Output string `mapstructure:"output"`
}

// DeepSeekConfig 深度搜索配置
type DeepSeekConfig struct {
	APIURL string `mapstructure:"api_url"`
}

// LoadConfig 加载配置
func LoadConfig(configPath string) (*Config, error) {
	viper.SetConfigName("config")
	viper.SetConfigType("yaml")
	viper.AddConfigPath(".")
	viper.AddConfigPath("./config")

	// 设置环境变量前缀
	viper.SetEnvPrefix("QUANTIX")
	viper.AutomaticEnv()

	// 设置默认值
	setDefaults()

	// 读取配置文件
	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return nil, fmt.Errorf("读取配置文件失败: %w", err)
		}
		log.Println("未找到配置文件，使用默认配置")
	}

	var config Config
	if err := viper.Unmarshal(&config); err != nil {
		return nil, fmt.Errorf("解析配置失败: %w", err)
	}

	// 验证配置
	if err := validateConfig(&config); err != nil {
		return nil, fmt.Errorf("配置验证失败: %w", err)
	}

	return &config, nil
}

// setDefaults 设置默认配置
func setDefaults() {
	// 服务器默认配置
	viper.SetDefault("server.port", "8080")
	viper.SetDefault("server.read_timeout", "30s")
	viper.SetDefault("server.write_timeout", "30s")
	viper.SetDefault("server.idle_timeout", "60s")

	// 数据库默认配置
	viper.SetDefault("database.host", "localhost")
	viper.SetDefault("database.port", 5432)
	viper.SetDefault("database.sslmode", "disable")

	// Redis默认配置
	viper.SetDefault("redis.host", "localhost")
	viper.SetDefault("redis.port", 6379)
	viper.SetDefault("redis.db", 0)

	// 数据源默认配置
	viper.SetDefault("data.cache_expiration", 3600)
	viper.SetDefault("data.max_retries", 3)

	// ML默认配置
	viper.SetDefault("ml.confidence_threshold", 0.7)
	viper.SetDefault("ml.max_features", 20)
	viper.SetDefault("ml.cross_validation", true)

	// 日志默认配置
	viper.SetDefault("log.level", "info")
	viper.SetDefault("log.format", "json")
	viper.SetDefault("log.output", "stdout")
}

// validateConfig 验证配置
func validateConfig(config *Config) error {
	// 验证端口号
	if port, err := strconv.Atoi(config.Server.Port); err != nil || port <= 0 || port > 65535 {
		return fmt.Errorf("无效的端口号: %s", config.Server.Port)
	}

	// 验证数据库配置
	if config.Database.Host == "" {
		return fmt.Errorf("数据库主机不能为空")
	}

	// 验证Redis配置
	if config.Redis.Host == "" {
		return fmt.Errorf("Redis主机不能为空")
	}

	return nil
}

// GetEnv 获取环境变量，支持默认值
func GetEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// GetEnvInt 获取整数环境变量
func GetEnvInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if intValue, err := strconv.Atoi(value); err == nil {
			return intValue
		}
	}
	return defaultValue
}

// GetEnvBool 获取布尔环境变量
func GetEnvBool(key string, defaultValue bool) bool {
	if value := os.Getenv(key); value != "" {
		if boolValue, err := strconv.ParseBool(value); err == nil {
			return boolValue
		}
	}
	return defaultValue
}
