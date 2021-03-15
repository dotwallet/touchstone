package services

import (
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/dotwallet/touchstone/conf"
	"github.com/dotwallet/touchstone/mapi"
	"github.com/dotwallet/touchstone/models"
	"github.com/golang/glog"
)

// {
// 	"Host": "127.0.0.1:10002",
// 	"Pubkey": "036af584f4f274e3b6831f9c8cfb8cce56d441887a9349cc93b180eb9a913d06cd"
// }

const (
	confitStr = `{
		"Env": "mainnet",
		"MongoHost": "127.0.0.1:27037",
		"MempoolHost": "https://api.ddpurse.com",
		"MempoolPkiMnemonic": "earn economy machine gauge grass during gain pencil spread absent wall ugly",
		"MempoolPkiMnemonicPassword": "",
		"ServerPrivatekey": "1015e88bfaccd79c3f896e99cdf39cde76a8c36d311c4b0be8cc4ab47c5e6c48",
		"PeersConfigs": [
			{
				"Host": "127.0.0.1:10001",
				"Pubkey": "02ed510541e694b3a126292e8aba20e402c45e474275b4e5a6ef2ccc29949f6675"
			}
		],
		"P2pHost": "0.0.0.0:10003",
		"HttpHost": "0.0.0.0:20003",
		"DbName": "touchstone_3"
	}`
)

var glbTestTouchStoneServer *TouchstoneServer

var gstart int64
var gend int64

func TestMain(m *testing.M) {
	flag.Set("alsologtostderr", "true")
	flag.Set("log_dir", "/tmp")
	flag.Set("v", "3")
	start := flag.Int64("start", 0, "Path of config file")
	end := flag.Int64("end", 100, "Path of config file")
	flag.Parse()
	gstart = *start
	gend = *end
	ret := m.Run()
	os.Exit(ret)
}

func init() {
	configJSON := []byte(confitStr)
	config := &conf.Config{}
	err := json.Unmarshal(configJSON, config)
	if err != nil {
		glog.Infof("main 2 Unmarshal %s", err)
		glog.Flush()
		panic(err)
	}
	err = conf.InitGConfig(config.Env)
	if err != nil {
		glog.Infof("main 3 InitGConfig %s", err)
		glog.Flush()
		panic(err)
	}
	db, err := models.NewDb(config.MongoHost, config.DbName)
	if err != nil {
		glog.Infof("main 4 NewDb %s", err)
		glog.Flush()
		panic(err)
	}
	txInfoRepository := &models.TxInfoRepository{
		Db: db,
	}
	err = txInfoRepository.CreateIndex()
	if err != nil {
		glog.Infof("main 5 txInfoRepository CreateIndex %s", err)
		glog.Flush()
		panic(err)
	}

	txPointRepository := &models.TxPointRepository{
		Db: db,
	}
	err = txPointRepository.CreateIndex()
	if err != nil {
		glog.Infof("main 5 txPointRepository CreateIndex %s", err)
		glog.Flush()
		panic(err)
	}

	partitionInfoRepository := &models.PartitionInfoRepository{
		Db: db,
	}
	err = partitionInfoRepository.CreateIndex()
	if err != nil {
		glog.Infof("main 5 partitionInfoRepository CreateIndex %s", err)
		glog.Flush()
		panic(err)
	}

	addrInfoRepository := &models.AddrInfoRepository{
		Db: db,
	}
	err = addrInfoRepository.CreateIndex()
	if err != nil {
		glog.Infof("main 5 addrInfoRepository CreateIndex %s", err)
		glog.Flush()
		panic(err)
	}

	mapiClient, err := mapi.NewMempoolMapiClient(config.MempoolHost, config.MempoolPkiMnemonic, config.MempoolPkiMnemonicPassword)
	if err != nil {
		glog.Infof("main 6 NewMempoolMapiClient CreateIndex %s", err)
		glog.Flush()
		panic(err)
	}

	glbTestTouchStoneServer = &TouchstoneServer{
		TxInfoRepository:                 txInfoRepository,
		TxPointRepository:                txPointRepository,
		PartitionInfoRepository:          partitionInfoRepository,
		MapiClient:                       mapiClient,
		AddrInfoRepository:               addrInfoRepository,
		NeedRecomputehashPartitionsCache: make(map[int64]bool),
	}
	glbTestTouchStoneServer.SetPrivateKey(config.ServerPrivatekey)
	for i := 0; i < 10; i++ {
		err = glbTestTouchStoneServer.ConnectPeer(config.PeersConfigs)
		if err != nil {
			glog.Infof("SyncPatitions %d ConnectPeer err:%s", i, err)
			time.Sleep(time.Second)
			continue
		}
		break
	}
}

func testSyncState(t *testing.T) {
	for i := gstart; i < gend; i++ {
		err := glbTestTouchStoneServer.SyncPatitions(i, i+1, fmt.Sprintf("processId-%d", i))
		if err != nil {
			glog.Infof("SyncPatitions %d err %s", i, err)
			continue
		}
		glog.Infof("SyncPatitions %d done", i)
	}
}

func TestComputePartitionHash(t *testing.T) {
	hash, err := glbTestTouchStoneServer.ComputePartitionHash(1712)
	if err != nil {
		panic(err)
	}
	fmt.Println(hex.EncodeToString(hash))
}
