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

	TX_POINT_STATE_ALL               = 0
	TX_POINT_STATE_MAY_BE_UNSPENT    = 1
	TX_POINT_STATE_PRETTY_SURE_SPENT = 2
)

type TxPoint struct {
	Addr      string `json:"addr" bson:"addr"`
	Txid      string `json:"txid" bson:"txid"`
	Index     int    `json:"index" bson:"index"`
	Type      int    `json:"-" bson:"type"`
	Value     int64  `json:"value" bson:"value"`
	PreTxid   string `json:"pretxid" bson:"pretxid"`
	PreIndex  int    `json:"preindex" bson:"preindex"`
	BadgeCode string `json:"badge_code" bson:"badge_code"`
	Timestamp int64  `json:"timestamp" bson:"timestamp"`
	State     int    `json:"-" bson:"state"`
}

type TxPointRepository struct {
	Db *MongoDb
}

func (this *TxPointRepository) TableName() string {
	return TBL_TX_POINT
}

func (this *TxPointRepository) CreateIndex() error {
	return this.Db.CreateIndex(
		this.TableName(),
		[]*mgo.Index{
			{
				Key:    []string{TXID, INDEX, TYPE},
				Unique: true,
			},
			{
				Key:    []string{ADDR},
				Unique: false,
			},
			{
				Key:    []string{STATE},
				Unique: false,
			},
			{
				Key:    []string{TIMESTAMP},
				Unique: false,
			},
		},
	)
}

func (this *TxPointRepository) ForearchUnspentVinTxPoint(lastTimestamp int64, container interface{}, handle func() error) error {
	condition := bson.M{
		TIMESTAMP: bson.M{
			MONGO_OPERATOR_LT: lastTimestamp,
		},
		STATE: TX_POINT_STATE_MAY_BE_UNSPENT,
		TYPE:  TX_POINT_TYPE_VIN,
	}
	return this.Db.Foreach(this.TableName(), condition, container, handle)
}

func (this *TxPointRepository) AddTxPoint(txPoint *TxPoint) error {
	return this.Db.Insert(this.TableName(), txPoint)
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
	err := this.Db.GetOne(this.TableName(), condition, nil, txPoint)
	return txPoint, err
}

func (this *TxPointRepository) GetTxPoints(txid string) ([]*TxPoint, error) {
	condition := bson.M{
		TXID: txid,
	}
	txPoints := make([]*TxPoint, 0, 8)
	err := this.Db.GetAll(this.TableName(), condition, nil, INDEX, &txPoints)
	return txPoints, err
}

func (this *TxPointRepository) GetTxPointsByAddr(addr string, badgeCode string, state int) ([]*TxPoint, error) {
	condition := bson.M{
		ADDR:       addr,
		BADGE_CODE: badgeCode,
	}
	if state != TX_POINT_STATE_ALL {
		condition[STATE] = state
	}
	txPoints := make([]*TxPoint, 0, 8)
	err := this.Db.GetAll(this.TableName(), condition, nil, "-"+TIMESTAMP, &txPoints)
	return txPoints, err
}

func (this *TxPointRepository) DeleteTxPoints(txid string) error {
	condition := bson.M{
		TXID: txid,
	}
	return this.Db.DeleteAll(this.TableName(), condition)
}

func (this *TxPointRepository) SetTxPointState(txid string, index int, Type int, state int) error {
	if Type == TX_POINT_TYPE_ALL {
		return errors.New("only support in or out,not all")
	}
	conditions := bson.M{
		TXID:  txid,
		INDEX: index,
		TYPE:  Type,
	}
	updator := bson.M{
		STATE: state,
	}
	return this.Db.UpdateOne(this.TableName(), conditions, updator)
}
