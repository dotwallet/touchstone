package models

import (
	"errors"
	"fmt"

	"github.com/dotwallet/touchstone/util"
	"gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"
)

const (
	MAX_RECONNECT_TIME = 5

	MONGO_ERROR_DUPLICATE = "E11000"
	MONGO_NOT_FOUND       = "not found"

	MONGO_ID                     = "_id"
	MONGO_OPERATOR_SET           = "$set"
	MONGO_OPERATOR_GTE           = "$gte"
	MONGO_OPERATOR_LT            = "$lt"
	MONGO_OPERATOR_OR            = "$or"
	MONGO_OPERATOR_MATCH         = "$match"
	MONGO_OPERATOR_PROJECT       = "$project"
	MONGO_OPERATOR_UNWIND        = "$unwind"
	MONGO_OPERATOR_LOOKUP        = "$lookup"
	MONGO_OPERATOR_FROM          = "from"
	MONGO_OPERATOR_LOACL_FIELD   = "localField"
	MONGO_OPERATOR_FOREIGN_FIELD = "foreignField"
	MONGO_OPERATOR_AS            = "as"
)

type MongoDb struct {
	sess   *mgo.Session
	host   string
	dbname string
}

func NewMongoDb(host string, dbname string) (*MongoDb, error) {
	sess, err := mgo.Dial(host)
	if err != nil {
		return nil, err
	}
	return &MongoDb{
		sess:   sess,
		host:   host,
		dbname: dbname,
	}, nil
}

func (this *MongoDb) NewSession() (*mgo.Session, error) {
	for i := 0; i < MAX_RECONNECT_TIME; i++ {
		sess := this.sess.Clone()
		err := sess.Ping()
		if err == nil {
			return sess, nil
		}
		sess, err = mgo.Dial(this.host)
		if err != nil {
			continue
		}
		this.sess = sess
	}
	errStr := fmt.Sprintf("reconnect %s %d failed", this.host, MAX_RECONNECT_TIME)
	return nil, errors.New(errStr)
}

func (this *MongoDb) Exec(colName string, opreation func(*mgo.Collection) error) error {
	sess, err := this.NewSession()
	if err != nil {
		return err
	}
	defer sess.Close()
	col := sess.DB(this.dbname).C(colName)
	return opreation(col)
}

func (this *MongoDb) Distinct(colName string, condition bson.M, key string, result interface{}) error {
	operation := func(col *mgo.Collection) error {
		return col.Find(condition).Distinct(key, result)
	}
	return this.Exec(colName, operation)
}

func (this *MongoDb) GetOne(colName string, condition bson.M, selector bson.M, result interface{}) error {
	operation := func(col *mgo.Collection) error {
		if selector != nil {
			return col.Find(condition).Select(selector).One(result)
		}
		return col.Find(condition).One(result)
	}
	return this.Exec(colName, operation)
}

func (this *MongoDb) GetMany(colName string, condition bson.M, selector bson.M, sort string, offset int, limit int, result interface{}) error {
	operation := func(col *mgo.Collection) error {
		if selector != nil {
			return col.Find(condition).Select(selector).Sort(sort).Skip(offset).Limit(limit).All(result)
		}
		return col.Find(condition).Sort(sort).Skip(offset).Limit(limit).All(result)
	}
	return this.Exec(colName, operation)
}

func (this *MongoDb) GetAll(colName string, condition bson.M, selector bson.M, sort string, result interface{}) error {
	operation := func(col *mgo.Collection) error {
		if selector != nil {
			return col.Find(condition).Select(selector).Sort(sort).All(result)
		}
		return col.Find(condition).Sort(sort).All(result)
	}
	return this.Exec(colName, operation)
}

func (this *MongoDb) Insert(colName string, data interface{}) error {
	operation := func(col *mgo.Collection) error {
		return col.Insert(data)
	}
	return this.Exec(colName, operation)
}

func (this *MongoDb) Count(colName string, condition bson.M) (int64, error) {
	var count = int64(-1)
	operation := func(col *mgo.Collection) error {
		countTmp, err := col.Find(condition).Count()
		if err != nil {
			return err
		}
		count = int64(countTmp)
		return nil
	}
	err := this.Exec(colName, operation)
	return count, err
}

func (this *MongoDb) UpdateAll(colName string, condition bson.M, updator bson.M) error {
	operation := func(col *mgo.Collection) error {
		setUpdator := bson.M{
			MONGO_OPERATOR_SET: updator,
		}
		_, err := col.UpdateAll(condition, setUpdator)
		return err
	}
	err := this.Exec(colName, operation)
	return err
}

func (this *MongoDb) UpdateOne(colName string, condition bson.M, updator bson.M) error {
	operation := func(col *mgo.Collection) error {
		setUpdator := bson.M{
			MONGO_OPERATOR_SET: updator,
		}
		return col.Update(condition, setUpdator)
	}
	return this.Exec(colName, operation)
}

func (this *MongoDb) CreateIndex(colName string, Indexs []*mgo.Index) error {
	opreation := func(col *mgo.Collection) error {
		for _, index := range Indexs {
			err := col.EnsureIndex(*index)
			if err != nil {
				return err
			}
		}
		return nil
	}
	return this.Exec(colName, opreation)
}

func (this *MongoDb) DeleteAll(colName string, condition bson.M) error {
	operation := func(col *mgo.Collection) error {
		_, err := col.RemoveAll(condition)
		return err
	}
	return this.Exec(colName, operation)
}

func (this *MongoDb) AggregateAll(colName string, conditions []bson.M, result interface{}) error {
	operation := func(col *mgo.Collection) error {
		err := col.Pipe(conditions).All(result)
		return err
	}
	return this.Exec(colName, operation)
}

func (this *MongoDb) AggregateOne(colName string, conditions []bson.M, result interface{}) error {
	operation := func(col *mgo.Collection) error {
		err := col.Pipe(conditions).One(result)
		return err
	}
	return this.Exec(colName, operation)
}

func (this *MongoDb) Foreach(colName string, conditions bson.M, result interface{}, handle func() error) error {
	operation := func(col *mgo.Collection) error {
		fmt.Println(util.ToJson(conditions))
		iter := col.Find(conditions).Iter()
		for iter.Next(result) {
			err := handle()
			if err != nil {
				return err
			}
		}
		return iter.Err()
	}
	return this.Exec(colName, operation)
}
