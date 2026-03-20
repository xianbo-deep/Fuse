package core

import (
	"time"
)

type Config struct {
	StopTimeout         time.Duration // 服务停机超时时间
	Port                string        // 服务监听的端口
	MuxHandShakeTimeout time.Duration // 协议分发器配置
	Mode                string        // 运行模式
}

func DefaultConfig() *Config {
	return &Config{
		StopTimeout:         5 * time.Second,
		Port:                "8080",
		MuxHandShakeTimeout: 5 * time.Second,
		Mode:                "dev",
	}
}
