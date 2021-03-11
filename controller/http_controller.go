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

func (this *SendRawTransactionReq) NewHttpReqBody() interceptor.HttpReqBody {
	return new(SendRawTransactionReq)
}

func (this *HttpController) SendRawTransaction(rsp http.ResponseWriter, req *http.Request, httpReqStruct interceptor.HttpReqBody, reqid string) (interface{}, error) {
	request := httpReqStruct.(*SendRawTransactionReq)
	return this.TouchstoneServer.SendRawTransaction(request.RawTx)
}

type GetTransactionInventoryReq struct {
	Txid string `json:"txid"`
}

func (this *GetTransactionInventoryReq) NewHttpReqBody() interceptor.HttpReqBody {
	return new(GetTransactionInventoryReq)
}

func (this *HttpController) GetTransactionInventory(rsp http.ResponseWriter, req *http.Request, httpReqStruct interceptor.HttpReqBody, reqid string) (interface{}, error) {
	request := httpReqStruct.(*GetTransactionInventoryReq)
	return this.TouchstoneServer.GetTransactionInventory(request.Txid)
}

type GetAddrUtxosReq struct {
	Addr      *string `json:"addr"`
	BadgeCode *string `json:"badge_code"`
	Offset    int     `json:"offset"`
	Limit     int     `json:"limit"`
}

func (this *GetAddrUtxosReq) NewHttpReqBody() interceptor.HttpReqBody {
	return &GetAddrUtxosReq{
		Offset: 0,
		Limit:  10,
	}
}

func (this *HttpController) GetAddrUtxos(rsp http.ResponseWriter, req *http.Request, httpReqStruct interceptor.HttpReqBody, reqid string) (interface{}, error) {
	request := httpReqStruct.(*GetAddrUtxosReq)
	return this.TouchstoneServer.GetAddrUtxos(*request.Addr, *request.BadgeCode, request.Offset, request.Limit)
}

type GetAddrBalanceReq struct {
	Addr      *string `json:"addr"`
	BadgeCode *string `json:"badge_code"`
}

func (this *GetAddrBalanceReq) NewHttpReqBody() interceptor.HttpReqBody {
	return &GetAddrBalanceReq{}
}

func (this *HttpController) GetAddrBalance(rsp http.ResponseWriter, req *http.Request, httpReqStruct interceptor.HttpReqBody, reqid string) (interface{}, error) {
	request := httpReqStruct.(*GetAddrBalanceReq)
	return this.TouchstoneServer.GetAddrBalance(*request.Addr, *request.BadgeCode)
}

type GetAddrInventorysReq struct {
	Addr      *string `json:"addr"`
	BadgeCode *string `json:"badge_code"`
	Offset    int     `json:"offset"`
	Limit     int     `json:"limit"`
}

func (this *GetAddrInventorysReq) NewHttpReqBody() interceptor.HttpReqBody {
	return &GetAddrInventorysReq{
		Offset: 0,
		Limit:  10,
	}
}

func (this *HttpController) GetAddrInventorys(rsp http.ResponseWriter, req *http.Request, httpReqStruct interceptor.HttpReqBody, reqid string) (interface{}, error) {
	request := httpReqStruct.(*GetAddrInventorysReq)
	return this.TouchstoneServer.GetAddrInventorys(*request.Addr, *request.BadgeCode, request.Offset, request.Limit)
}

type SetAddrInfoReq struct {
	Appid     *string `json:"appid"`
	UserID    *int64  `json:"userid"`
	UserIndex *int64  `json:"user_index"`
	Addr      *string `json:"addr"`
}

func (this *SetAddrInfoReq) NewHttpReqBody() interceptor.HttpReqBody {
	return new(SetAddrInfoReq)
}

func (this *HttpController) SetAddrInfo(rsp http.ResponseWriter, req *http.Request, httpReqStruct interceptor.HttpReqBody, reqid string) (interface{}, error) {
	request := httpReqStruct.(*SetAddrInfoReq)
	err := this.TouchstoneServer.SetAddrInfo(*request.Appid, *request.UserID, *request.UserIndex, *request.Addr)
	return nil, err
}

type GetUserUtxosReq struct {
	Appid     *string `json:"appid"`
	UserID    *int64  `json:"userid"`
	UserIndex *int64  `json:"user_index"`
	BadgeCode *string `json:"badge_code"`
	Offset    int     `json:"offset"`
	Limit     int     `json:"limit"`
}

func (this *GetUserUtxosReq) NewHttpReqBody() interceptor.HttpReqBody {
	return &GetUserUtxosReq{
		Offset: 0,
		Limit:  10,
	}
}

func (this *HttpController) GetUserUtxos(rsp http.ResponseWriter, req *http.Request, httpReqStruct interceptor.HttpReqBody, reqid string) (interface{}, error) {
	request := httpReqStruct.(*GetUserUtxosReq)
	return this.TouchstoneServer.GetUserUtxos(*request.Appid, *request.UserID, *request.UserIndex, *request.BadgeCode, request.Offset, request.Limit)
}

type GetUserBalanceReq struct {
	Appid     *string `json:"appid"`
	UserID    *int64  `json:"userid"`
	UserIndex *int64  `json:"user_index"`
	BadgeCode *string `json:"badge_code"`
}

func (this *GetUserBalanceReq) NewHttpReqBody() interceptor.HttpReqBody {
	return &GetUserBalanceReq{}
}

func (this *HttpController) GetUserBalance(rsp http.ResponseWriter, req *http.Request, httpReqStruct interceptor.HttpReqBody, reqid string) (interface{}, error) {
	request := httpReqStruct.(*GetUserBalanceReq)
	return this.TouchstoneServer.GetUserBalance(*request.Appid, *request.UserID, *request.UserIndex, *request.BadgeCode)
}

type GetUserInventorysReq struct {
	Appid     *string `json:"appid"`
	UserID    *int64  `json:"userid"`
	UserIndex *int64  `json:"user_index"`
	BadgeCode *string `json:"badge_code"`
	Offset    int     `json:"offset"`
	Limit     int     `json:"limit"`
}

func (this *GetUserInventorysReq) NewHttpReqBody() interceptor.HttpReqBody {
	return &GetUserInventorysReq{
		Offset: 0,
		Limit:  10,
	}
}

func (this *HttpController) GetUserInventorys(rsp http.ResponseWriter, req *http.Request, httpReqStruct interceptor.HttpReqBody, reqid string) (interface{}, error) {
	request := httpReqStruct.(*GetUserInventorysReq)
	return this.TouchstoneServer.GetUserInventorys(*request.Appid, *request.UserID, *request.UserIndex, *request.BadgeCode, request.Offset, request.Limit)
}

type SendBadgeToAddressReq struct {
	Appid       *string                `json:"appid"`
	UserID      *int64                 `json:"userid"`
	UserIndex   *int64                 `json:"user_index"`
	BadgeCode   *string                `json:"badge_code"`
	ChangeAddr  *string                `json:"change_addr"`
	AddrAmounts []*services.AddrAmount `json:"addr_amounts"`
	Amount2Burn int64                  `json:"amount2burn"`
}

func (this *SendBadgeToAddressReq) NewHttpReqBody() interceptor.HttpReqBody {
	return &SendBadgeToAddressReq{}
}

func (this *HttpController) SendBadgeToAddress(rsp http.ResponseWriter, req *http.Request, httpReqStruct interceptor.HttpReqBody, reqid string) (interface{}, error) {
	request := httpReqStruct.(*SendBadgeToAddressReq)
	return this.TouchstoneServer.SendBadgeToAddress(*request.Appid, *request.UserID, *request.UserIndex, *request.BadgeCode, *request.ChangeAddr, request.AddrAmounts, request.Amount2Burn)
}
