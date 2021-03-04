package controller

import (
	"net/http"

	"github.com/dotwallet/touchstone/interceptor"
	"github.com/dotwallet/touchstone/services"
)

type HttpController struct {
	TouchstoneServer *services.TouchstoneServer
}

type SendRawTransactionReq struct {
	RawTx string `json:"rawtx"`
}

func (this *SendRawTransactionReq) New() interceptor.HttpReqBody {
	return new(SendRawTransactionReq)
}

func (this *HttpController) SendRawTransaction(rsp http.ResponseWriter, req *http.Request, httpReqStruct interceptor.HttpReqBody, reqid string) (interface{}, error) {
	requset := httpReqStruct.(*SendRawTransactionReq)
	return this.TouchstoneServer.SendRawTransaction(requset.RawTx)
}

type GetTransactionInventoryReq struct {
	Txid string `json:"txid"`
}

func (this *GetTransactionInventoryReq) New() interceptor.HttpReqBody {
	return new(GetTransactionInventoryReq)
}

func (this *HttpController) GetTransactionInventory(rsp http.ResponseWriter, req *http.Request, httpReqStruct interceptor.HttpReqBody, reqid string) (interface{}, error) {
	requset := httpReqStruct.(*GetTransactionInventoryReq)
	return this.TouchstoneServer.GetTransactionInventory(requset.Txid)
}
