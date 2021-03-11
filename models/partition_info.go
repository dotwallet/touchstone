package models

import (
	"gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"
)

const (
	TBL_PARTITION_INFO = "partition_info"
)

type PartitionInfo struct {
	Id   int64  `bson:"id"`
	Hash string `bson:"hash"`
}

type PartitionInfoRepository struct {
	Db *MongoDb
}

func (this *PartitionInfoRepository) TableName() string {
	return TBL_PARTITION_INFO
}

func (this *PartitionInfoRepository) CreateIndex() error {
	return this.Db.CreateIndex(
		this.TableName(),
		[]*mgo.Index{
			{
				Key:    []string{ID},
				Unique: true,
			},
		},
	)
}

func (this *PartitionInfoRepository) GetPartitionInfo(Id int64) (*PartitionInfo, error) {
	partitionInfo := &PartitionInfo{}
	condition := bson.M{
		ID: Id,
	}
	err := this.Db.GetOne(this.TableName(), condition, nil, partitionInfo)
	return partitionInfo, err
}

func (this *PartitionInfoRepository) GetPartitionInfos(offset int, limit int) ([]*PartitionInfo, error) {
	partitionInfos := make([]*PartitionInfo, 0, 1024)
	err := this.Db.GetMany(this.TableName(), nil, nil, ID, offset, limit, &partitionInfos)
	return partitionInfos, err
}

func (this *PartitionInfoRepository) AddPartitionInfo(id int64, hash string) error {
	partitionInfo := &PartitionInfo{
		Id:   id,
		Hash: hash,
	}
	return this.Db.Insert(this.TableName(), partitionInfo)
}

func (this *PartitionInfoRepository) UpdatePartitionInfo(id int64, hash string) error {
	condition := bson.M{
		ID: id,
	}
	updator := bson.M{
		HASH: hash,
	}
	return this.Db.UpdateAll(this.TableName(), condition, updator)
}

func (this *PartitionInfoRepository) GetPartitionsCount() (int64, error) {
	return this.Db.Count(this.TableName(), nil)
}
