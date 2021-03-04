package models

import (
	"errors"

	"gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"
)

const (
	TBL_TX_POINT = "tx_point"

	TX_POINT_TYPE_ALL  = 0
	TX_POINT_TYPE_VIN  = 1
	TX_POINT_TYPE_VOUT = 2
)

type TxPoint struct {
	Addr      string `json:"addr" bson:"addr"`
	Txid      string `json:"-" bson:"txid"`
	Index     int    `json:"index" bson:"index"`
	Type      int    `json:"-" bson:"type"`
	Value     int64  `json:"value" bson:"value"`
	PreTxid   string `json:"pretxid,omitempty" bson:"pretxid,omitempty"`
	PreIndex  int    `json:"preindex,omitempty" bson:"preindex,omitempty"`
	BadgeCode string `json:"badge_code" bson:"badge_code"`
	Timestamp int64  `json:"timestamp" bson:"timestamp"`
}

type TxPointRepository struct {
	Db *MongoDb
}

func (this *TxPointRepository) CreateIndex() error {
	return this.Db.CreateIndex(
		TBL_TX_POINT,
		[]*mgo.Index{
			{
				Key:    []string{TXID, INDEX, TYPE},
				Unique: true,
			},
			{
				Key:    []string{ADDR},
				Unique: false,
			},
		},
	)
}

func (this *TxPointRepository) AddTxPoint(txPoint *TxPoint) error {
	return this.Db.Insert(TBL_TX_POINT, txPoint)
}

func (this *TxPointRepository) GetTxPoint(txid string, index int, Type int) (*TxPoint, error) {
	if Type == TX_POINT_TYPE_ALL {
		return nil, errors.New("only support in or out,not all")
	}
	condition := bson.M{
		TXID:  txid,
		INDEX: index,
		TYPE:  Type,
	}
	txPoint := &TxPoint{}
	err := this.Db.GetOne(TBL_TX_POINT, condition, nil, txPoint)
	return txPoint, err
}

func (this *TxPointRepository) GetTxPoints(txid string) ([]*TxPoint, error) {

	condition := bson.M{
		TXID: txid,
	}
	txPoints := make([]*TxPoint, 0, 8)
	err := this.Db.GetAll(TBL_TX_POINT, condition, nil, INDEX, &txPoints)
	return txPoints, err
}

func (this *TxPointRepository) DeleteTxPoints(txid string) error {
	condition := bson.M{
		TXID: txid,
	}
	return this.Db.DeleteAll(TBL_TX_POINT, condition)
}
