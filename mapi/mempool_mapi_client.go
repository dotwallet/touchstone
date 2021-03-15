package mapi

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"

	"github.com/btcsuite/btcd/btcec"
	"github.com/btcsuite/btcd/chaincfg"
	"github.com/btcsuite/btcutil/hdkeychain"
	"github.com/dotwallet/touchstone/util"
	"github.com/golang/glog"
	"github.com/tyler-smith/go-bip39"
)

const (
	STR_CLIENT_ID     = "client_id"
	STR_CLIENT_SRCRET = "client_secret"
	STR_GRANDET_TYPE  = "grant_type"
)

func getExtendedKey(rootExtendedKey *hdkeychain.ExtendedKey) (*hdkeychain.ExtendedKey, error) {
	var err error
	extendedKey := rootExtendedKey
	// paths := "m/44'/0'/0'/0/0"
	paths := []uint32{hdkeychain.HardenedKeyStart + 44,
		hdkeychain.HardenedKeyStart + uint32(0),
		hdkeychain.HardenedKeyStart + uint32(0),
		uint32(0),
		uint32(0)}
	for _, path := range paths {
		extendedKey, err = extendedKey.Child(path)
		if err != nil {
			glog.Infof("Get child extentdKey error, error: %s", err)
			return nil, err
		}
	}
	return extendedKey, nil
}

type MempoolMapiAdaptor struct {
	Host           string
	privateKey     *btcec.PrivateKey
	getFeeQuoteUrl string
	getTxStateUrl  string
	sendRawTxUrl   string
}

func (this *MempoolMapiAdaptor) doRequest(method string, url string, headers map[string]string, param interface{}) (*MapiResponse, error) {
	paramByte, err := json.Marshal(param)
	if err != nil {
		return nil, err
	}
	hashForSign := sha256.Sum256(paramByte)
	signature, err := this.privateKey.Sign(hashForSign[:])
	if err != nil {
		return nil, err
	}
	signatureBytes := signature.Serialize()
	publicKeyBytes := this.privateKey.PubKey().SerializeCompressed()
	request := make(map[string]interface{})
	request["payload"] = string(paramByte)
	request["signature"] = hex.EncodeToString(signatureBytes)
	request["pubKey"] = hex.EncodeToString(publicKeyBytes)

	newHeader := make(map[string]string)
	for key, value := range headers {
		newHeader[key] = value
	}
	newHeader[util.HTTP_CONTENT_TYPE] = "application/json"

	httpResult, err := util.HttpRequest(method, url, newHeader, request)
	if err != nil {
		return nil, err
	}
	result := &MapiResponse{}
	err = json.Unmarshal(httpResult, result)
	return result, err
}

func (this *MempoolMapiAdaptor) GetFeeQuote() (*MapiResponse, error) {
	return this.doRequest(util.HTTP_METHOD_GET, this.getFeeQuoteUrl, nil, nil)
}
func (this *MempoolMapiAdaptor) GetTxState(txid string) (*MapiResponse, error) {
	url := this.getTxStateUrl + txid
	return this.doRequest(util.HTTP_METHOD_GET, url, nil, nil)

}
func (this *MempoolMapiAdaptor) SendTx(sendTxRequest *SendTxRequest) (*MapiResponse, error) {
	return this.doRequest(util.HTTP_METHOD_POST, this.sendRawTxUrl, nil, sendTxRequest)
}

func NewMempoolMapiClient(host string, mnemonicWords string, password string) (*MapiClient, error) {
	seed := bip39.NewSeed(mnemonicWords, password)
	rootExtendKey, err := hdkeychain.NewMaster(seed, &chaincfg.MainNetParams)
	if err != nil {
		return nil, err
	}

	extendKey, err := getExtendedKey(rootExtendKey)
	if err != nil {
		return nil, err
	}

	privateKey, err := extendKey.ECPrivKey()
	if err != nil {
		return nil, err
	}
	getFeeQuoteUrl := fmt.Sprintf("%s/v1/mapi/feeQuote", host)
	getTxStateUrl := fmt.Sprintf("%s/v1/mapi/tx/", host)
	sendRawTxUrl := fmt.Sprintf("%s/v1/mapi/tx", host)
	mempoolMapiAdaptor := &MempoolMapiAdaptor{
		privateKey:     privateKey,
		Host:           host,
		getFeeQuoteUrl: getFeeQuoteUrl,
		getTxStateUrl:  getTxStateUrl,
		sendRawTxUrl:   sendRawTxUrl,
	}
	return &MapiClient{
		MapiClientAdaptor: mempoolMapiAdaptor,
	}, nil
}
