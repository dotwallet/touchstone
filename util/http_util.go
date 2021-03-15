package util

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"
)

const (
	HTTP_METHOD_POST  = "POST"
	HTTP_METHOD_GET   = "GET"
	HTTP_CONTENT_TYPE = "Content-Type"
)

func ToCurlStr(method string, header map[string]string, body []byte, url string) {
	var b strings.Builder
	b.WriteString("curl ")
	for key, value := range header {
		b.WriteString("-H ")
		b.WriteString("\"")
		b.WriteString(key)
		b.WriteString(":")
		b.WriteString(value)
		b.WriteString("\" ")
	}
	b.WriteString("-X ")
	b.WriteString(method)

	b.WriteString(" --data '")
	b.Write(body)
	b.WriteString("' ")
	b.WriteString(url)
	fmt.Println(b.String())
}

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
	// ToCurlStr(method, headers, content, url)
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
