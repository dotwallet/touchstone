package services

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"g.mempool.com/base/touchstone/conf"
	"g.mempool.com/base/touchstone/interceptor"
	"g.mempool.com/base/touchstone/mapi"
	"g.mempool.com/base/touchstone/message"
	"g.mempool.com/base/touchstone/models"
	"g.mempool.com/base/touchstone/util"
	"github.com/btcsuite/btcd/btcec"
	"github.com/btcsuite/btcd/wire"
	"github.com/golang/glog"
	"google.golang.org/grpc"
)

type LocalSingleTxSource struct {
	txid    []byte
	txbytes []byte
}

func NewLocalSingleTxSource(rawTx string) (*LocalSingleTxSource, error) {
	msgTxByte, err := hex.DecodeString(rawTx)
	if err != nil {
		return nil, err
	}
	msgTx, err := util.DeserializeTxBytes(msgTxByte)
	if err != nil {
		return nil, err
	}
	hash := msgTx.TxHash()
	return &LocalSingleTxSource{
		txid:    util.GetHashByte(hash),
		txbytes: msgTxByte,
	}, nil
}

func (this *LocalSingleTxSource) GetTxBytes(txids [][]byte) ([][]byte, error) {
	if len(txids) == 0 {
		return nil, nil
	}
	if len(txids) > 1 {
		return nil, errors.New("only support single tx")
	}
	if !bytes.Equal(this.txid, txids[0]) {
		return nil, errors.New("tx not found")
	}
	result := [][]byte{
		this.txbytes,
	}
	return result, nil
}

type Node struct {
	message.P2PClient
}

func (this *Node) GetTxBytes(txids [][]byte) ([][]byte, error) {
	if len(txids) == 0 {
		return nil, nil
	}
	request := &message.GetTxsRequest{
		Txids: txids,
	}
	getTxsResponse, err := this.P2PClient.GetTxs(context.Background(), request)
	if err != nil {
		return nil, err
	}
	return getTxsResponse.Rawtxs, nil
}

type TouchstoneServer struct {
	peers                            map[string]*Node
	TxInfoRepository                 *models.TxInfoRepository
	PartitionInfoRepository          *models.PartitionInfoRepository
	MapiClient                       *mapi.MapiClient
	TxPointRepository                *models.TxPointRepository
	NeedRecomputehashPartitionsCache map[int64]bool
	cacheLock                        sync.Mutex
	syncTxLock                       sync.RWMutex
	privateKey                       *btcec.PrivateKey
}

func (this *TouchstoneServer) Peers() map[string]*Node {
	return this.peers
}

func (this *TouchstoneServer) AddNeedRecomputehashPartition(id int64) {
	this.cacheLock.Lock()
	defer this.cacheLock.Unlock()
	this.NeedRecomputehashPartitionsCache[id] = true
}

func (this *TouchstoneServer) AddNeedRecomputehashPartitionByHeight(height int64) {
	if height == -1 {
		return
	}
	patition := (height - *conf.GStartHeight) / conf.PARTITION_BLOCK_COUNT
	this.AddNeedRecomputehashPartition(patition)
	return
}

func (this *TouchstoneServer) ClearPartitionsCache() map[int64]bool {
	this.cacheLock.Lock()
	defer this.cacheLock.Unlock()
	oldCache := this.NeedRecomputehashPartitionsCache
	this.NeedRecomputehashPartitionsCache = make(map[int64]bool)
	return oldCache
}

func (this *TouchstoneServer) ClearCacheAndSetHash() error {
	//todo here may be should compute before clear,or it may compute fail
	cache := this.ClearPartitionsCache()
	for id := range cache {
		if id == -1 {
			continue
		}
		err := this.ComputeAndSetPartitionHash(id)
		if err != nil {
			glog.Infof("TouchstoneServer.ClearCacheAndSetHash ComputeAndSetPartitionHash")
			return err
		}
	}
	return nil
}

type TxInventory struct {
	Vins  []*models.TxPoint `json:"vins"`
	Vouts []*models.TxPoint `json:"vouts"`
}

func NewTxInventory() *TxInventory {
	return &TxInventory{
		Vins:  make([]*models.TxPoint, 0, 8),
		Vouts: make([]*models.TxPoint, 0, 8),
	}
}

func (this *TouchstoneServer) ParseMsgTx(MsgTx *wire.MsgTx, timestamp int64) (*TxInventory, error) {
	txInventory := NewTxInventory()
	badgeValues := make(map[string]int64)
	illegalVin := false
	for index, vin := range MsgTx.TxIn {
		//todo
		if !(len(vin.SignatureScript) >= len(util.BADGE_FLAG) && string(vin.SignatureScript[len(vin.SignatureScript)-len(util.BADGE_FLAG):]) == util.BADGE_FLAG) {
			continue
		}
		txPoint, err := this.TxPointRepository.GetTxPoint(vin.PreviousOutPoint.Hash.String(), int(vin.PreviousOutPoint.Index), models.TX_POINT_TYPE_VOUT)
		if err != nil {
			if !strings.Contains(err.Error(), models.MONGO_NOT_FOUND) {
				return nil, err
			}
			msgTxBriefInfo, err := this.TxInfoRepository.GetMsgTxBriefInfo(vin.PreviousOutPoint.Hash.String())
			if err != nil {
				if !strings.Contains(err.Error(), models.MONGO_NOT_FOUND) {
					return nil, err
				}
				errStr := fmt.Sprintf("unknow utxo %s", vin.PreviousOutPoint.String())
				return nil, errors.New(errStr)
			}
			if msgTxBriefInfo.State != models.TX_STATE_CLOSED {
				errStr := fmt.Sprintf("unknow utxo %s", vin.PreviousOutPoint.String())
				return nil, errors.New(errStr)
			}
			illegalVin = true
			continue
		}
		_, ok := badgeValues[txPoint.BadgeCode]
		if !ok {
			badgeValues[txPoint.BadgeCode] = 0
		}
		badgeValues[txPoint.BadgeCode] += txPoint.Value
		newTxPoint := &models.TxPoint{
			Addr:      txPoint.Addr,
			Txid:      MsgTx.TxHash().String(),
			Index:     index,
			Type:      models.TX_POINT_TYPE_VIN,
			Value:     -txPoint.Value,
			PreTxid:   txPoint.Txid,
			PreIndex:  txPoint.Index,
			BadgeCode: txPoint.BadgeCode,
			Timestamp: timestamp,
		}
		txInventory.Vins = append(txInventory.Vins, newTxPoint)
	}
	if illegalVin {
		return txInventory, nil
	}

	if len(badgeValues) > 1 {
		// cointain different badges
		return txInventory, nil
	}

	badgeCode := MsgTx.TxHash().String()
	badgeValue := int64(util.MAX_CREATE_BADGE_VALUE)
	for assetCodeTmp, valueTmp := range badgeValues {
		badgeCode = assetCodeTmp
		badgeValue = valueTmp
	}

	voutTxPoints := make([]*models.TxPoint, 0, 8)
	for index, vout := range MsgTx.TxOut {
		badgeVout, err := util.ParseBadgeVoutScript(vout.PkScript, conf.GNetParam)
		if err != nil {
			continue
		}
		badgeValue -= badgeVout.BadgeValue
		if badgeValue < 0 {
			//vin less than vout
			return txInventory, nil
		}
		newOutPoint := &models.TxPoint{
			Addr:      badgeVout.Address.String(),
			Txid:      MsgTx.TxHash().String(),
			Index:     index,
			Type:      models.TX_POINT_TYPE_VOUT,
			Value:     badgeVout.BadgeValue,
			BadgeCode: badgeCode,
			Timestamp: timestamp,
		}
		voutTxPoints = append(voutTxPoints, newOutPoint)
	}
	txInventory.Vouts = append(txInventory.Vouts, voutTxPoints...)
	return txInventory, nil
}

func (this *TouchstoneServer) ParseAndAddTxPoints(msgTx *wire.MsgTx, timestamp int64) (*TxInventory, error) {
	txInventory, err := this.ParseMsgTx(msgTx, timestamp)
	if err != nil {
		glog.Infof("TouchstoneServer.ParseAndAddTxPoints ParseMsgTx txid:%s err:%s", msgTx.TxHash().String(), err)
		return nil, err
	}
	for _, txPoint := range txInventory.Vins {
		err := this.TxPointRepository.AddTxPoint(txPoint)
		if err != nil {
			if !strings.Contains(err.Error(), models.MONGO_ERROR_DUPLICATE) {
				return nil, err
			}
		}
	}

	for _, txPoint := range txInventory.Vouts {
		err := this.TxPointRepository.AddTxPoint(txPoint)
		if err != nil {
			if !strings.Contains(err.Error(), models.MONGO_ERROR_DUPLICATE) {
				return nil, err
			}
		}
	}
	return txInventory, nil
}

func (this *TouchstoneServer) ProcessMsgTx(msgTx *wire.MsgTx, timestamp int64) (*TxInventory, error) {
	state := models.TX_STATE_CLOSED
	txInventory, err := this.ParseAndAddTxPoints(msgTx, timestamp)
	if err != nil {
		if !strings.Contains(err.Error(), "unknow utxo") {
			glog.Infof("TouchstoneServer.ProcessMsgTx ProcessMsgTx txid:%s err:%s", msgTx.TxHash().String(), err)
			return nil, err
		}
		state = models.TX_STATE_OPEN
	}
	{
		err := this.TxInfoRepository.SetMsgTxState(msgTx.TxHash().String(), state)
		if err != nil {
			glog.Infof("TouchstoneServer.ProcessMsgTx SetMsgTxState txid:%s state:%d err:%s", msgTx.TxHash().String(), state, err)
			return nil, err
		}
	}
	return txInventory, err
}

type TxidMsg struct {
	Txid string `json:"txid"`
	Msg  string `json:"msg"`
}

type ProcessMsgTxsResult struct {
	TxInventorys []*TxInventory `json:"tx_inventorys"`
	ErrTxs       []*TxidMsg     `json:"error_txs"`
}

func SortAndDistinctMsgTxs(msgTxs []*wire.MsgTx) []*wire.MsgTx {
	txs := make(map[string]*wire.MsgTx)
	for _, msgTx := range msgTxs {
		txs[msgTx.TxHash().String()] = msgTx
	}
	tmp := make([]*wire.MsgTx, 0, len(msgTxs))
	for _, msgTx := range msgTxs {
		tmp = append(tmp, msgTx)
		for _, vin := range msgTx.TxIn {
			preTx, ok := txs[vin.PreviousOutPoint.Hash.String()]
			if ok {
				tmp = append(tmp, preTx)
			}
		}
	}
	result := make([]*wire.MsgTx, 0, len(msgTxs))
	for i := len(tmp) - 1; i >= 0; i-- {
		msgTx, ok := txs[tmp[i].TxHash().String()]
		if ok {
			result = append(result, msgTx)
			delete(txs, msgTx.TxHash().String())
		}
	}
	return result
}

func (this *TouchstoneServer) ProcessMsgTxs(msgTxs []*wire.MsgTx, timestamp int64) *ProcessMsgTxsResult {
	msgTxs = SortAndDistinctMsgTxs(msgTxs)
	processMsgTxsResult := &ProcessMsgTxsResult{
		TxInventorys: make([]*TxInventory, 0, len(msgTxs)),
		ErrTxs:       make([]*TxidMsg, 0, 8),
	}
	for _, msgTx := range msgTxs {
		inventory, err := this.ProcessMsgTx(msgTx, timestamp)
		if err != nil {
			txidMsg := &TxidMsg{
				Txid: msgTx.TxHash().String(),
				Msg:  err.Error(),
			}
			processMsgTxsResult.ErrTxs = append(processMsgTxsResult.ErrTxs, txidMsg)
			continue
		}
		processMsgTxsResult.TxInventorys = append(processMsgTxsResult.TxInventorys, inventory)
	}
	return processMsgTxsResult
}

func (this *TouchstoneServer) NotifyTxs(notifyTxsRequest *message.NotifyTxsRequest) {
	if len(notifyTxsRequest.Txids) == 0 {
		return
	}
	for _, peer := range this.Peers() {
		go peer.NotifyTxs(context.Background(), notifyTxsRequest)
	}
}

type SyncTxsResult struct {
	TxInventorys     []*TxInventory
	ErrTxs           []*TxidMsg
	AlreadyClosedTxs []string
}

type TxSource interface {
	GetTxBytes([][]byte) ([][]byte, error)
}

func (this *TouchstoneServer) SyncTxs(txidBytes [][]byte, txSource TxSource) (*SyncTxsResult, error) {
	lackTxids := make([][]byte, 0, 8)
	txStatesCache := make(map[string]*mapi.TxState)
	needProcessTx := make([]*wire.MsgTx, 0, 8)
	alreadyClosedTxs := make([]string, 0, 8)
	for _, txidByte := range txidBytes {
		txid := hex.EncodeToString(txidByte)
		txBriefInfo, err := this.TxInfoRepository.GetMsgTxBriefInfo(txid)
		if err != nil {
			if !strings.Contains(err.Error(), models.MONGO_NOT_FOUND) {
				glog.Infof("TouchstoneServer.SyncTxs GetMsgTxBriefInfo err:%s", err)
				return nil, err
			}
			txState, err := this.MapiClient.GetTxState(txid)
			if err != nil {
				glog.Infof("TouchstoneServer.SyncTxs GetTxState err:%s", err)
				continue
			}
			if txState.Payload.ReturnResult != mapi.RETURN_RESULT_FAILURE {
				txStatesCache[txid] = txState
				lackTxids = append(lackTxids, txidByte)
			}
			continue
		}
		if txBriefInfo.State != models.TX_STATE_CLOSED {
			msgTxInfo, err := this.TxInfoRepository.GetMsgTxInfo(txid)
			if err != nil {
				glog.Infof("TouchstoneServer.SyncTxs GetMsgTxInfo err:%s", err)
				return nil, err
			}
			this.AddNeedRecomputehashPartitionByHeight(txBriefInfo.Height)
			needProcessTx = append(needProcessTx, msgTxInfo.MsgTx)
		}
		alreadyClosedTxs = append(alreadyClosedTxs, txid)
	}
	txsbytes, err := txSource.GetTxBytes(lackTxids)
	if err != nil {
		return nil, err
	}

	notifyTxsRequest := &message.NotifyTxsRequest{
		Txids: make([][]byte, 0, len(txsbytes)),
	}
	this.syncTxLock.RLock()
	defer this.syncTxLock.RUnlock()
	for _, txBytes := range txsbytes {
		msgTx, err := util.DeserializeTxBytes(txBytes)
		if err != nil {
			glog.Infof("TouchstoneServer.SyncTxs DeserializeTxBytes %s err:%s", err)
			continue
		}
		txState, ok := txStatesCache[msgTx.TxHash().String()]
		if !ok {
			glog.Infof("TouchstoneServer.SyncTxs txStatesCache[msgTx.TxHash().String()] not found %s", msgTx.TxHash().String())
			continue
		}
		height := int64(models.UNCONFIRM_TX_HEIGHT)
		if txState.Payload.BlockHash != "" {
			height = txState.Payload.BlockHeight
		}
		err = this.TxInfoRepository.AddMsgTxInfo(msgTx, height, txState.Payload.BlockHash, time.Now().Unix())
		if err != nil {
			glog.Infof("TouchstoneServer.SyncTxs AddMsgTxInfo err:%s %s", err, msgTx.TxHash().String())
			return nil, err
		}
		this.AddNeedRecomputehashPartitionByHeight(txState.Payload.BlockHeight)
		needProcessTx = append(needProcessTx, msgTx)
		txhash := msgTx.TxHash()
		notifyTxsRequest.Txids = append(notifyTxsRequest.Txids, util.GetHashByte(txhash))
	}
	processMsgTxsResult := this.ProcessMsgTxs(needProcessTx, time.Now().Unix())
	this.NotifyTxs(notifyTxsRequest)
	return &SyncTxsResult{
		AlreadyClosedTxs: alreadyClosedTxs,
		ErrTxs:           processMsgTxsResult.ErrTxs,
		TxInventorys:     processMsgTxsResult.TxInventorys,
	}, nil
}

func (this *TouchstoneServer) ComputePartitionHash(id int64) ([]byte, error) {
	startHeight := *conf.GStartHeight + id*conf.PARTITION_BLOCK_COUNT
	txPoints, err := this.TxInfoRepository.GetTxidsByHeightRangeOrderByTxid(startHeight, startHeight+conf.PARTITION_BLOCK_COUNT, models.TX_STATE_CLOSED, false)
	if err != nil {
		glog.Infof("TouchstoneServer.ComputePartitionHash GetClosedTxidsByHeightRange %d err:%s", id, err)
		return nil, err
	}
	hashComputer := sha256.New()
	for _, txPoint := range txPoints {
		txid, err := hex.DecodeString(txPoint.Txid)
		if err != nil {
			//todo here may not return err
			return nil, err
		}
		_, err = hashComputer.Write(txid)
		if err != nil {
			return nil, err
		}
	}
	return hashComputer.Sum(nil), nil
}

func (this *TouchstoneServer) ComputeAndSetPartitionHash(id int64) error {
	hash, err := this.ComputePartitionHash(id)
	if err != nil {
		glog.Infof("TouchstoneServer.ComputeAndSetPartitionHash ComputePartitionHash id:%d err:%s", id, err)
		return err
	}
	hashStr := hex.EncodeToString(hash)
	err = this.PartitionInfoRepository.AddPartitionInfo(id, hashStr)
	if err != nil {
		if !strings.Contains(err.Error(), models.MONGO_ERROR_DUPLICATE) {
			glog.Infof("TouchstoneServer.ComputeAndSetPartitionHash AddPartitionInfo id:%d err:%s", id, err)
			return err
		}
	}
	return this.PartitionInfoRepository.UpdatePartitionInfo(id, hashStr)
}

func (this *TouchstoneServer) SyncPatitions(start int64, limit int64) error {
	getPartitionsHashRequest := &message.GetPartitionsHashRequest{
		Offset: start,
		Limit:  limit,
	}
	err := this.ClearCacheAndSetHash()
	if err != nil {
		glog.Infof("TouchstoneServer.SyncState ClearCacheAndSetHash err:%s", err)
		return err
	}
	for pubkey, peer := range this.Peers() {
		getPartitionsHashResponse, err := peer.GetPartitionsHash(context.Background(), getPartitionsHashRequest)
		if err != nil {
			glog.Infof("TouchstoneServer.SyncState GetPartitionsHash %s err:%s", pubkey, err)
			continue
		}
		partitionInfos, err := this.PartitionInfoRepository.GetPartitionInfos(int(start), int(limit))
		if err != nil {
			glog.Infof("TouchstoneServer.SyncState GetPartitionInfos err:%s", err)
			return err
		}
		getTxidsByPartitionsRequest := &message.GetTxidsByPartitionsRequest{
			Ids: make([]int64, 0, 8),
		}
		for index, partitionInfo := range partitionInfos {
			if index >= len(getPartitionsHashResponse.Hashs) {
				break
			}
			hash, err := hex.DecodeString(partitionInfo.Hash)
			if err != nil {
				//todo maybe
				glog.Infof("TouchstoneServer.SyncState DecodeString %d %s err:%s", partitionInfo.Id, partitionInfo.Hash, err)
				continue
			}
			if bytes.Equal(hash, getPartitionsHashResponse.Hashs[index]) {
				continue
			}
			getTxidsByPartitionsRequest.Ids = append(getTxidsByPartitionsRequest.Ids, partitionInfo.Id)
		}
		if len(getTxidsByPartitionsRequest.Ids) == 0 {
			continue
		}
		getTxidsResponse, err := peer.GetTxidsByPartitions(context.Background(), getTxidsByPartitionsRequest)
		if err != nil {
			glog.Infof("TouchstoneServer.SyncState GetTxidsByPartitions %s err:%s", pubkey, err)
			continue
		}
		_, err = this.SyncTxs(getTxidsResponse.Txids, peer)
		if err != nil {
			glog.Infof("TouchstoneServer.SyncState SyncTxs %s err:%s", pubkey, err)
			continue
		}
		err = this.ClearCacheAndSetHash()
		if err != nil {
			glog.Infof("TouchstoneServer.SyncState ClearCacheAndSetHash err:%s", err)
			return err
		}
	}
	return nil
}

func (this *TouchstoneServer) SyncUnconfirmTx() {
	getUnconfirmTxidsRequest := &message.GetUnconfirmTxidsRequest{}
	for pubkey, peer := range this.Peers() {
		getTxidsResponse, err := peer.GetUnconfirmTxids(context.Background(), getUnconfirmTxidsRequest)
		if err != nil {
			glog.Infof("TouchstoneServer.SyncUnconfirmTx GetTxidsByHeights %s err:%s", pubkey, err)
			continue
		}
		_, err = this.SyncTxs(getTxidsResponse.Txids, peer)
		if err != nil {
			glog.Infof("TouchstoneServer.SyncUnconfirmTx SyncTxs %s err:%s", pubkey, err)
			continue
		}
	}
}

func (this *TouchstoneServer) SyncState(syncAll bool, processid string) error {
	feeQuote, err := this.MapiClient.GetFeeQuote()
	if err != nil {
		glog.Infof("TouchstoneServer.SyncState GetFeeQuote %s", err)
		return err
	}
	count, err := this.PartitionInfoRepository.GetPartitionsCount()
	if err != nil {
		glog.Infof("TouchstoneServer.SyncState GetPartitionsCount %s", err)
		return err
	}
	this.SyncUnconfirmTx()

	expectPartitionsCount := (feeQuote.Payload.CurrentHighestBlockHeight-*conf.GStartHeight)/conf.PARTITION_BLOCK_COUNT + 1
	start := count - conf.RE_CONPUTE_PARTITION_COUNT
	if start < 0 || syncAll {
		start = 0
	}
	glog.Infof("TouchstoneServer.SyncState current height %d %t %d processid:%s", feeQuote.Payload.CurrentHighestBlockHeight, syncAll, start, processid)
	for i := start; i < expectPartitionsCount; i++ {
		this.AddNeedRecomputehashPartition(i)
	}
	return this.SyncPatitions(start, expectPartitionsCount)
}

func (this *TouchstoneServer) ConnectPeer(peerConfigs []*conf.PeerConfig) error {
	var opts []grpc.DialOption
	authPerRPCCredential := interceptor.NewAuthPerRPCCredential(this.privateKey)
	opts = append(opts, grpc.WithPerRPCCredentials(authPerRPCCredential))
	authCredential := interceptor.NewClientAuthCredential(this.privateKey)
	opts = append(opts, grpc.WithTransportCredentials(authCredential))
	oldPeers := this.Peers()
	peers := make(map[string]*Node)
	for _, peerConfig := range peerConfigs {
		peer, ok := oldPeers[peerConfig.Pubkey]
		if ok {
			peers[peerConfig.Pubkey] = peer
			continue
		}
		conn, err := grpc.Dial(peerConfig.Host, opts...)
		if err != nil {
			continue
		}
		c := message.NewP2PClient(conn)
		peers[peerConfig.Pubkey] = &Node{
			P2PClient: c,
		}
	}
	this.peers = peers
	if len(peers) == len(peerConfigs) {
		return nil
	}
	return errors.New("still have connect failed")
}

func (this *TouchstoneServer) ConnectPeerLoop(peerConfigs []*conf.PeerConfig) {
	for {
		processId := util.RandStringBytes(8)
		glog.Infof("CheckTxStateLoop ConnectPeerLoop start %s", processId)
		err := this.ConnectPeer(peerConfigs)
		if err == nil {
			glog.Infof("CheckTxStateLoop ConnectPeerLoop total done %s", processId)
			return
		}
		glog.Infof("CheckTxStateLoop ConnectPeerLoop done %s", processId)
		time.Sleep(time.Minute)
	}
}

func (this *TouchstoneServer) SyncStateLoop() {
	count := 0
	syncAll := false
	for {
		processId := util.RandStringBytes(8)
		glog.Infof("TouchstoneServer SyncStateLoop start %s", processId)
		//every hour sync all state once
		if count%12 == 0 {
			syncAll = true
		}
		err := this.SyncState(syncAll, processId)
		if err != nil {
			glog.Infof("TouchstoneServer.SyncStateLoop SyncState err:%s", err)
		}
		syncAll = false
		count++
		glog.Infof("TouchstoneServer SyncStateLoop done %s", processId)
		time.Sleep(time.Minute * 5)
	}
}

func (this *TouchstoneServer) NotifiedTxs(request *message.NotifyTxsRequest, peerPubkey string) {
	peer, ok := this.Peers()[peerPubkey]
	if !ok {
		glog.Infof("TouchstoneServer.NotifiedTxs err:peer not found %s", peerPubkey)
		return
	}
	_, err := this.SyncTxs(request.Txids, peer)
	if err != nil {
		glog.Infof("TouchstoneServer.NotifiedTxs SyncTxs %s err:%s", peerPubkey, err)
		return
	}
}

func (this *TouchstoneServer) GetTxs(request *message.GetTxsRequest) (*message.GetTxsResponse, error) {
	getTxsResponse := &message.GetTxsResponse{
		Rawtxs: make([][]byte, 0, len(request.Txids)),
	}
	for _, txidBytes := range request.Txids {
		txid := hex.EncodeToString(txidBytes)
		msgTxInfo, err := this.TxInfoRepository.GetMsgTxInfo(txid)
		if err != nil {
			if !strings.Contains(err.Error(), models.MONGO_NOT_FOUND) {
				glog.Infof("TouchstoneServer.GetTxs GetMsgTxInfo %s err:%s", txid, err)
				return nil, err
			}
			continue
		}
		msgTxBytes := util.SeserializeMsgTxBytes(msgTxInfo.MsgTx)
		getTxsResponse.Rawtxs = append(getTxsResponse.Rawtxs, msgTxBytes)
	}
	return getTxsResponse, nil
}

func (this *TouchstoneServer) ClearMsgTx(txid string) error {
	this.syncTxLock.Lock()
	defer this.syncTxLock.Unlock()
	err := this.TxPointRepository.DeleteTxPoints(txid)
	if err != nil {
		return err
	}
	return this.TxInfoRepository.DeleteMsgTx(txid)
}

func (this *TouchstoneServer) CheckTxState() error {
	feeQuote, err := this.MapiClient.GetFeeQuote()
	if err != nil {
		glog.Infof("TouchstoneServer.CheckTxState GetFeeQuote %s", err)
		return err
	}
	msgTxBriefInfos, err := this.TxInfoRepository.GetMsgTxBriefInfoByHeightRange(feeQuote.Payload.CurrentHighestBlockHeight-8, feeQuote.Payload.CurrentHighestBlockHeight+1, true)
	if err != nil {
		glog.Infof("TouchstoneServer.CheckTxState GetMsgTxBriefInfoByHeightRange %s", err)
		return err
	}
	for _, msgTxBriefInfo := range msgTxBriefInfos {
		txState, err := this.MapiClient.GetTxState(msgTxBriefInfo.Txid)
		if err != nil {
			glog.Infof("TouchstoneServer.CheckTxState GetTxState %s", err)
			return err
		}
		if strings.Contains(txState.Payload.ResultDescription, "No such mempool or blockchain transaction") {
			err := this.ClearMsgTx(msgTxBriefInfo.Txid)
			if err != nil {
				glog.Infof("TouchstoneServer.CheckTxState ClearMsgTx %s", err)
				return err
			}
			continue
		}
		if msgTxBriefInfo.BlockHash != txState.Payload.BlockHash || msgTxBriefInfo.Height != txState.Payload.BlockHeight {
			err := this.TxInfoRepository.SetMsgTxHeightHash(msgTxBriefInfo.Txid, txState.Payload.BlockHeight, txState.Payload.BlockHash)
			if err != nil {
				glog.Infof("TouchstoneServer.CheckTxState ClearMsgTx %s", err)
				return err
			}
			this.AddNeedRecomputehashPartition(msgTxBriefInfo.Height)
		}
	}
	return nil
}

func (this *TouchstoneServer) CheckTxStateLoop() {
	for {
		processId := util.RandStringBytes(8)
		glog.Infof("TouchstoneServer CheckTxStateLoop start %s", processId)
		err := this.CheckTxState()
		if err != nil {
			glog.Infof("TouchstoneServer.CheckTxStateLoop CheckTxState %s", err)
		}
		glog.Infof("TouchstoneServer CheckTxStateLoop done %s", processId)
		time.Sleep(time.Minute)
	}
}

func (this *TouchstoneServer) Init(nodeConfigs []*conf.PeerConfig, privateKeyHex string) error {
	keyByte, err := hex.DecodeString(privateKeyHex)
	if err != nil {
		return err
	}
	privateKey, _ := btcec.PrivKeyFromBytes(btcec.S256(), keyByte)
	this.privateKey = privateKey
	err = this.ConnectPeer(nodeConfigs)
	if err != nil {
		go this.ConnectPeerLoop(nodeConfigs)
	}
	go this.SyncStateLoop()
	go this.CheckTxStateLoop()
	return nil
}

func TxPoints2TxInventory(txPoints []*models.TxPoint) *TxInventory {
	txInventory := NewTxInventory()
	for _, txPoint := range txPoints {
		if txPoint.Type == models.TX_POINT_TYPE_VIN {
			txInventory.Vins = append(txInventory.Vins, txPoint)
			continue
		}
		txInventory.Vouts = append(txInventory.Vins, txPoint)
	}
	return txInventory
}

func (this *TouchstoneServer) SendRawTransaction(rawTx string) (*TxInventory, error) {
	msgTxByte, err := hex.DecodeString(rawTx)
	if err != nil {
		return nil, err
	}
	msgTx, err := util.DeserializeTxBytes(msgTxByte)
	if err != nil {
		return nil, err
	}
	hash := msgTx.TxHash()
	localSingleTxSource := &LocalSingleTxSource{
		txid:    util.GetHashByte(hash),
		txbytes: msgTxByte,
	}
	txidbytes := [][]byte{
		util.GetHashByte(hash),
	}
	syncTxsResult, err := this.SyncTxs(txidbytes, localSingleTxSource)
	if err != nil {
		return nil, err
	}
	if len(syncTxsResult.ErrTxs) > 0 {
		return nil, errors.New(syncTxsResult.ErrTxs[0].Msg)
	}
	if len(syncTxsResult.TxInventorys) > 0 {
		return syncTxsResult.TxInventorys[0], nil
	}
	return this.GetTransactionInventory(hash.String())
}

func (this *TouchstoneServer) GetTransactionInventory(txid string) (*TxInventory, error) {
	txPoints, err := this.TxPointRepository.GetTxPoints(txid)
	if err != nil {
		return nil, err
	}
	if len(txPoints) == 0 {
		return nil, errors.New("unknow tx")
	}
	return TxPoints2TxInventory(txPoints), nil
}

func (this *TouchstoneServer) GetPartitionsHash(req *message.GetPartitionsHashRequest) (*message.GetPartitionsHashResponse, error) {
	partitionInfos, err := this.PartitionInfoRepository.GetPartitionInfos(int(req.Offset), int(req.Limit))
	if err != nil {
		return nil, err
	}
	getPartitionsHashResponse := &message.GetPartitionsHashResponse{
		Hashs: make([][]byte, 0, len(partitionInfos)),
	}
	for _, partitionInfo := range partitionInfos {
		hash, err := hex.DecodeString(partitionInfo.Hash)
		if err != nil {
			return nil, err
		}
		getPartitionsHashResponse.Hashs = append(getPartitionsHashResponse.Hashs, hash)
	}
	return getPartitionsHashResponse, nil
}

func (this *TouchstoneServer) GetUnconfirmTxids(request *message.GetUnconfirmTxidsRequest) (*message.GetTxidsResponse, error) {
	txidBsons, err := this.TxInfoRepository.GetTxidsByHeightRangeOrderByTxid(-1, 1, models.TX_STATE_CLOSED, true)
	if err != nil {
		return nil, err
	}
	getTxidsResponse := &message.GetTxidsResponse{
		Txids: make([][]byte, 0, len(txidBsons)),
	}
	for _, txidBson := range txidBsons {
		txidByte, err := hex.DecodeString(txidBson.Txid)
		if err != nil {
			return nil, err
		}
		getTxidsResponse.Txids = append(getTxidsResponse.Txids, txidByte)
	}
	return getTxidsResponse, nil
}

func (this *TouchstoneServer) GetPartitionsTxids(request *message.GetTxidsByPartitionsRequest) (*message.GetTxidsResponse, error) {
	getTxidsResponse := &message.GetTxidsResponse{
		Txids: make([][]byte, 0, 128),
	}
	for _, id := range request.Ids {
		start := *conf.GStartHeight + conf.PARTITION_BLOCK_COUNT*id
		txidBsons, err := this.TxInfoRepository.GetTxidsByHeightRangeOrderByTxid(start, start+conf.PARTITION_BLOCK_COUNT, models.TX_STATE_CLOSED, true)
		if err != nil {
			return nil, err
		}
		for _, txidBson := range txidBsons {
			txidByte, err := hex.DecodeString(txidBson.Txid)
			if err != nil {
				return nil, err
			}
			getTxidsResponse.Txids = append(getTxidsResponse.Txids, txidByte)
		}
	}
	return getTxidsResponse, nil
}
