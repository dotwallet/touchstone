package models

import (
	"time"

	"gopkg.in/mgo.v2"
)

const (
	TXID       = "txid"
	PRETXID    = "pretxid"
	PREINDEX   = "preindex"
	HASH       = "hash"
	INDEX      = "index"
	USER_INDEX = "user_index"
	ID         = "id"
	HEIGHT     = "height"
	TYPE       = "type"
	STATE      = "state"
	ADDR       = "addr"
	APPID      = "appid"
	USERID     = "userid"
	TIMESTAMP  = "timestamp"
	BADGE_CODE = "badge_code"
	VALUE      = "value"
)

func NewDb(host string, dbname string) (*MongoDb, error) {
	sess, err := mgo.Dial(host)
	if err != nil {
		return nil, err
	}
	sess.SetMode(mgo.Monotonic, true)
	sess.SetSocketTimeout(24 * time.Hour)
	sess.SetPoolLimit(1024)
	return &MongoDb{
		sess:   sess,
		host:   host,
		dbname: dbname,
	}, nil
}
