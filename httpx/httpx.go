package httpx

import (
	"context"
	"encoding/xml"
	"io"
	"mime/multipart"
	"net/http"
	"net/url"
	"sync"

	"github.com/go-playground/form"
	"github.com/gorilla/websocket"
	jsoniter "github.com/json-iterator/go"
)

var (
	json        = jsoniter.ConfigFastest
	formDecoder = form.NewDecoder()
)

// Context 请求上下文接口
type Context interface {
	// 请求信息
	Request() *http.Request
	PathParam(name string) string
	QueryParam(name string) string
	FormParam(name string) string
	Bind(v interface{}) error
	MultipartFile(name string) (*multipart.FileHeader, error)

	// 响应控制
	Status(code int) Context
	Header(key, value string)
	JSON(code int, v interface{}) error
	XML(code int, v interface{}) error
	String(code int, s string) error
	HTML(code int, s string) error
	File(filepath string) error
	Stream(code int, contentType string, r io.Reader) error

	// 元数据存取
	Set(key string, value interface{})
	Get(key string) (interface{}, bool)

	// Websocket 支持
	Websocket() (WebSocketConn, error)
}

// WebSocketConn websocket 连接接口
type WebSocketConn interface {
	ReadJSON(v interface{}) error
	WriteJSON(v interface{}) error
	Close()
}

// Handler 统一的处理函数类型
type Handler func(Context) error

// ErrHandler 统一的错误处理函数类型
type ErrHandler func(Context, error)

// Middleware 中间件类型
type Middleware func(Handler) Handler

// Router 路由器接口
type Router interface {
	GET(path string, h Handler, ms ...Middleware)
	POST(path string, h Handler, ms ...Middleware)
	PUT(path string, h Handler, ms ...Middleware)
	DELETE(path string, h Handler, ms ...Middleware)
	PATCH(path string, h Handler, ms ...Middleware)
	HEAD(path string, h Handler, ms ...Middleware)
	OPTIONS(path string, h Handler, ms ...Middleware)

	Static(prefix, root string)
	Group(prefix string, ms ...Middleware) Router
	Use(ms ...Middleware)
}

// Adapter 适配器接口
type Adapter interface {
	NewRouter() Router
	ConverHandler(h Handler) interface{}
	WrapContext(ctx interface{}) Context
	Serve(addr string) error
	Shutdown(ctx context.Context) error
}

var (
	adapter     Adapter
	contextPool = &sync.Pool{
		New: func() any {
			return &baseContext{
				store: make(map[string]interface{}),
			}
		},
	}
)

// 在适配器中获取上下文
func getContext() *baseContext {
	return contextPool.Get().(*baseContext)
}

// 释放上下文
func releaseContext(ctx *baseContext) {
	ctx.reset()
	contextPool.Put(ctx)
}

// 基础上下文实现
type baseContext struct {
	request  *http.Request
	response http.ResponseWriter
	params   map[string]string // 路径参数
	query    url.Values        // 缓存查询参数
	form     url.Values        // 缓存表单参数
	files    map[string][]*multipart.FileHeader
	store    map[string]interface{} // 元数据
	status   int                    // 响应状态码
}

// 重置上下文状态, 用于对象池
func (c *baseContext) reset() {
	c.request = nil
	c.response = nil
	c.params = nil
	c.query = nil
	c.form = nil
	c.files = nil
	c.status = 0
	for k := range c.store {
		delete(c.store, k)
	}
}

// Request implements Context.
func (c *baseContext) Request() *http.Request { return c.request }

// PathParam implements Context.
func (c *baseContext) PathParam(name string) string {
	return c.params[name]
}

// QueryParam implements Context.
func (c *baseContext) QueryParam(name string) string {
	if c.query == nil {
		c.query = c.request.URL.Query()
	}
	return c.query.Get(name)
}

// FormParam implements Context.
func (c *baseContext) FormParam(name string) string {
	if c.form == nil {
		c.parseForm()
	}
	return c.form.Get(name)
}

func (c *baseContext) parseForm() {
	if c.form != nil {
		return
	}

	if c.request.Form == nil {
		_ = c.request.ParseForm()
	}
	c.form = c.request.Form
}

// Bind implements Context.
func (c *baseContext) Bind(v interface{}) error {
	contentType := c.request.Header.Get("Content-Type")
	switch {
	case contentType == "application/json":
		return c.bindJSON(v)
	case contentType == "application/xml":
		return c.bindXML(v)
	default:
		return c.bindForm(v)
	}
}

func (c *baseContext) bindJSON(v interface{}) error {
	body, err := io.ReadAll(c.request.Body)
	if err != nil {
		return err
	}
	return json.Unmarshal(body, v)
}

func (c *baseContext) bindXML(v interface{}) error {
	body, err := io.ReadAll(c.request.Body)
	if err != nil {
		return err
	}
	return xml.Unmarshal(body, v)
}

func (c *baseContext) bindForm(v interface{}) error {
	c.parseForm()

	return formDecoder.Decode(v, c.form)
}

// func (c *baseContext) parseForm() error {
// }

// MultipartFile implements Context.
func (c *baseContext) MultipartFile(name string) (*multipart.FileHeader, error) {
	if c.files == nil {
		if err := c.parseMultipartForm(); err != nil {
			return nil, err
		}
	}
	files := c.files[name]
	if len(files) == 0 {
		return nil, http.ErrMissingFile
	}
	return files[0], nil
}

func (c *baseContext) parseMultipartForm() error {
	if c.files != nil {
		return nil
	}

	if err := c.request.ParseMultipartForm(32 << 20); // 32MB
	err != nil {
		return err
	}

	c.form = c.request.PostForm
	c.files = c.request.MultipartForm.File
	return nil
}

// Status implements Context.
func (c *baseContext) Status(code int) Context {
	c.status = code
	c.response.WriteHeader(code)
	return c
}

// Header implements Context.
func (c *baseContext) Header(key string, value string) {
	c.response.Header().Set(key, value)
}

// JSON implements Context.
func (c *baseContext) JSON(code int, v interface{}) error {
	c.Header("Content-Type", "application/json")
	c.Status(code)
	return json.NewEncoder(c.response).Encode(v)
}

// XML implements Context.
func (c *baseContext) XML(code int, v interface{}) error {
	c.Header("Content-Type", "application/xml")
	c.Status(code)
	return xml.NewEncoder(c.response).Encode(v)
}

// String implements Context.
func (c *baseContext) String(code int, s string) error {
	c.Header("Content-Type", "text/plain")
	c.Status(code)
	_, err := io.WriteString(c.response, s)
	return err
}

// HTML implements Context.
func (c *baseContext) HTML(code int, s string) error {
	c.Header("Content-Type", "text/html")
	c.Status(code)
	_, err := io.WriteString(c.response, s)
	return err
}

// File implements Context.
func (c *baseContext) File(filepath string) error {
	http.ServeFile(c.response, c.request, filepath)
	return nil
}

// Stream implements Context.
func (c *baseContext) Stream(code int, contentType string, r io.Reader) error {
	c.Header("Content-Type", contentType)
	c.Status(code)
	_, err := io.Copy(c.response, r)
	return err
}

// Set implements Context.
func (c *baseContext) Set(key string, value interface{}) {
	if c.store == nil {
		c.store = make(map[string]interface{})
	}
	c.store[key] = value
}

// Get implements Context.
func (c *baseContext) Get(key string) (interface{}, bool) {
	if c.store == nil {
		return nil, false
	}
	val, ok := c.store[key]
	return val, ok
}

// Websocket implements Context.
func (c *baseContext) Websocket() (WebSocketConn, error) {
	upgrader := websocket.Upgrader{
		CheckOrigin: func(r *http.Request) bool { return true },
	}
	conn, err := upgrader.Upgrade(c.response, c.request, nil)
	if err != nil {
		return nil, err
	}
	return &wsConn{conn}, nil
}

type wsConn struct {
	*websocket.Conn
}

// ReadJSON implements WebSocketConn.
// Subtle: this method shadows the method (*Conn).ReadJSON of wsConn.Conn.
func (c *wsConn) ReadJSON(v interface{}) error {
	return c.Conn.ReadJSON(v)
}

// WriteJSON implements WebSocketConn.
// Subtle: this method shadows the method (*Conn).WriteJSON of wsConn.Conn.
func (c *wsConn) WriteJSON(v interface{}) error {
	return c.Conn.WriteJSON(v)
}

// Close implements WebSocketConn.
// Subtle: this method shadows the method (*Conn).Close of wsConn.Conn.
func (c *wsConn) Close() {
	_ = c.Conn.Close()
}
