package util

import (
	"bytes"
	"encoding/json"
	"io/ioutil"
	"net/http"
)

const (
	HTTP_METHOD_POST  = "POST"
	HTTP_METHOD_GET   = "GET"
	HTTP_CONTENT_TYPE = "Content-Type"
)

func HttpRequest(method string, url string, headers map[string]string, reqBody interface{}) ([]byte, error) {
	httpClient := &http.Client{}
	content, err := json.Marshal(reqBody)
	if err != nil {
		return nil, err
	}
	request, err := http.NewRequest(method, url, bytes.NewReader(content))
	if err != nil {
		return nil, err
	}
	for key, value := range headers {
		request.Header.Add(key, value)
	}
	resp, err := httpClient.Do(request)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	return body, nil
}

func HttpPost(url string, headers map[string]string, reqBody interface{}) ([]byte, error) {
	return HttpRequest(HTTP_METHOD_POST, url, headers, reqBody)
}

func HttpGet(url string, headers map[string]string, reqBody interface{}) ([]byte, error) {
	return HttpRequest(HTTP_METHOD_GET, url, headers, reqBody)
}
