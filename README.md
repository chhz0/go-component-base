# go-component-base

Go语言组件库，为快速开发提供基础组件，各组件设计上将接口设计+合适的设计模式，提供优雅的API调用

项目结构主要为
- `pkg`: 组件包
- `utils`: 工具类

## Components
- cli: 命令行程序库
- config：配置库
- log：日志库
- meta：元数据库
- rest：restful http请求库

## Install
`go get github.com/chhz0/go-component-base`

## Todo
- [httpx]: 兼容gin, fasthttp, echo 等http框架
- [corn]: 定时任务
- [cache]: 缓存
- [grace]: 优雅关闭
- ...
