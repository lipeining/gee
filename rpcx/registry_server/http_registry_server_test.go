package registry_server

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func _assert(condition bool, msg string, v ...interface{}) {
	if !condition {
		panic(fmt.Sprintf("assertion failed: "+msg, v...))
	}
}

func TestHttpRegistryServer(t *testing.T) {
	svr := httptest.NewServer(http.HandlerFunc(DefaultHttpRegisteryerver.ServeHTTP))
	defer svr.Close()

	baseUrl := svr.URL + defaultPath
	log.Println(baseUrl)

	// Post
	data := strings.NewReader("{\"service\":\"store\",\"addr\":\"tcp::9999\"}")
	resp, err := http.Post(baseUrl, "application/json", data)
	_assert(err == nil, "", err)
	_assert(resp.StatusCode == 200, "post got not 200 response")

	// Get
	result := test_get(baseUrl)
	if len(result) != 1 || result[0] != "tcp::9999" {
		_assert(err == nil, "unexpected result:", result)
	}

	// delete
	target := strings.NewReader("{\"service\":\"store\",\"addr\":\"tcp::9999\"}")
	request, err := http.NewRequest("DELETE", baseUrl, target)
	_assert(err == nil, "delte request error", err)

	resp, err = http.DefaultClient.Do(request)
	_assert(err == nil, "", err)
	_assert(resp.StatusCode == 200, "post got not 200 response")

	// Get
	result = test_get(baseUrl)
	if len(result) != 0 {
		_assert(err == nil, "unexpected result:", result)
	}
}

func test_get(baseUrl string) []string {
	resp, err := http.Get(baseUrl + "?service=store")
	_assert(err == nil, "", err)
	_assert(resp.StatusCode == 200, "get got not 200 response")

	result := make([]string, 0)
	err = json.NewDecoder(resp.Body).Decode(&result)
	_assert(err == nil, "", err)

	return result
}
