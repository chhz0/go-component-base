// vconfig
// 对 spf13/viper 二次封装,简化使用，一次性完成配置
// 配置优先级：vconfig 与 viper 保证一致
// set > flag > env(.env) > config > key/value > default
// 支持：
// - 设置默认值
// - 读取配置文件
// - 支持key/value存储
// - 热更新配置
// - flag
// - env
package vconfig

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"strings"
	"sync"
	"time"

	"github.com/BurntSushi/toml"
	"github.com/fsnotify/fsnotify"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
	"gopkg.in/yaml.v2"
)

var (
	ErrConfigNotFound = errors.New("config file not found")
	ErrReaderIO       = errors.New("reader io error")
	ErrInvalidType    = errors.New("invalid config type")
	ErrRemoteConfig   = errors.New("remote config error")
	ErrUnmarshal      = errors.New("unmarshal error")
	ErrUnmarshalNil   = errors.New("unmarshal nil")
)

type RemoteProvider struct {
	Provider string
	Endpoint string
	Path     string
	Type     string
}

type Env struct {
	Binds       []string // 环境变量命
	Prefix      string   // 环境变量前缀
	KeyReplacer *strings.Replacer
	// TODO: allow empty env
}

// TODO: 多配置文件来源
type Local struct {
	ConfigName  string    // 配置文件名
	ConfigType  string    // 配置文件类型
	ConfigPaths []string  // 配置文件路径
	ConfigIO    io.Reader // 配置读取 IO
}

type Options struct {
	Sets     map[string]any
	Defaults map[string]any

	Local *Local

	// 在 dotenv 中支持环境变量读取
	DotEnv *Local
	Env    *Env

	Flags []*pflag.FlagSet // flags

	// UnmarshalPtr 反序列化对象, 必须是 指针
	// 如果提供了 UnmarshalPtr 且开启了Watcher，在配置文件更新时自动反序列化
	UnmarshalPtr any

	RemoteS             struct{}
	Remote              *RemoteProvider
	RemoteWatch         bool
	RemoteWatchInterval time.Duration

	EnableEnv    bool // 是否开启环境变量
	EnableFlag   bool // 是否使用flag
	EnableRemote bool // 是否开启远程配置中心
}

type VConfig struct {
	// TODO:  viper or vipers
	v    *viper.Viper
	opts *Options
	mu   sync.RWMutex
}

// New 使用 options 模式创建配置实例
func NewWith(optFuncs ...func(*Options)) *VConfig {
	defaultOpts := &Options{
		Local: &Local{
			ConfigName:  "",
			ConfigPaths: []string{"."},
		},
		Env: &Env{
			KeyReplacer: defaultKeyReplacer(),
		},
		EnableEnv:           true,
		RemoteWatchInterval: 30 * time.Second,
	}
	for _, fn := range optFuncs {
		fn(defaultOpts)
	}

	vc := &VConfig{
		v:    viper.New(),
		opts: defaultOpts,
	}

	vc.initialize()

	return vc
}

// NewInOptions 使用Options创建配置实例
// 预期：opts 必须全部配置
func New(opts *Options) *VConfig {
	vc := &VConfig{
		v:    viper.New(),
		opts: opts,
	}

	vc.initialize()

	return vc
}

func (vc *VConfig) initialize() {
	vc.setDefault()

	// 加载 flag 参数
	if vc.opts.EnableFlag {
		vc.bindFlags()
	}

	// 加载环境变量
	if vc.opts.EnableEnv {
		vc.setupEnv()
	}

	vc.setInRead("local")
	// 加载本地配置文件
	if err := vc.loadLocal(); err != nil && !errors.Is(err, ErrConfigNotFound) {
		log.Printf("Warning: Error loading config file: %v", err)
	}

	if vc.opts.DotEnv != nil {
		vc.setInRead("dotenv")
		if err := vc.loadLocal(); err != nil && !errors.Is(err, ErrConfigNotFound) {
			log.Printf("Warning: Error loading config file: %v", err)
		}
	}

	// 加载远程配置文件
	if vc.opts.EnableRemote {
		if err := vc.loadRemote(); err != nil {
			log.Printf("Warning: Error loading remote config: %v", err)
		}
	}

	// 加载 key/value 参数
	for key, val := range vc.opts.Sets {
		vc.v.Set(key, val)
	}
}

func (vc *VConfig) setupEnv() {
	vc.v.AutomaticEnv()
	if vc.opts.Env.Prefix != "" {
		vc.v.SetEnvPrefix(vc.opts.Env.Prefix)
	}
	if vc.opts.Env.Binds != nil {
		for _, env := range vc.opts.Env.Binds {
			_ = vc.v.BindEnv(env)
		}
	}
	if vc.opts.Env.KeyReplacer != nil {
		vc.v.SetEnvKeyReplacer(vc.opts.Env.KeyReplacer)
	}
}

func (vc *VConfig) bindFlags() {
	for _, fs := range vc.opts.Flags {
		fs.VisitAll(func(f *pflag.Flag) {
			if err := vc.v.BindPFlag(f.Name, f); err != nil {
				log.Printf("failed to bind flag %s: %v", f.Name, err)
			}
		})
	}
}

func (vc *VConfig) loadLocal() error {
	if err := vc.v.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); ok && vc.opts.Local.ConfigIO != nil {
			return vc.loadReaderIO()
		}
		return fmt.Errorf("config file read error: %v\n", err)
	}

	return nil
}

func (vc *VConfig) setInRead(in string) {
	use := vc.opts.Local
	if in == "dotenv" {
		use = vc.opts.DotEnv
	}

	vc.v.SetConfigName(use.ConfigName)
	vc.v.SetConfigType(use.ConfigType)
	for _, cp := range use.ConfigPaths {
		vc.v.AddConfigPath(cp)
	}
}

func (vc *VConfig) loadReaderIO() error {
	if err := vc.v.ReadConfig(vc.opts.Local.ConfigIO); err != nil {
		return ErrReaderIO
	}

	return nil
}

func (vc *VConfig) loadRemote() error {
	if vc.opts.Remote == nil || vc.opts.EnableRemote {
		return nil
	}

	remote := vc.opts.Remote
	if err := vc.v.AddRemoteProvider(remote.Provider, remote.Endpoint, remote.Path); err != nil {
		log.Printf("failed to remote provider: %v\n", err)
		return ErrRemoteConfig
	}

	vc.v.SetConfigType(remote.Type)
	if err := vc.v.ReadRemoteConfig(); err != nil {
		return ErrRemoteConfig
	}

	return nil
}

// Watcher 监听配置文件变化, changedFunc 将在配置文件更新并重新加载完成后调用
func (vc *VConfig) Watcher(changedFunc func()) {
	vc.enableWatch(changedFunc)
}

func (vc *VConfig) enableWatch(fn func()) {
	vc.v.OnConfigChange(func(in fsnotify.Event) {
		log.Printf("config file changed: %v\n", in.Name)
		if err := vc.v.ReadInConfig(); err != nil {
			log.Printf("reload config file error: %v\n", err)
		}
		_ = vc.unmarshal()
		fn()
	})
	vc.v.WatchConfig()

	if vc.opts.RemoteWatch {
		go vc.watchRemote(context.Background())
	}
}

func (vc *VConfig) watchRemote(ctx context.Context) {
	ticker := time.NewTicker(vc.opts.RemoteWatchInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if err := vc.v.WatchRemoteConfig(); err != nil {
				log.Printf("reload remote config error: %v\n", err)
			}
		}
	}
}

func (vc *VConfig) Unmarshal(ptr any) error {
	if err := vc.v.Unmarshal(ptr); err != nil {
		return ErrUnmarshal
	}

	return nil
}

func (vc *VConfig) unmarshal() error {
	if vc.opts.UnmarshalPtr == nil {
		return ErrUnmarshalNil
	}
	if err := vc.v.Unmarshal(vc.opts.UnmarshalPtr); err != nil {
		return ErrUnmarshal
	}

	return nil
}

// Marshal 将vc.v.AllSettings()序列化为字符串
// 目前支持：json, yaml, toml
func (vc *VConfig) MarshalToString(marshalType string) string {
	m := vc.v.AllSettings()
	var buf []byte
	var err error
	switch marshalType {
	case "json":
		buf, err = json.Marshal(m)
		if err != nil {
			panic(err)
		}
	case "yaml":
		buf, err = yaml.Marshal(m)
		if err != nil {
			panic(err)
		}
	case "toml":
		buf, err = toml.Marshal(m)
		if err != nil {
			panic(err)
		}
	}
	return string(buf)
}
func (vc *VConfig) setDefault() {
	for k, v := range vc.opts.Defaults {
		vc.v.SetDefault(k, v)
	}
}

func (vc *VConfig) BindPFlag(mFlag map[string]*pflag.Flag) {
	for key, flag := range mFlag {
		_ = vc.v.BindPFlag(key, flag)
	}
}

func (vc *VConfig) BindPFlags(pfs ...*pflag.FlagSet) {
	for _, pf := range pfs {
		_ = vc.v.BindPFlags(pf)
	}
}

// BindEnvs 绑定环境变量，不同于viper.BindEnv限制一个传入的参数
// 如果想使用viper.BindEnv，请调用函数 V() 获取 *viper.Viper实例
func (vc *VConfig) BindEnvs(input string) {
	_ = vc.v.BindEnv(input)
}

func (vc *VConfig) GetEnv(key string) string {
	return vc.v.GetString(key)
}

func (vc *VConfig) Set(key string, value any) {
	vc.mu.Lock()
	defer vc.mu.Unlock()
	vc.v.Set(key, value)
}

// Get 允许访问给定key 的value
// 如果有嵌套的key，则使用点号分隔符访问："section.key"
func (vc *VConfig) Get(key string) (any, bool) {
	if !vc.v.IsSet(key) {
		return nil, false
	}

	v := vc.v.Get(key)
	return v, true
}

func (vc *VConfig) AllSettings() map[string]any {
	return vc.v.AllSettings()
}

// V returns the viper instance
func (vc *VConfig) V() *viper.Viper {
	return vc.v
}
func WithSets(sets map[string]any) func(*Options) {
	return func(o *Options) {
		o.Sets = sets
	}
}
func WithDefaults(defaluts map[string]any) func(*Options) {
	return func(o *Options) {
		o.Defaults = defaluts
	}
}

func WithDotEnv(mode string, path ...string) func(*Options) {
	return func(o *Options) {
		o.DotEnv = &Local{
			ConfigName:  mode,
			ConfigType:  "env",
			ConfigPaths: path,
		}
	}
}

func WithLocal(local *Local) func(*Options) {
	return func(o *Options) {
		o.Local = local
	}
}

func WithConfigName(name string) func(*Options) {
	return func(o *Options) {
		o.Local.ConfigName = name
	}
}

func WithConfigType(configType string) func(*Options) {
	return func(o *Options) {
		o.Local.ConfigType = configType
	}
}

func WithConfigPaths(paths ...string) func(*Options) {
	return func(o *Options) {
		o.Local.ConfigPaths = append(o.Local.ConfigPaths, paths...)
	}
}

func WithUnmarshal(ptr any) func(*Options) {
	return func(o *Options) {
		o.UnmarshalPtr = ptr
	}
}

// WithEnv 允许设置环境变量, 如果使用 WithEnv ， 必须传入的 Env.KeyReplacer
func WithEnv(env *Env) func(*Options) {
	return func(o *Options) {
		if env.KeyReplacer == nil {
			env.KeyReplacer = defaultKeyReplacer()
		}
		o.Env = env
	}
}

func WithEnvBinds(binds ...string) func(*Options) {
	return func(o *Options) {
		o.Env.Binds = append(o.Env.Binds, binds...)
	}
}

func WithEnvPrefix(prefix string) func(*Options) {
	return func(o *Options) {
		o.Env.Prefix = prefix
	}
}

func WithEnvKeyReplacer(replacer *strings.Replacer) func(*Options) {
	return func(o *Options) {
		o.Env.KeyReplacer = replacer
	}
}

func WithRemote(remote *RemoteProvider) func(*Options) {
	return func(o *Options) {
		o.Remote = remote
	}
}

func EnableEnv(enable bool) func(*Options) {
	return func(o *Options) {
		o.EnableEnv = enable
	}
}

func EnableFlag(flags ...*pflag.FlagSet) func(*Options) {
	return func(o *Options) {
		o.EnableFlag = true
		o.Flags = append(o.Flags, flags...)
	}
}

func EnableRemote(enable bool) func(*Options) {
	return func(o *Options) {
		o.EnableRemote = enable
	}
}

func EnableRemoteWatch(enable bool) func(*Options) {
	return func(o *Options) {
		o.RemoteWatch = enable
	}
}

func defaultKeyReplacer() *strings.Replacer {
	return strings.NewReplacer(".", "_", "-", "_")
}
