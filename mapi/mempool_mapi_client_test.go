package mapi

import (
	"encoding/json"
	"fmt"
	"testing"
)

var gMempoolMapiClient *MapiClient

func init() {
	// mapiClient, err := NewMempoolMapiClient("https://api.ddpurse.com", "border napkin domain blush hammer what avocado venue delay network tell art", "")
	mapiClient, err := NewMempoolMapiClient("http://192.168.1.13:6001", "border napkin domain blush hammer what avocado venue delay network tell art", "")
	if err != nil {
		panic(err)
	}
	gMempoolMapiClient = mapiClient
}

func ToJson(data interface{}) string {
	b, err := json.Marshal(data)
	if err != nil {
		panic(err)
	}
	return string(b)
}

func TestGetTxState(t *testing.T) {
	result, err := gMempoolMapiClient.GetTxState("e624fd69683d27c48982e3e62e1e73b276e7b4c7763c514c00091cbcff19f700")
	if err != nil {
		t.Fatal(err)
	}
	fmt.Println(ToJson(result))
}
