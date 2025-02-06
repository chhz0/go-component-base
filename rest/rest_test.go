package rest

import (
	"net/http"
	"testing"
)

type Result struct {
}

// GET请求示例
func Test_getExample(t *testing.T) {
	resp, err := NewRequest().
		MustMethod(http.MethodGet).
		SetURL("http://127.0.0.1:8080/ping").
		SetQuery(map[string]string{
			"name":  "John",
			"email": "john@example.com",
		}).
		Send()

	if err != nil {
		t.Log(err)
	}

	t.Log(string(resp.body))
}

// POST JSON示例
func Test_postJSONExample(t *testing.T) {
	data := map[string]interface{}{
		"name":  "John",
		"email": "john@example.com",
	}

	resp, err := NewRequest().
		MustMethod(http.MethodPost).
		SetURL("https://api.example.com/users").
		SetJSONBody(data).
		Send()

	t.Log(resp, err)
}

// 文件上传示例
func Test_uploadExample(t *testing.T) {
	resp, err := NewRequest().
		MustMethod(http.MethodPost).
		SetURL("https://api.example.com/upload").
		AddMultipartFile("file", "/path/to/file.jpg").
		SetFormBody(map[string]string{
			"description": "My photo",
		}).
		Send()
	t.Log(resp, err)
}
