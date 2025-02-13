package config

import (
	"log"
	"path/filepath"
	"strings"

	"github.com/fsnotify/fsnotify"
	"github.com/spf13/viper"
)

const (
	defaultModeEnv = "dev"

	defaultEnvFileName = "dev"
	defaultEnvFileType = "env"
	defaultEnvFilePath = "."

	defaultConfigFileName = "config"
	defaultConfigFileType = "yaml"
	defaultConfigFilePath = "./config"
)

type Config struct {
	v       *viper.Viper
	vdotenv *viper.Viper

	ModeEnv         string   // 运行环境
	EnvPrefix       string   // 环境变量前缀
	EnvFileName     string   // .env 配置文件名
	EnvFilePath     string   // .env 文件路径
	BindEnv         []string // 绑定环境变量
	ConfigFileName  string   // 配置文件名
	ConfigFileType  string   // 配置文件类型
	ConfigFilePath  string   // 配置文件路径
	Watcher         bool     // 是否开启热更新
	UnmarshalStruct any      // 配置文件结构体，反序列化对象
}

type Option func(*Config)

// WithModeEnv 设置运行环境 dev test prod
func WithModeEnv(modeEnv string) Option {
	return func(c *Config) {
		c.ModeEnv = modeEnv
	}
}

// WithEnvPrefix 设置环境变量前缀
func WithEnvPrefix(envPrefix string) Option {
	return func(c *Config) {
		c.EnvPrefix = envPrefix
	}
}

// WithEnvFileName 设置.env 文件名
func WithEnvFileName(envFileName string) Option {
	return func(c *Config) {
		c.EnvFileName = envFileName
	}
}

// WithEnvFilePath 设置.env 文件路径
func WithEnvFilePath(envFilePath string) Option {
	return func(c *Config) {
		c.EnvFilePath = envFilePath
	}
}

// WithBindEnv 设置绑定环境变量
func WithBindEnv(env ...string) Option {
	return func(c *Config) {
		c.BindEnv = env
	}
}

// WithWatcher 设置是否开启热更新
func WithWatcher(watcher bool) Option {
	return func(c *Config) {
		c.Watcher = watcher
	}
}

// WithConfigFile 设置配置文件名
func WithConfigFile(fileName string) Option {
	return func(c *Config) {
		c.ConfigFileName = fileName
	}
}

// WithConfigFileType 设置配置文件类型
func WithConfigFileType(fileType string) Option {
	return func(c *Config) {
		c.ConfigFileType = fileType
	}
}

// WithConfigFilePath 设置配置文件路径
func WithConfigFilePath(filePath string) Option {
	return func(c *Config) {
		c.ConfigFilePath = filePath
	}
}

// WithUnmarshalStruct 设置反序列化对象
func WithUnmarshalStruct(unmarshalStruct any) Option {
	return func(c *Config) {
		c.UnmarshalStruct = unmarshalStruct
	}
}

// New 创建配置示例
func New(opts ...Option) *Config {
	conf := &Config{
		v:               viper.New(),
		vdotenv:         nil,
		ModeEnv:         "MODE_ENV",
		Watcher:         false,
		EnvPrefix:       "",
		EnvFileName:     defaultEnvFileName,
		EnvFilePath:     defaultEnvFilePath,
		BindEnv:         []string{},
		ConfigFileName:  defaultConfigFileName,
		ConfigFileType:  defaultConfigFileType,
		ConfigFilePath:  defaultConfigFilePath,
		UnmarshalStruct: nil,
	}

	for _, o := range opts {
		o(conf)
	}

	return conf
}

func (c *Config) LoadConfig() {
	// 读取环境变量
	c.loadEnvConfig()

	// 读取配置文件
	c.readConfigFile()

	if c.UnmarshalStruct != nil {
		c.unmarshal()
	}

	if c.Watcher {
		c.v.OnConfigChange(func(in fsnotify.Event) {
			log.Printf("config file changed: %v\n", in.Name)
		})
		c.v.WatchConfig()
	}
}

// LoadDotEnv  从.env 文件加载环境变量
func (c *Config) LoadDotEnv() {
	c.vdotenv = viper.New()
	c.readDotEnv()
}

// LoadEnvConfig 设置环境变量
func (c *Config) loadEnvConfig() {
	if c.EnvPrefix != "" {
		c.v.SetEnvPrefix(c.EnvPrefix)
	}

	for _, b := range c.BindEnv {
		_ = c.v.BindEnv(b)
	}

	c.v.AutomaticEnv()
	c.v.SetEnvKeyReplacer(strings.NewReplacer("-", "_", ".", "_"))
}

func (c *Config) readDotEnv() {
	// c.v.SetFs(afero.NewMemMapFs())

	c.vdotenv.SetConfigName(c.EnvFileName)
	c.vdotenv.SetConfigType("env")
	c.vdotenv.AddConfigPath(c.EnvFilePath)
	// c.v.SetConfigFile(c.EnvFileName)

	c.readInConfig("dotenv")
}

func (c *Config) readConfigFile() {
	c.v.SetConfigName(c.ConfigFileName)
	c.v.SetConfigType(c.ConfigFileType)
	configPath := filepath.Join(c.ConfigFilePath, c.getModeEnv())
	c.v.AddConfigPath(configPath)

	c.readInConfig("config")
}

func (c *Config) readInConfig(vType string) {
	var v *viper.Viper
	if vType == "dotenv" {
		v = c.vdotenv
	} else {
		v = c.v
	}

	if err := v.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); ok {
			log.Panicf("config file not found; please check your config file. error: %v", err)
		} else {
			log.Panicf("config file read error: %v\n", err)
		}
	}
}

func (c *Config) unmarshal() {
	if err := c.v.Unmarshal(c.UnmarshalStruct); err != nil {
		log.Panicf("unmarshal config error: %v\n", err)
	}
}

func (c *Config) BindEnvs(input ...string) {
	_ = c.v.BindEnv(input...)
}

func (c *Config) GetEnv(key string) string {
	getkey := c.v.GetString(key)
	return getkey
}

func (c *Config) GetDotEnv(key string) string {
	if c.vdotenv == nil {
		return ""
	}
	return c.vdotenv.GetString(key)
}

func (c *Config) getModeEnv() string {
	env := c.v.GetString(c.ModeEnv)
	if len(env) == 0 {
		return defaultModeEnv
	}
	return env
}
