package models

import (
	"errors"
	"fmt"
	"strings"

	"github.com/btcsuite/btcd/wire"
	"github.com/dotwallet/touchstone/util"

	"gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"
)

const (
	TBL_RAW_TX_INFO     = "raw_tx_info"
	MAX_SEGMENT_SIZE    = 131070
	TX_INFO_INDEX       = -1
	UNCONFIRM_TX_HEIGHT = -1
	TX_STATE_NEW        = 1
	TX_STATE_OPEN       = 2
	TX_STATE_CLOSED     = 3
)

type RawTxInfo struct {
	Txid      string `bson:"txid"`
	Index     int    `bson:"index"`
	Data      string `bson:"data,omitempty"`
	Height    int64  `bson:"height,omitempty"`
	BlockHash string `bson:"blockhash,omitempty"`
	Timestamp int64  `bson:"timestamp,omitempty"`
	State     int    `bson:"state,omitempty"`
}

type MsgTxInfo struct {
	MsgTx     *wire.MsgTx
	Height    int64
	BlockHash string
	Timestamp int64
	State     int
}

type TxInfoRepository struct {
	Db *MongoDb
}

func (this *TxInfoRepository) TableName() string {
	return TBL_RAW_TX_INFO
}

func (this *TxInfoRepository) CreateIndex() error {
	return this.Db.CreateIndex(
		this.TableName(),
		[]*mgo.Index{
			{
				Key:    []string{TXID, INDEX},
				Unique: true,
			},
			{
				Key:    []string{HEIGHT},
				Unique: false,
			},
		},
	)
}

func (this *TxInfoRepository) IsMsgTxClosed(txid string) (bool, error) {
	condition := bson.M{
		TXID:  txid,
		INDEX: TX_INFO_INDEX,
		STATE: TX_STATE_CLOSED,
	}
	count, err := this.Db.Count(this.TableName(), condition)
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

type MsgTxBriefInfo struct {
	Txid      string `bson:"txid"`
	Height    int64  `bson:"height"`
	BlockHash string `bson:"blockhash"`
	Timestamp int64  `bson:"timestamp"`
	State     int    `bson:"state"`
}

func (this *TxInfoRepository) SetMsgTxState(txid string, state int) error {
	switch state {
	case TX_STATE_NEW, TX_STATE_OPEN, TX_STATE_CLOSED:
		condition := bson.M{
			TXID:  txid,
			INDEX: TX_INFO_INDEX,
		}
		updator := bson.M{
			STATE: state,
		}
		return this.Db.UpdateOne(this.TableName(), condition, updator)
	}
	errStr := fmt.Sprintf("not support state %d", state)
	return errors.New(errStr)
}

func (this *TxInfoRepository) SetMsgTxHeightHash(txid string, height int64, hash string) error {
	condition := bson.M{
		TXID:  txid,
		INDEX: TX_INFO_INDEX,
	}
	updator := bson.M{
		HASH:   hash,
		HEIGHT: height,
	}
	return this.Db.UpdateOne(this.TableName(), condition, updator)
}

func (this *TxInfoRepository) GetMsgTxBriefInfo(txid string) (*MsgTxBriefInfo, error) {
	condition := bson.M{
		TXID:  txid,
		INDEX: TX_INFO_INDEX,
	}
	msgTxBriefInfo := &MsgTxBriefInfo{}
	err := this.Db.GetOne(this.TableName(), condition, nil, msgTxBriefInfo)
	return msgTxBriefInfo, err
}

func (this *TxInfoRepository) GetMsgTxBriefInfoByHeightRange(startHeight int64, endHeight int64, unconfirm bool) ([]*MsgTxBriefInfo, error) {
	result := make([]*MsgTxBriefInfo, 0, 128)
	condition := bson.M{
		HEIGHT: bson.M{MONGO_OPERATOR_GTE: startHeight, MONGO_OPERATOR_LT: endHeight},
		INDEX:  TX_INFO_INDEX,
	}
	if unconfirm {
		condition = bson.M{
			MONGO_OPERATOR_OR: []bson.M{
				{HEIGHT: bson.M{MONGO_OPERATOR_GTE: startHeight, MONGO_OPERATOR_LT: endHeight}},
				{HEIGHT: UNCONFIRM_TX_HEIGHT},
			},
			INDEX: TX_INFO_INDEX,
		}
	}
	err := this.Db.GetAll(this.TableName(), condition, nil, MONGO_ID, &result)
	return result, err
}

func (this *TxInfoRepository) GetMsgTxInfo(txid string) (*MsgTxInfo, error) {
	condition := bson.M{
		TXID: txid,
	}
	offset := 0
	limit := 128
	msgTxInfo := &MsgTxInfo{}
	rawTxParts := make([]string, 0, 4)
	completed := false
	for {
		transactionInfosTmp := make([]*RawTxInfo, 0, 8)
		err := this.Db.GetMany(this.TableName(), condition, nil, INDEX, offset, limit, &transactionInfosTmp)
		if err != nil {
			return nil, err
		}
		for _, transactionInfo := range transactionInfosTmp {
			if transactionInfo.Index == TX_INFO_INDEX {
				msgTxInfo.Height = transactionInfo.Height
				msgTxInfo.BlockHash = transactionInfo.BlockHash
				msgTxInfo.Timestamp = transactionInfo.Timestamp
				msgTxInfo.State = transactionInfo.State
				completed = true
				continue
			}
			rawTxParts = append(rawTxParts, transactionInfo.Data)
		}
		if len(transactionInfosTmp) < limit/2 {
			break
		}
	}
	if !completed {
		return nil, errors.New(MONGO_NOT_FOUND)
	}
	rawtx := strings.Join(rawTxParts, "")
	msgTx, err := util.DeserializeTxStr(rawtx)
	if err != nil {
		//todo
		return nil, err
	}
	msgTxInfo.MsgTx = msgTx
	return msgTxInfo, nil
}

func (this *TxInfoRepository) AddMsgTxInfo(msgTx *wire.MsgTx, Height int64, BlockHash string, Timestamp int64) error {
	rawTx := util.SeserializeMsgTxStr(msgTx)
	piecewiseRawTxs := util.SplitString(rawTx, MAX_SEGMENT_SIZE)
	for index, PiecewiseRawTx := range piecewiseRawTxs {
		rawTxInfo := &RawTxInfo{
			Txid:  msgTx.TxHash().String(),
			Index: index,
			Data:  PiecewiseRawTx,
		}
		err := this.Db.Insert(this.TableName(), rawTxInfo)
		if err != nil {
			if !strings.Contains(err.Error(), MONGO_ERROR_DUPLICATE) {
				return err
			}
		}
	}
	rawTxInfo := &RawTxInfo{
		Txid:      msgTx.TxHash().String(),
		Index:     TX_INFO_INDEX,
		Height:    Height,
		BlockHash: BlockHash,
		Timestamp: Timestamp,
		State:     TX_STATE_NEW,
	}
	err := this.Db.Insert(this.TableName(), rawTxInfo)
	if err != nil {
		if !strings.Contains(err.Error(), MONGO_ERROR_DUPLICATE) {
			return err
		}
	}
	return nil
}

type TxidBson struct {
	Txid string `bson:"txid"`
}

func (this *TxInfoRepository) GetTxidsByHeightRangeOrderByTxid(startHeight int64, endHeight int64, State int, unconfirm bool) ([]*TxidBson, error) {
	selector := bson.M{
		TXID: true,
	}
	txidBsons := make([]*TxidBson, 0, 1024)
	condition := bson.M{
		INDEX: TX_INFO_INDEX,
		STATE: State,
		HEIGHT: bson.M{
			MONGO_OPERATOR_GTE: startHeight,
			MONGO_OPERATOR_LT:  endHeight,
		},
	}
	if unconfirm {
		condition = bson.M{
			INDEX: TX_INFO_INDEX,
			STATE: State,
			MONGO_OPERATOR_OR: []bson.M{
				{HEIGHT: bson.M{MONGO_OPERATOR_GTE: startHeight, MONGO_OPERATOR_LT: endHeight}},
				{HEIGHT: UNCONFIRM_TX_HEIGHT},
			},
		}
	}
	err := this.Db.GetAll(this.TableName(), condition, selector, TXID, &txidBsons)
	return txidBsons, err
}

func (this *TxInfoRepository) DeleteMsgTx(txid string) error {
	condition := bson.M{
		TXID: txid,
	}
	return this.Db.DeleteAll(this.TableName(), condition)
}
