package main

import (
	"encoding/json"
	"flag"
	"io/ioutil"
	"net"
	"net/http"
	"time"

	"github.com/dotwallet/touchstone/conf"
	"github.com/dotwallet/touchstone/controller"
	"github.com/dotwallet/touchstone/interceptor"
	"github.com/dotwallet/touchstone/mapi"
	"github.com/dotwallet/touchstone/message"
	"github.com/dotwallet/touchstone/models"
	"github.com/dotwallet/touchstone/services"
	"github.com/golang/glog"
	"github.com/gorilla/mux"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

func StartP2pServer(p2pController *controller.P2pController, host string, allowPeers []*conf.PeerConfig) {
	p2pListener, err := net.Listen("tcp", host)
	if err != nil {
		glog.Infof("StartP2pServer Listen %s", err)
		glog.Flush()
		panic(err)
	}

	var opts []grpc.ServerOption

	authCredential := interceptor.NewServerAuthCredential(allowPeers)

	opts = append(opts, grpc.Creds(authCredential))
	authInterceptor := interceptor.NewAuthInterceptor(allowPeers)
	opts = append(opts, grpc.UnaryInterceptor(authInterceptor.Intercept))

	//need opt
	s := grpc.NewServer(opts...)
	message.RegisterP2PServer(s, p2pController)
	reflection.Register(s)
	err = s.Serve(p2pListener)
	if err != nil {
		glog.Infof("StartHttpServer Serve %s", err)
		glog.Flush()
		panic(err)
	}
}

func StartHttpServer(httpController *controller.HttpController, host string) {
	r := mux.NewRouter()
	r.HandleFunc("/v1/touchstone/sendrawtransaction", interceptor.Aspect(httpController.SendRawTransaction, &controller.SendRawTransactionReq{}))
	r.HandleFunc("/v1/touchstone/gettxinventory", interceptor.Aspect(httpController.GetTransactionInventory, &controller.GetTransactionInventoryReq{}))
	r.HandleFunc("/v1/touchstone/getaddrutxos", interceptor.Aspect(httpController.GetAddrUtxos, &controller.GetAddrUtxosReq{}))
	r.HandleFunc("/v1/touchstone/getaddrbalance", interceptor.Aspect(httpController.GetAddrBalance, &controller.GetAddrBalanceReq{}))
	r.HandleFunc("/v1/touchstone/getaddrinventorys", interceptor.Aspect(httpController.GetAddrInventorys, &controller.GetAddrInventorysReq{}))
	r.HandleFunc("/v1/touchstone/setaddrinfo", interceptor.Aspect(httpController.SetAddrInfo, &controller.SetAddrInfoReq{}))
	r.HandleFunc("/v1/touchstone/getuserutxos", interceptor.Aspect(httpController.GetUserUtxos, &controller.GetUserUtxosReq{}))
	r.HandleFunc("/v1/touchstone/getuserbalance", interceptor.Aspect(httpController.GetUserBalance, &controller.GetUserBalanceReq{}))
	r.HandleFunc("/v1/touchstone/getuserinventorys", interceptor.Aspect(httpController.GetUserInventorys, &controller.GetUserInventorysReq{}))
	r.HandleFunc("/v1/touchstone/sendbadgetoaddress", interceptor.Aspect(httpController.SendBadgeToAddress, &controller.SendBadgeToAddressReq{}))
	err := http.ListenAndServe(host, r)
	if err != nil {
		glog.Infof("StartHttpServer ListenAndServe %s", err)
		glog.Flush()
		panic(err)
	}
}

func main() {
	configFilePath := flag.String("config", "conf/config.json", "Path of config file")
	flag.Parse()
	configJSON, err := ioutil.ReadFile(*configFilePath)
	if err != nil {
		glog.Infof("main 1 ReadFile %s", err)
		glog.Flush()
		panic(err)
	}
	config := &conf.Config{}
	err = json.Unmarshal(configJSON, config)
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

	touchstoneServer := &services.TouchstoneServer{
		TxInfoRepository:                 txInfoRepository,
		TxPointRepository:                txPointRepository,
		PartitionInfoRepository:          partitionInfoRepository,
		MapiClient:                       mapiClient,
		AddrInfoRepository:               addrInfoRepository,
		NeedRecomputehashPartitionsCache: make(map[int64]bool),
	}

	p2pController := &controller.P2pController{
		TouchstoneServer: touchstoneServer,
	}
	go StartP2pServer(p2pController, config.P2pHost, config.PeersConfigs)

	httpController := &controller.HttpController{
		TouchstoneServer: touchstoneServer,
	}

	go StartHttpServer(httpController, config.HttpHost)

	err = touchstoneServer.Init(config.PeersConfigs, config.ServerPrivatekey)
	if err != nil {
		glog.Infof("main 4 touchstoneServer Init %s", err)
		glog.Flush()
		panic(err)
	}

	for {
		glog.Infof("main %s", time.Now().String())
		time.Sleep(time.Hour)
	}

}
