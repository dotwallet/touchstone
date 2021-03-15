package controller

import (
	"context"
	"errors"

	"github.com/dotwallet/touchstone/interceptor"
	"github.com/dotwallet/touchstone/message"
	"github.com/dotwallet/touchstone/services"
	"github.com/dotwallet/touchstone/util"
	"google.golang.org/grpc/metadata"
)

type P2pController struct {
	TouchstoneServer *services.TouchstoneServer
}

func GetPeerPubkey(content context.Context) (string, error) {
	md, ok := metadata.FromIncomingContext(content)
	if !ok {
		return "", errors.New("no header")
	}
	pubkeyHex, ok := md[interceptor.KEY_PUBKEY]
	if !ok {
		return "", errors.New("no pubkey")
	}
	return pubkeyHex[0], nil
}

func (this *P2pController) NotifyTxs(content context.Context, request *message.NotifyTxsRequest) (*message.EmptyDataResponse, error) {
	pubkey, err := GetPeerPubkey(content)
	if err != nil {
		return nil, err
	}
	go this.TouchstoneServer.NotifiedTxs(request, pubkey, util.RandStringBytes(8))
	return &message.EmptyDataResponse{}, nil
}

func (this *P2pController) GetTxs(context context.Context, request *message.GetTxsRequest) (*message.GetTxsResponse, error) {
	return this.TouchstoneServer.GetTxs(request)
}

func (this *P2pController) GetPartitionsHash(context context.Context, request *message.GetPartitionsHashRequest) (*message.GetPartitionsHashResponse, error) {
	return this.TouchstoneServer.GetPartitionsHash(request)
}

func (this *P2pController) GetUnconfirmTxids(context context.Context, request *message.GetUnconfirmTxidsRequest) (*message.GetTxidsResponse, error) {
	return this.TouchstoneServer.GetUnconfirmTxids(request)
}

func (this *P2pController) GetTxidsByPartitions(context context.Context, request *message.GetTxidsByPartitionsRequest) (*message.GetTxidsResponse, error) {
	return this.TouchstoneServer.GetPartitionsTxids(request)
}
