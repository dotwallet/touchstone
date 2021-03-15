package services

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/btcsuite/btcd/btcec"
	"github.com/btcsuite/btcd/chaincfg/chainhash"
	"github.com/btcsuite/btcd/wire"
	"github.com/btcsuite/btcutil"
	"github.com/dotwallet/touchstone/conf"
	"github.com/dotwallet/touchstone/interceptor"
	"github.com/dotwallet/touchstone/mapi"
	"github.com/dotwallet/touchstone/message"
	"github.com/dotwallet/touchstone/models"
	"github.com/dotwallet/touchstone/util"
	"github.com/golang/glog"
	"google.golang.org/grpc"
)

const (
	TX_VERSION       = 2
	BADGE_DUST_LIMIT = 888
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
	AddrInfoRepository               *models.AddrInfoRepository
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
	if height < *conf.GStartHeight {
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
		glog.Infof("TouchstoneServer.ClearCacheAndSetHash info %d", id)
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
			State:     models.TX_POINT_STATE_MAY_BE_UNSPENT,
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
			PreIndex:  -1,
			Type:      models.TX_POINT_TYPE_VOUT,
			Value:     badgeVout.BadgeValue,
			BadgeCode: badgeCode,
			Timestamp: timestamp,
			State:     models.TX_POINT_STATE_MAY_BE_UNSPENT,
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
	tmp := make([]string, 0, len(msgTxs))
	for _, msgTx := range msgTxs {
		tmp = append(tmp, msgTx.TxHash().String())
		for _, vin := range msgTx.TxIn {
			preTx, ok := txs[vin.PreviousOutPoint.Hash.String()]
			if ok {
				tmp = append(tmp, preTx.TxHash().String())
			}
		}
	}
	result := make([]*wire.MsgTx, 0, len(msgTxs))
	for i := len(tmp) - 1; i >= 0; i-- {
		msgTx, ok := txs[tmp[i]]
		if ok {
			result = append(result, msgTx)
			delete(txs, msgTx.TxHash().String())
		}
	}
	glog.Infof("SortAndDistinctMsgTxs info %s", util.ToJson(result))
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

func (this *TouchstoneServer) SyncTxs(txidBytes [][]byte, txSource TxSource, processId string) (*SyncTxsResult, error) {
	lackTxids := make([][]byte, 0, 8)
	txStatesCache := make(map[string]*mapi.TxState)
	needProcessTx := make([]*wire.MsgTx, 0, 8)
	alreadyClosedTxs := make([]string, 0, 8)
	for _, txidByte := range txidBytes {
		txid := hex.EncodeToString(txidByte)
		txBriefInfo, err := this.TxInfoRepository.GetMsgTxBriefInfo(txid)
		if err != nil {
			if !strings.Contains(err.Error(), models.MONGO_NOT_FOUND) {
				glog.Infof("TouchstoneServer.SyncTxs GetMsgTxBriefInfo err:%s %s", err, processId)
				return nil, err
			}
			txState, err := this.MapiClient.GetTxState(txid)
			if err != nil {
				glog.Infof("TouchstoneServer.SyncTxs GetTxState err:%s %s", err, processId)
				return nil, err
			}
			if txState.Payload.ReturnResult != mapi.RETURN_RESULT_FAILURE {
				txStatesCache[txid] = txState
				lackTxids = append(lackTxids, txidByte)
				//todo
				glog.Infof("SyncTxs info lack: %s", hex.EncodeToString(txidByte))
			}
			continue
		}
		if txBriefInfo.State != models.TX_STATE_CLOSED {
			msgTxInfo, err := this.TxInfoRepository.GetMsgTxInfo(txid)
			if err != nil {
				glog.Infof("TouchstoneServer.SyncTxs GetMsgTxInfo err:%s %s", err, processId)
				return nil, err
			}
			this.AddNeedRecomputehashPartitionByHeight(txBriefInfo.Height)
			//todo
			glog.Infof("SyncTxs info still open: %s", hex.EncodeToString(txidByte))
			needProcessTx = append(needProcessTx, msgTxInfo.MsgTx)
		}
		alreadyClosedTxs = append(alreadyClosedTxs, txid)
	}
	txsbytes, err := txSource.GetTxBytes(lackTxids)
	if err != nil {
		glog.Infof("TouchstoneServer.SyncTxs GetTxBytes err:%s %s", err, processId)
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
			glog.Infof("TouchstoneServer.SyncTxs DeserializeTxBytes %s err:%s %s", hex.EncodeToString(txBytes), err, processId)
			continue
		}
		txState, ok := txStatesCache[msgTx.TxHash().String()]
		if !ok {
			glog.Infof("TouchstoneServer.SyncTxs txStatesCache[msgTx.TxHash().String()] not found %s %s", msgTx.TxHash().String(), processId)
			continue
		}
		height := int64(models.UNCONFIRM_TX_HEIGHT)
		if txState.Payload.BlockHash != "" {
			height = txState.Payload.BlockHeight
		}
		err = this.TxInfoRepository.AddMsgTxInfo(msgTx, height, txState.Payload.BlockHash, time.Now().Unix())
		if err != nil {
			glog.Infof("TouchstoneServer.SyncTxs AddMsgTxInfo err:%s %s %s", err, msgTx.TxHash().String(), processId)
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
	hash := hashComputer.Sum(nil)
	//todo
	glog.Infof("ComputePartitionHash id:%d hash %s", id, hex.EncodeToString(hash))
	return hash, nil
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

func (this *TouchstoneServer) SyncPatitions(start int64, limit int64, processid string) error {
	getPartitionsHashRequest := &message.GetPartitionsHashRequest{
		Offset: start,
		Limit:  limit,
	}
	err := this.ClearCacheAndSetHash()
	if err != nil {
		glog.Infof("TouchstoneServer.SyncPatitions ClearCacheAndSetHash err:%s", err)
		return err
	}
	for pubkey, peer := range this.Peers() {
		getPartitionsHashResponse, err := peer.GetPartitionsHash(context.Background(), getPartitionsHashRequest)
		if err != nil {
			glog.Infof("TouchstoneServer.SyncPatitions GetPartitionsHash %s err:%s %s", pubkey, err, processid)
			continue
		}
		partitionInfos, err := this.PartitionInfoRepository.GetPartitionInfos(int(start), int(limit))
		if err != nil {
			glog.Infof("TouchstoneServer.SyncPatitions GetPartitionInfos err:%s %s", err, processid)
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
				glog.Infof("TouchstoneServer.SyncPatitions DecodeString %d %s err:%s", partitionInfo.Id, partitionInfo.Hash, err)
				continue
			}
			peerHash := getPartitionsHashResponse.Hashs[index]
			if bytes.Equal(hash, peerHash) {
				continue
			}
			glog.Infof("TouchstoneServer.SyncPatitions hash info diff:%s %s %s %d %s", hex.EncodeToString(hash), hex.EncodeToString(peerHash), pubkey, partitionInfo.Id, processid)
			getTxidsByPartitionsRequest.Ids = append(getTxidsByPartitionsRequest.Ids, partitionInfo.Id)
		}
		if len(getTxidsByPartitionsRequest.Ids) == 0 {
			continue
		}
		getTxidsResponse, err := peer.GetTxidsByPartitions(context.Background(), getTxidsByPartitionsRequest)
		if err != nil {
			glog.Infof("TouchstoneServer.SyncPatitions GetTxidsByPartitions %s err:%s", pubkey, err)
			continue
		}
		_, err = this.SyncTxs(getTxidsResponse.Txids, peer, processid)
		if err != nil {
			glog.Infof("TouchstoneServer.SyncPatitions SyncTxs %s err:%s", pubkey, err)
			continue
		}
		err = this.ClearCacheAndSetHash()
		if err != nil {
			glog.Infof("TouchstoneServer.SyncPatitions ClearCacheAndSetHash err:%s", err)
			return err
		}
	}
	return nil
}

func (this *TouchstoneServer) SyncUnconfirmTx(processid string) {
	getUnconfirmTxidsRequest := &message.GetUnconfirmTxidsRequest{}
	for pubkey, peer := range this.Peers() {
		getTxidsResponse, err := peer.GetUnconfirmTxids(context.Background(), getUnconfirmTxidsRequest)
		if err != nil {
			glog.Infof("TouchstoneServer.SyncUnconfirmTx GetTxidsByHeights %s err:%s", pubkey, err)
			continue
		}
		_, err = this.SyncTxs(getTxidsResponse.Txids, peer, processid)
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
		glog.Infof("TouchstoneServer.SyncState GetPartitionsCount %s %s", err, processid)
		return err
	}
	this.SyncUnconfirmTx(processid)

	expectPartitionsCount := (feeQuote.Payload.CurrentHighestBlockHeight-*conf.GStartHeight)/conf.PARTITION_BLOCK_COUNT + 1
	start := count - conf.RE_CONPUTE_PARTITION_COUNT
	if start < 0 || syncAll {
		start = 0
	}
	glog.Infof("TouchstoneServer.SyncState current height %d %t %d processid:%s", feeQuote.Payload.CurrentHighestBlockHeight, syncAll, start, processid)
	for i := start; i < expectPartitionsCount; i++ {
		this.AddNeedRecomputehashPartition(i)
	}
	return this.SyncPatitions(start, expectPartitionsCount, processid)
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
		glog.Infof("TouchstoneServer ConnectPeerLoop start %s", processId)
		err := this.ConnectPeer(peerConfigs)
		if err == nil {
			glog.Infof("TouchstoneServer ConnectPeerLoop total done %s", processId)
			return
		}
		glog.Infof("TouchstoneServer ConnectPeerLoop done %s", processId)
		time.Sleep(time.Minute)
	}
}

func (this *TouchstoneServer) SetSpent() error {
	feeQuote, err := this.MapiClient.GetFeeQuote()
	if err != nil {
		glog.Infof("TouchstoneServer.SyncState GetFeeQuote %s", err)
		return err
	}
	txPoint := &models.TxPoint{}
	f := func() error {
		msgTxBriefInfo, err := this.TxInfoRepository.GetMsgTxBriefInfo(txPoint.Txid)
		if err != nil {
			return err
		}
		if msgTxBriefInfo.Height == -1 || feeQuote.Payload.CurrentHighestBlockHeight-msgTxBriefInfo.Height <= 20 {
			return nil
		}
		err = this.TxPointRepository.SetTxPointState(txPoint.PreTxid, txPoint.PreIndex, models.TX_POINT_TYPE_VOUT, models.TX_POINT_STATE_PRETTY_SURE_SPENT)
		if err != nil {
			return err
		}
		err = this.TxPointRepository.SetTxPointState(txPoint.Txid, txPoint.Index, models.TX_POINT_TYPE_VIN, models.TX_POINT_STATE_PRETTY_SURE_SPENT)
		if err != nil {
			return err
		}
		return nil
	}
	return this.TxPointRepository.ForearchUnspentVinTxPoint(time.Now().Unix()-60*60, txPoint, f)
}

func (this *TouchstoneServer) SetSpentLoop() {
	for {
		processId := util.RandStringBytes(8)
		glog.Infof("TouchstoneServer SetSpentLoop start %s", processId)
		err := this.SetSpent()
		if err != nil {
			glog.Infof("TouchstoneServer CheckSpentLoop SetSpent %s %s", err, processId)
		}
		glog.Infof("TouchstoneServer SetSpentLoop done %s", processId)
		time.Sleep(time.Minute * 10)
	}
}

func (this *TouchstoneServer) SyncStateLoop() {
	for {
		processId := util.RandStringBytes(8)
		glog.Infof("TouchstoneServer SyncStateLoop start %s", processId)
		err := this.SyncState(true, processId)
		if err != nil {
			glog.Infof("TouchstoneServer.SyncStateLoop SyncState err:%s", err)
		}
		glog.Infof("TouchstoneServer SyncStateLoop done %s", processId)
		time.Sleep(time.Minute * 5)
	}
}

func (this *TouchstoneServer) NotifiedTxs(request *message.NotifyTxsRequest, peerPubkey string, processid string) {
	peer, ok := this.Peers()[peerPubkey]
	if !ok {
		glog.Infof("TouchstoneServer.NotifiedTxs err:peer not found %s", peerPubkey)
		return
	}
	_, err := this.SyncTxs(request.Txids, peer, processid)
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

		currentheight := txState.Payload.BlockHeight
		if currentheight == 0 {
			currentheight = models.UNCONFIRM_TX_HEIGHT
		}

		if msgTxBriefInfo.BlockHash != txState.Payload.BlockHash || msgTxBriefInfo.Height != currentheight {
			err := this.TxInfoRepository.SetMsgTxHeightHash(msgTxBriefInfo.Txid, txState.Payload.BlockHeight, txState.Payload.BlockHash)
			if err != nil {
				glog.Infof("TouchstoneServer.CheckTxState ClearMsgTx %s", err)
				return err
			}
			this.AddNeedRecomputehashPartitionByHeight(msgTxBriefInfo.Height)
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

func (this *TouchstoneServer) SetPrivateKey(privateKeyHex string) error {
	keyByte, err := hex.DecodeString(privateKeyHex)
	if err != nil {
		return err
	}
	privateKey, _ := btcec.PrivKeyFromBytes(btcec.S256(), keyByte)
	this.privateKey = privateKey
	return nil
}

func (this *TouchstoneServer) Init(nodeConfigs []*conf.PeerConfig, privateKeyHex string) error {
	err := this.SetPrivateKey(privateKeyHex)
	if err != nil {
		return err
	}
	err = this.ConnectPeer(nodeConfigs)
	if err != nil {
		go this.ConnectPeerLoop(nodeConfigs)
	}
	go this.SyncStateLoop()
	go this.CheckTxStateLoop()
	go this.SetSpentLoop()
	return nil
}

func TxPoints2TxInventory(txPoints []*models.TxPoint) *TxInventory {
	txInventory := NewTxInventory()
	for _, txPoint := range txPoints {
		if txPoint.Type == models.TX_POINT_TYPE_VIN {
			txInventory.Vins = append(txInventory.Vins, txPoint)
			continue
		}
		txInventory.Vouts = append(txInventory.Vouts, txPoint)
	}
	return txInventory
}

func (this *TouchstoneServer) SendRawTransaction(rawTx string, processid string) (*TxInventory, error) {
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
	syncTxsResult, err := this.SyncTxs(txidbytes, localSingleTxSource, processid)
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
	txidBsons, err := this.TxInfoRepository.GetTxidsByHeightRangeOrderByTxid(-1, 1, models.TX_STATE_CLOSED, false)
	if err != nil {
		return nil, err
	}
	getTxidsResponse := &message.GetTxidsResponse{
		Txids: make([][]byte, 0, len(txidBsons)),
	}
	txidSet := make(map[string]bool)
	for _, txidBson := range txidBsons {
		_, ok := txidSet[txidBson.Txid]
		if ok {
			continue
		}
		txidSet[txidBson.Txid] = true
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
	txidSet := make(map[string]bool)
	for _, id := range request.Ids {
		start := *conf.GStartHeight + conf.PARTITION_BLOCK_COUNT*id
		txidBsons, err := this.TxInfoRepository.GetTxidsByHeightRangeOrderByTxid(start, start+conf.PARTITION_BLOCK_COUNT, models.TX_STATE_CLOSED, false)
		if err != nil {
			return nil, err
		}
		for _, txidBson := range txidBsons {
			_, ok := txidSet[txidBson.Txid]
			if ok {
				continue
			}
			txidSet[txidBson.Txid] = true
			txidByte, err := hex.DecodeString(txidBson.Txid)
			if err != nil {
				return nil, err
			}
			getTxidsResponse.Txids = append(getTxidsResponse.Txids, txidByte)
		}
	}
	return getTxidsResponse, nil
}

func (this *TouchstoneServer) SetAddrInfo(appid string, userId int64, userIndex int64, addr string) error {
	addrInfo := &models.AddrInfo{
		Appid:     appid,
		UserID:    userId,
		UserIndex: userIndex,
		Addr:      addr,
		Timestamp: time.Now().Unix(),
	}
	return this.AddrInfoRepository.AddOrUpdateAddrInfo(addrInfo)
}

type TxPointsSorter []*models.TxPoint

func (this TxPointsSorter) Len() int {
	return len(this)
}

func (this TxPointsSorter) Swap(i, j int) {
	this[i], this[j] = this[j], this[i]
}

func (this TxPointsSorter) Less(i, j int) bool {
	return this[i].Timestamp < this[j].Timestamp
}

func PageTxPoints(txPoints []*models.TxPoint, offset int, limit int) []*models.TxPoint {
	sort.Sort(TxPointsSorter(txPoints))
	result := make([]*models.TxPoint, 0, 8)
	if len(txPoints) <= offset {
		return result
	}
	if len(txPoints) < limit+offset {
		result = txPoints[offset:]
		return result
	}
	result = txPoints[offset : limit+offset]
	return result
}

func (this *TouchstoneServer) CalculateUtxos(txPoints []*models.TxPoint) []*models.TxPoint {
	SpentUtxoSet := make(map[string]bool)
	for _, txPoint := range txPoints {
		if txPoint.Type != models.TX_POINT_TYPE_VIN {
			continue
		}
		key := util.GenerateStrFromStrInt(txPoint.PreTxid, txPoint.PreIndex)
		SpentUtxoSet[key] = true
	}
	result := make([]*models.TxPoint, 0, 8)
	for _, txPoint := range txPoints {
		if txPoint.Type != models.TX_POINT_TYPE_VOUT {
			continue
		}
		key := util.GenerateStrFromStrInt(txPoint.Txid, txPoint.Index)
		_, ok := SpentUtxoSet[key]
		if ok {
			continue
		}
		result = append(result, txPoint)
	}
	return result
}

func (this *TouchstoneServer) GetAllAddrUtxos(addr string, badgeCode string) ([]*models.TxPoint, error) {
	txPoints, err := this.TxPointRepository.GetTxPointsByAddr(addr, badgeCode, models.TX_POINT_STATE_MAY_BE_UNSPENT)
	if err != nil {
		return nil, err
	}
	return this.CalculateUtxos(txPoints), nil
}

type GetUtxosResult struct {
	Utxos []*models.TxPoint `json:"utxos"`
}

func (this *TouchstoneServer) GetAddrUtxos(addr string, badgeCode string, offset int, limit int) (*GetUtxosResult, error) {
	utxos, err := this.GetAllAddrUtxos(addr, badgeCode)
	if err != nil {
		return nil, err
	}
	return &GetUtxosResult{
		Utxos: PageTxPoints(utxos, offset, limit),
	}, nil
}

func (this *TouchstoneServer) GetAllUserUtxos(appid string, userid int64, userIndex int64, badgeCode string) ([]*models.TxPoint, error) {
	txPoints, err := this.AddrInfoRepository.GetUserTxPoints(appid, userid, userIndex, badgeCode, models.TX_POINT_STATE_MAY_BE_UNSPENT)
	if err != nil {
		return nil, err
	}
	return this.CalculateUtxos(txPoints), nil
}

func (this *TouchstoneServer) GetUserUtxos(appid string, userid int64, userIndex int64, badgeCode string, offset int, limit int) (*GetUtxosResult, error) {
	utxos, err := this.GetAllUserUtxos(appid, userid, userIndex, badgeCode)
	if err != nil {
		return nil, err
	}
	return &GetUtxosResult{
		Utxos: PageTxPoints(utxos, offset, limit),
	}, nil
}

func SumTxPoints(txPoints []*models.TxPoint) int64 {
	sum := int64(0)
	for _, txPoint := range txPoints {
		sum += txPoint.Value
	}
	return sum
}

type GetBalanceRsp struct {
	Balance int64 `json:"balance"`
}

func (this *TouchstoneServer) GetAddrBalance(addr string, badgeCode string) (*GetBalanceRsp, error) {
	utxos, err := this.GetAllAddrUtxos(addr, badgeCode)
	if err != nil {
		return nil, err
	}
	return &GetBalanceRsp{
		Balance: SumTxPoints(utxos),
	}, nil
}

func (this *TouchstoneServer) GetUserBalance(appid string, userid int64, userIndex int64, badgeCode string) (*GetBalanceRsp, error) {
	utxos, err := this.GetAllUserUtxos(appid, userid, userIndex, badgeCode)
	if err != nil {
		return nil, err
	}
	return &GetBalanceRsp{
		Balance: SumTxPoints(utxos),
	}, nil
}

type AddrInventory struct {
	Addr      string `json:"addr"`
	Txid      string `json:"txid"`
	Timestamp int64  `json:"timestamp"`
	Value     int64  `json:"value"`
}

type AddrInventorySorter []*AddrInventory

func (this AddrInventorySorter) Len() int {
	return len(this)
}

func (this AddrInventorySorter) Less(i, j int) bool {
	return this[i].Timestamp > this[j].Timestamp
}

func (this AddrInventorySorter) Swap(i, j int) {
	this[i], this[j] = this[j], this[i]
}

func TxPoints2AddrInventory(txPoints []*models.TxPoint) []*AddrInventory {
	addrInventorys := make([]*AddrInventory, 0, 8)
	addrInventorySet := make(map[string]*AddrInventory)
	for _, txPoint := range txPoints {
		addrInventory, ok := addrInventorySet[txPoint.Txid]
		if !ok {
			addrInventory = &AddrInventory{
				Addr:      txPoint.Addr,
				Txid:      txPoint.Txid,
				Timestamp: txPoint.Timestamp,
			}
			addrInventorySet[txPoint.Txid] = addrInventory
			addrInventorys = append(addrInventorys, addrInventory)
		}
		addrInventory.Value += txPoint.Value
	}
	return addrInventorys
}

func (this *TouchstoneServer) GetAllAddrInventorys(addr string, badgeCode string) ([]*AddrInventory, error) {
	txPoints, err := this.TxPointRepository.GetTxPointsByAddr(addr, badgeCode, models.TX_POINT_STATE_ALL)
	if err != nil {
		return nil, err
	}
	return TxPoints2AddrInventory(txPoints), nil
}

func (this *TouchstoneServer) GetUserAddrInventorys(appid string, userid int64, userIndex int64, badgeCode string) ([]*AddrInventory, error) {
	txPoints, err := this.AddrInfoRepository.GetUserTxPoints(appid, userid, userIndex, badgeCode, models.TX_POINT_STATE_ALL)
	if err != nil {
		return nil, err
	}
	return TxPoints2AddrInventory(txPoints), nil
}

func PageAddrInventorys(addrInventorys []*AddrInventory, offset int, limit int) []*AddrInventory {
	sort.Sort(AddrInventorySorter(addrInventorys))
	result := make([]*AddrInventory, 0, 8)
	if len(addrInventorys) <= offset {
		return result
	}
	if len(addrInventorys) < limit+offset {
		result = addrInventorys[offset:]
		return result
	}
	result = addrInventorys[offset : limit+offset]
	return result
}

type GetAddrInventoryRsp struct {
	AddrInventorys []*AddrInventory `json:"addr_inventorys"`
}

func (this *TouchstoneServer) GetAddrInventorys(addr string, badgeCode string, offset int, limit int) (*GetAddrInventoryRsp, error) {
	addrInventorys, err := this.GetAllAddrInventorys(addr, badgeCode)
	if err != nil {
		return nil, err
	}
	return &GetAddrInventoryRsp{
		AddrInventorys: PageAddrInventorys(addrInventorys, offset, limit),
	}, nil
}

func (this *TouchstoneServer) GetUserInventorys(appid string, userid int64, userIndex int64, badgeCode string, offset int, limit int) (*GetAddrInventoryRsp, error) {
	addrInventorys, err := this.GetUserAddrInventorys(appid, userid, userIndex, badgeCode)
	if err != nil {
		return nil, err
	}
	return &GetAddrInventoryRsp{
		AddrInventorys: PageAddrInventorys(addrInventorys, offset, limit),
	}, nil
}

type AddrAmount struct {
	Addr   string
	Amount int64
}

type SendBadgeToAddressRsp struct {
	UnFinishedTx string            `json:"unfinished_tx"`
	Vins         []*models.TxPoint `json:"vins"`
}

func (this *TouchstoneServer) SendBadgeToAddress(appid string, userid int64, userIndex int64, badgeCode string, changeAddrStr string, addrAmounts []*AddrAmount, amount2burn int64) (*SendBadgeToAddressRsp, error) {
	if amount2burn < 0 {
		return nil, errors.New("amount2burn < 0")
	}
	changeAddr, err := btcutil.DecodeAddress(changeAddrStr, conf.GNetParam)
	if err != nil {
		return nil, err
	}
	msgTx := wire.NewMsgTx(TX_VERSION)
	voutValue := amount2burn
	for _, addrAmount := range addrAmounts {
		addr, err := btcutil.DecodeAddress(addrAmount.Addr, conf.GNetParam)
		if err != nil {
			return nil, err
		}
		script, err := util.CreateBadgeLockScript(addr, addrAmount.Amount)
		if err != nil {
			return nil, err
		}
		voutValue += addrAmount.Amount
		vout := wire.NewTxOut(BADGE_DUST_LIMIT, script)
		msgTx.AddTxOut(vout)
	}
	txPoints, err := this.GetAllUserUtxos(appid, userid, userIndex, badgeCode)
	if err != nil {
		return nil, err
	}
	usedVins := make([]*models.TxPoint, 0)
	vinValue := int64(0)
	for _, txPoint := range txPoints {
		if vinValue >= voutValue {
			break
		}
		hash, err := chainhash.NewHashFromStr(txPoint.Txid)
		if err != nil {
			return nil, err
		}
		outPoint := wire.NewOutPoint(hash, uint32(txPoint.Index))
		vin := wire.NewTxIn(outPoint, nil, nil)
		msgTx.AddTxIn(vin)
		usedVins = append(usedVins, txPoint)
		vinValue += txPoint.Value
	}
	change := vinValue - voutValue
	if change < 0 {
		return nil, errors.New("not enough badge")
	}
	if change == 0 {
		UnFinishedTx := util.SeserializeMsgTxStr(msgTx)
		return &SendBadgeToAddressRsp{
			UnFinishedTx: UnFinishedTx,
			Vins:         usedVins,
		}, nil
	}
	script, err := util.CreateBadgeLockScript(changeAddr, change)
	if err != nil {
		return nil, err
	}
	vout := wire.NewTxOut(BADGE_DUST_LIMIT, script)
	msgTx.AddTxOut(vout)

	UnFinishedTx := util.SeserializeMsgTxStr(msgTx)

	return &SendBadgeToAddressRsp{
		UnFinishedTx: UnFinishedTx,
		Vins:         usedVins,
	}, nil
}
