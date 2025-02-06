package xhttp

import (
	"net"
	"time"
)

type Config struct {
	Http HttpInfo
	TLS  TLSInfo
	JWT  JWTInfo

	ShutdownTimeout time.Duration
	Middlewares     []string

	HealthCheck     bool
	EnableProfiling bool
	EnableMetrics   bool
}

type HttpInfo struct {
	BindAddress string
	BindPort    string
}

func (h *HttpInfo) Address() string {
	return net.JoinHostPort(h.BindAddress, h.BindPort)
}

type TLSInfo struct {
	BindAddress string
	BindPort    string
	CertKey     CertKey
}

func (t *TLSInfo) Address() string {
	return net.JoinHostPort(t.BindAddress, t.BindPort)
}

type CertKey struct {
	Cert string
	Key  string
}

type JWTInfo struct {
	Realm      string
	Key        string
	Timeout    time.Duration
	MaxRefresh time.Duration
}

func NewZeroConfig() *Config {
	return &Config{
		Http: HttpInfo{
			BindAddress: "127.0.0.1",
			BindPort:    "80",
		},
		TLS: TLSInfo{
			BindAddress: "127.0.0.1",
			BindPort:    "443",
			CertKey: CertKey{
				Cert: "",
				Key:  "",
			},
		},
		JWT: JWTInfo{
			Realm:      "",
			Key:        "",
			Timeout:    time.Hour * 24 * 7,
			MaxRefresh: time.Hour * 24 * 7,
		},
		ShutdownTimeout: time.Second * 5,
		Middlewares:     []string{},
		HealthCheck:     false,
		EnableProfiling: false,
		EnableMetrics:   false,
	}
}
