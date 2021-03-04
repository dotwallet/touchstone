package models

import (
	"time"

	"gopkg.in/mgo.v2"
)

const (
	TXID    = "txid"
	PRETXID = "pretxid"
	HASH    = "hash"
	INDEX   = "index"
	ID      = "id"
	HEIGHT  = "height"
	TYPE    = "type"
	STATE   = "state"
	ADDR    = "addr"
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
