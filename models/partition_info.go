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

func (this *PartitionInfoRepository) CreateIndex() error {
	return this.Db.CreateIndex(
		TBL_PARTITION_INFO,
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
	err := this.Db.GetOne(TBL_PARTITION_INFO, condition, nil, partitionInfo)
	return partitionInfo, err
}

func (this *PartitionInfoRepository) GetPartitionInfos(offset int, limit int) ([]*PartitionInfo, error) {
	partitionInfos := make([]*PartitionInfo, 0, 1024)
	err := this.Db.GetMany(TBL_PARTITION_INFO, nil, nil, ID, offset, limit, &partitionInfos)
	return partitionInfos, err
}

func (this *PartitionInfoRepository) AddPartitionInfo(id int64, hash string) error {
	partitionInfo := &PartitionInfo{
		Id:   id,
		Hash: hash,
	}
	return this.Db.Insert(TBL_PARTITION_INFO, partitionInfo)
}

func (this *PartitionInfoRepository) UpdatePartitionInfo(id int64, hash string) error {
	condition := bson.M{
		ID: id,
	}
	updator := bson.M{
		HASH: hash,
	}
	return this.Db.UpdateAll(TBL_PARTITION_INFO, condition, updator)
}

func (this *PartitionInfoRepository) GetPartitionsCount() (int64, error) {
	return this.Db.Count(TBL_PARTITION_INFO, nil)
}
