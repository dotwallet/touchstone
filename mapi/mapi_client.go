package mapi

import (
	"encoding/json"
)

const (
	RETURN_RESULT_SUCCESS = "success"
	RETURN_RESULT_FAILURE = "failure"
)

type MapiCertInfo struct {
	Signature string `json:"signature"`
	PublicKey string `json:"publicKey"`
	Encoding  string `json:"encoding"`
	Mimetype  string `json:"mimetype"`
}

type MapiClientAdaptor interface {
	GetFeeQuote() (*MapiResponse, error)
	GetTxState(txid string) (*MapiResponse, error)
	SendTx(sendTxRequest *SendTxRequest) (*MapiResponse, error)
}

type MapiResponse struct {
	MapiCertInfo
	Payload string `json:"payload"`
}

type MapiClient struct {
	MapiClientAdaptor
}

type Fee struct {
	Satoshis int `json:"satoshis"`
	Bytes    int `json:"bytes"`
}

type FeeInfo struct {
	FeeType   string `json:"feeType"`
	MiningFee *Fee   `json:"miningFee"`
	RelayFee  *Fee   `json:"relayFee"`
}

type FeeQuotePayload struct {
	ApiVersion                string     `json:"apiVersion"`
	Timestamp                 string     `json:"timestamp"`
	ExpiryTime                string     `json:"expiryTime"`
	MinerId                   string     `json:"minerId"`
	CurrentHighestBlockHash   string     `json:"currentHighestBlockHash"`
	CurrentHighestBlockHeight int64      `json:"currentHighestBlockHeight"`
	MinerReputation           string     `json:"minerReputation"`
	FeeInfos                  []*FeeInfo `json:"fees"`
}

type FeeQuote struct {
	Payload FeeQuotePayload `json:"payload"`
	MapiCertInfo
}

func (this *MapiClient) GetFeeQuote() (*FeeQuote, error) {
	mapiFeeQuote, err := this.MapiClientAdaptor.GetFeeQuote()
	if err != nil {
		return nil, err
	}
	feeQuote := &FeeQuote{}
	feeQuote.MapiCertInfo = mapiFeeQuote.MapiCertInfo
	err = json.Unmarshal([]byte(mapiFeeQuote.Payload), &feeQuote.Payload)
	if err != nil {
		return nil, err
	}
	return feeQuote, nil
}

type TxStatePayload struct {
	ApiVersion            string `json:"apiVersion"`
	Timestamp             string `json:"timestamp"`
	ReturnResult          string `json:"returnResult"`
	ResultDescription     string `json:"resultDescription"`
	BlockHash             string `json:"blockHash"`
	BlockHeight           int64  `json:"blockHeight"`
	MinerId               string `json:"minerId"`
	Confirmations         int64  `json:"confirmations"`
	TxSecondMempoolExpiry int64  `json:"txSecondMempoolExpiry"`
}

type TxState struct {
	Payload TxStatePayload `json:"payload"`
	MapiCertInfo
}

func (this *MapiClient) GetTxState(txid string) (*TxState, error) {
	mapiTxState, err := this.MapiClientAdaptor.GetTxState(txid)
	if err != nil {
		return nil, err
	}
	txState := &TxState{}
	txState.MapiCertInfo = mapiTxState.MapiCertInfo
	err = json.Unmarshal([]byte(mapiTxState.Payload), &txState.Payload)
	if err != nil {
		return nil, err
	}
	return txState, nil
}

type SendTxResultPayload struct {
	ApiVersion                string `json:"apiVersion"`
	Timestamp                 string `json:"timestamp"`
	Txid                      string `json:"txid"`
	ReturnResult              string `json:"returnResult"`
	MinerId                   string `json:"minerId"`
	CurrentHighestBlockHash   string `json:"currentHighestBlockHash"`
	CurrentHighestBlockHeight int64  `json:"currentHighestBlockHeight"`
	TxSecondMempoolExpiry     int64  `json:"txSecondMempoolExpiry"`
	ResultDescription         string `json:"resultDescription"`
}

type SendTxResult struct {
	MapiCertInfo
	Payload SendTxResultPayload `json:"payload"`
}

type SendTxRequest struct {
	RawTx string `json:"rawtx"`
}

func (this *MapiClient) SendTx(rawTx string) (*SendTxResult, error) {
	sendTxRequest := &SendTxRequest{
		RawTx: rawTx,
	}
	mapiSendTxResult, err := this.MapiClientAdaptor.SendTx(sendTxRequest)
	if err != nil {
		return nil, err
	}
	sendTxResult := &SendTxResult{}
	sendTxResult.MapiCertInfo = mapiSendTxResult.MapiCertInfo
	err = json.Unmarshal([]byte(mapiSendTxResult.Payload), &sendTxResult.Payload)
	if err != nil {
		return nil, err
	}
	return sendTxResult, nil
}
