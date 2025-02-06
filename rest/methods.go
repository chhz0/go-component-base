package rest

import "net/http"

// Get 函数遵守RESTful 规范，使用GET请求时应通过URL路径或者查询字符串传递参数
// URL路径：/api/v1/users/1
// 查询字符串：/api/v1/users?id=1
func Get(url, urlSuffix string, queries, headers map[string]string) (*Response, error) {
	return NewRequest().
		MustMethod(http.MethodGet).
		SetURL(url + "/" + urlSuffix).
		SetQuery(queries).
		SetHeaders(headers).
		Send()
}

// Post 函数遵守RESTful 规范，用于创建新资源，或执行非幂等操作
func Post(url string, queries, headers map[string]string,
	bodyType string, body interface{}, multipartFiles map[string]string) (*Response, error) {
	postReq := NewRequest().MustMethod(http.MethodPost).
		SetURL(url).
		SetHeaders(headers)

	switch bodyType {
	case FormBodyType:
		postReq.SetFormBody(body.(map[string]string))
	case JsonBodyType:
		postReq.SetJSONBody(body)
	// case binaryBodyType:
	// 	postReq.SetBinaryBody(body.([]byte))
	case MultipartBodyType:
		for fn, fp := range multipartFiles {
			postReq.AddMultipartFile(fn, fp)
		}
	// case xmlBodyType:
	// 	postReq.SetXMLBody(body)
	// case htmlBodyType:
	// 	postReq.SetHTMLBody(body.(string))
	default:
	}

	return postReq.Send()
}

// Put 全量更新资源
func Put(url string, queries, headers map[string]string) (*Response, error) {
	resp, err := NewRequest().
		MustMethod(http.MethodPut).
		SetURL(url).
		SetQuery(queries).
		SetHeaders(headers).
		Send()

	return resp, err
}

// Delete 删除资源, 删除指定 URL 的资源
func Delete(url string, queries, headers map[string]string) (*Response, error) {
	resp, err := NewRequest().
		MustMethod(http.MethodDelete).
		SetURL(url).
		SetQuery(queries).
		SetHeaders(headers).
		Send()

	return resp, err
}

// Head 获取资源的元数据, 与Get相同，但无响应体
func Head(url string, queries, headers map[string]string) (http.Header, error) {
	resp, err := NewRequest().
		MustMethod(http.MethodPut).
		SetURL(url).
		SetQuery(queries).
		SetHeaders(headers).
		Send()

	return resp.Headers, err
}

// Patch 部分更新资源, 与Put相同，但只更新部分字段
func Patch(url string, queries, headers map[string]string) (*Response, error) {
	resp, err := NewRequest().
		MustMethod(http.MethodPatch).
		SetURL(url).
		SetQuery(queries).
		SetHeaders(headers).
		Send()

	return resp, err
}

// Options 获取资源支持的 HTTP 方法, 用于 CORS 预检请求（Preflight）
func Options(url string, queries, headers map[string]string) (http.Header, error) {
	resp, err := NewRequest().
		MustMethod(http.MethodOptions).
		SetURL(url).
		SetQuery(queries).
		SetHeaders(headers).
		Send()

	return resp.Headers, err
}

// TODO: 待实现
func Trace()   {}
func Connect() {}
