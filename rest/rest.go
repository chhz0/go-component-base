package rest

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"
)

const (
	defaultTimeout = 30 * time.Second
	maxRetries     = 3
	retryDelay     = 500 * time.Millisecond
)

const (
	TextBodyType      = "text"
	JsonBodyType      = "json"
	MultipartBodyType = "multipart"
	FormBodyType      = "form"
	BinaryBodyType    = "binary"
	XmlBodyType       = "xml"
	HtmlBodyType      = "html"
)

const (
	contentType          = "Content-Type"
	textContentType      = "text/plain"
	jsonContentType      = "application/json"
	multipartContentType = "multipart/form-data"
	formContentType      = "application/x-www-form-urlencoded"
	binaryContentType    = "application/octet-stream"
	xmlContentType       = "application/xml"
	htmlContentType      = "text/html"
)

type Request struct {
	client    *http.Client
	method    string
	url       string
	headers   map[string]string
	query     url.Values
	body      io.Reader
	bodyType  string
	timeout   time.Duration
	retries   int
	multipart []*multipartFile
}

type multipartFile struct {
	fileName string
	filePath string
}

func NewRequest() *Request {
	return &Request{
		client: &http.Client{
			Timeout: defaultTimeout,
		},
		method:   http.MethodGet,
		headers:  make(map[string]string),
		query:    make(url.Values),
		timeout:  defaultTimeout,
		retries:  maxRetries,
		bodyType: JsonBodyType,
	}
}

func (r *Request) MustMethod(method string) *Request {
	r.method = strings.ToUpper(method)
	return r
}

func (r *Request) SetURL(url string) *Request {
	r.url = url
	return r
}

func (r *Request) SetHeaders(headers map[string]string) *Request {
	for k, v := range headers {
		r.headers[k] = v
	}
	return r
}

func (r *Request) AddHeader(key, value string) *Request {
	r.headers[key] = value
	return r
}

func (r *Request) SetQuery(params map[string]string) *Request {
	for k, v := range params {
		r.query.Add(k, v)
	}
	return r
}

func (r *Request) SetJSONBody(data interface{}) *Request {
	body, _ := json.Marshal(data)
	r.body = bytes.NewBuffer(body)
	r.bodyType = JsonBodyType
	r.AddHeader(contentType, jsonContentType)
	return r
}

func (r *Request) SetFormBody(data map[string]string) *Request {
	form := url.Values{}
	for k, v := range data {
		form.Add(k, v)
	}
	r.body = strings.NewReader(form.Encode())
	r.bodyType = FormBodyType
	r.AddHeader(contentType, formContentType)
	return r
}

func (r *Request) AddMultipartFile(fileName, filePath string) *Request {
	r.multipart = append(r.multipart, &multipartFile{
		fileName: fileName,
		filePath: filePath,
	})
	r.bodyType = MultipartBodyType
	return r
}

func (r *Request) SetTimeout(timeout time.Duration) *Request {
	r.timeout = timeout
	return r
}

func (r *Request) SetRetries(retries int) *Request {
	r.retries = retries
	return r
}

func (r *Request) Send() (*Response, error) {

	if r.method == "" {
		return nil, errors.New("method is required")
	}

	var resp *http.Response
	var err error
	var attempt int

	if r.bodyType == MultipartBodyType {
		if err := r.prepareMultipart(); err != nil {
			return nil, err
		}
	}

	fullURL := r.url
	if len(r.query) > 0 {
		fullURL += "?" + r.query.Encode()
	}

	for attempt = 0; attempt <= r.retries; attempt++ {
		ctx, cancel := context.WithTimeout(context.Background(), r.timeout)
		defer cancel()

		req, errr := http.NewRequestWithContext(ctx, r.method, fullURL, r.body)
		if errr != nil {
			return nil, fmt.Errorf("failed to create request: %w", err)
		}

		for key, value := range r.headers {
			req.Header.Set(key, value)
		}

		resp, err = r.client.Do(req)
		if shouldRetry(err) && attempt < r.retries {
			time.Sleep(retryDelay * time.Duration(attempt+1))
			continue
		}
		break
	}

	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	return &Response{
		StatusCode: resp.StatusCode,
		Headers:    resp.Header.Clone(),
		body:       body,
	}, nil
}

func (r *Request) prepareMultipart() error {
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	for _, f := range r.multipart {
		file, err := os.Open(f.filePath)
		if err != nil {
			return err
		}
		defer file.Close()

		part, err := writer.CreateFormFile(f.fileName, f.filePath)
		if err != nil {
			return err
		}

		_, err = io.Copy(part, file)
		if err != nil {
			return err
		}
	}

	if r.body != nil {
		if form, ok := r.body.(*strings.Reader); ok {
			buf, _ := form.ReadByte()
			values, _ := url.ParseQuery(string(buf))
			for k := range values {
				_ = writer.WriteField(k, values.Get(k))
			}
		}
	}

	if err := writer.Close(); err != nil {
		return err
	}

	r.body = body
	r.AddHeader(contentType, writer.FormDataContentType())
	return nil
}

func shouldRetry(err error) bool {
	if err == nil {
		return false
	}

	var urlErr *url.Error
	if errors.As(err, &urlErr) {
		if urlErr.Timeout() || urlErr.Temporary() {
			return true
		}
	}

	return false
}

type Response struct {
	StatusCode int
	Headers    http.Header
	body       []byte
}

func (resp *Response) JSON(v interface{}) error {
	return json.Unmarshal(resp.body, v)
}

func (resp *Response) Text() string {
	return string(resp.body)
}
