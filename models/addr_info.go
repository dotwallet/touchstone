package models

import (
	"strings"

	"gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"
)

const (
	TBL_ADDR_INFO = "addr_info"
)

type AddrInfo struct {
	Appid     string `json:"appid" bson:"appid"`
	UserID    int64  `json:"userid" bson:"userid"`
	UserIndex int64  `json:"user_index" bson:"user_index"`
	Addr      string `json:"addr" bson:"addr"`
	Timestamp int64  `json:"timestamp" bson:"timestamp"`
}

type AddrInfoRepository struct {
	Db *MongoDb
}

func (*AddrInfoRepository) TableName() string {
	return TBL_ADDR_INFO
}

func (this *AddrInfoRepository) CreateIndex() error {
	return this.Db.CreateIndex(
		this.TableName(),
		[]*mgo.Index{
			{
				Key:    []string{ADDR},
				Unique: true,
			},
			{
				Key:    []string{APPID, USERID, USER_INDEX},
				Unique: false,
			},
		},
	)
}

func (this *AddrInfoRepository) AddOrUpdateAddrInfo(addrInfo *AddrInfo) error {
	err := this.Db.Insert(this.TableName(), addrInfo)
	if err != nil {
		if !strings.Contains(err.Error(), MONGO_ERROR_DUPLICATE) {
			return err
		}
		condition := bson.M{
			ADDR: addrInfo.Addr,
		}
		updator := bson.M{
			APPID:      addrInfo.Appid,
			USERID:     addrInfo.UserID,
			USER_INDEX: addrInfo.UserIndex,
			TIMESTAMP:  addrInfo.Timestamp,
		}
		return this.Db.UpdateAll(this.TableName(), condition, updator)
	}
	return nil
}

func (this *AddrInfoRepository) GetAddrInfo(addr string) (*AddrInfo, error) {
	condition := bson.M{
		ADDR: addr,
	}
	addrInfo := &AddrInfo{}
	err := this.Db.GetOne(this.TableName(), condition, nil, addrInfo)
	return addrInfo, err
}

type TxPointsBson struct {
	TxPoint []*TxPoint
}

// this code will do something like this
// db.addr_info.aggregate([
//     {
//         "$match":{"appid":"","userid":1,"user_index":0,"badge_code":""}
//     },
//     {
//         "$lookup":{
//             "from":"tx_point",
//             "localField":"addr",
//             "foreignField":"addr",
//             "as":"tx_point"
//         }
//     },
//     {
//         "$unwind":"$tx_point"
//     },
//     {
//         "$match":{"tx_point.state":1}
//     },
//     {
//         "$project":{
//             "addr":1,
//             "txid":"$tx_point.txid",
//             "index":"$tx_point.index",
//             "type":"$tx_point.type",
//             "value":"$tx_point.value",
//             "pretxid":"$tx_point.pretxid",
//             "preindex":"$tx_point.preindex",
//             "badge_code":"$tx_point.badge_code",
//             "timestamp":"$tx_point.timestamp",
//             "state":"$tx_point.state"
//         }
//     }
// ])
func (this *AddrInfoRepository) GetUserTxPoints(appid string, userid int64, userIndex int64, badgeCode string, state int) ([]*TxPoint, error) {
	conditions := []bson.M{
		{
			MONGO_OPERATOR_MATCH: bson.M{APPID: appid, USERID: userid, USER_INDEX: userIndex},
		},
		{
			MONGO_OPERATOR_LOOKUP: bson.M{
				MONGO_OPERATOR_FROM:          TBL_TX_POINT,
				MONGO_OPERATOR_LOACL_FIELD:   ADDR,
				MONGO_OPERATOR_FOREIGN_FIELD: ADDR,
				MONGO_OPERATOR_AS:            "tx_point",
			},
		},
		{
			MONGO_OPERATOR_UNWIND: "$tx_point",
		},
	}
	secondMatchCondition := bson.M{
		"tx_point.badge_code": badgeCode,
	}
	if state != TX_POINT_STATE_ALL {
		secondMatchCondition["tx_point.state"] = state
	}
	conditions = append(conditions, bson.M{
		MONGO_OPERATOR_MATCH: secondMatchCondition,
	})
	conditions = append(conditions,
		bson.M{
			MONGO_OPERATOR_PROJECT: bson.M{
				ADDR:       "$tx_point.addr",
				TXID:       "$tx_point.txid",
				INDEX:      "$tx_point.index",
				TYPE:       "$tx_point.type",
				VALUE:      "$tx_point.value",
				PRETXID:    "$tx_point.pretxid",
				PREINDEX:   "$tx_point.preindex",
				BADGE_CODE: "$tx_point.badge_code",
				TIMESTAMP:  "$tx_point.timestamp",
				STATE:      "$tx_point.state",
			},
		},
	)
	result := make([]*TxPoint, 0, 8)
	err := this.Db.AggregateAll(this.TableName(), conditions, &result)
	return result, err

}
