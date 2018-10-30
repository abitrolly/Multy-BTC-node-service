/*
Copyright 2018 Idealnaya rabota LLC
Licensed under Multy.io license.
See LICENSE for details
*/
package node

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"net"
	"sync"
	"time"

	"github.com/KristinaEtc/slf"
	_ "github.com/KristinaEtc/slflog"
	"github.com/Multy-io/Multy-BTC-node-service/btc"
	pb "github.com/Multy-io/Multy-BTC-node-service/node-streamer"
	"github.com/Multy-io/Multy-BTC-node-service/streamer"
	"github.com/Multy-io/Multy-back/store"
	"github.com/blockcypher/gobcy"
	"google.golang.org/grpc"
)

var (
	log = slf.WithContext("NodeClient")
)

// Multy is a main struct of service

// NodeClient is a main struct of service
type NodeClient struct {
	Config     *Configuration
	Instance   *btc.Client
	GRPCserver *streamer.Server
	Clients    *sync.Map // address to userid
	BtcApi     *gobcy.API
}

// Init initializes Multy instance
func (nc *NodeClient) Init(conf *Configuration) (*NodeClient, error) {
	cli := &NodeClient{
		Config: conf,
	}

	usersData := sync.Map{}

	usersData.Store("address", store.AddressExtended{
		UserID:       "kek",
		WalletIndex:  1,
		AddressIndex: 2,
	})

	api := gobcy.API{
		Token: conf.BTCAPI.Token,
		Coin:  conf.BTCAPI.Coin,
		Chain: conf.BTCAPI.Chain,
	}
	cli.BtcApi = &api
	log.Debug("btc api initialization done √")

	// initail initialization of clients data
	cli.Clients = &usersData
	log.Debug("Users data initialization done √")

	// init gRPC server
	lis, err := net.Listen("tcp", conf.GrpcPort)
	if err != nil {
		return nil, fmt.Errorf("failed to listen: %v", err.Error())
	}

	btcClient, err := btc.NewClient(getCertificate(conf.BTCSertificate), conf.BTCNodeAddress, cli.Clients)
	if err != nil {
		return nil, fmt.Errorf("Blockchain api initialization: %s", err.Error())
	}
	log.Debug("BTC client initialization done √")
	cli.Instance = btcClient

	// Creates a new gRPC server
	s := grpc.NewServer()
	srv := streamer.Server{
		UsersData:  cli.Clients,
		BtcAPI:     cli.BtcApi,
		M:          &sync.Mutex{},
		BtcCli:     btcClient,
		Info:       &conf.ServiceInfo,
		GRPCserver: s,
		Listener:   lis,
		ReloadChan: make(chan struct{}),
	}

	cli.GRPCserver = &srv

	pb.RegisterNodeCommuunicationsServer(s, &srv)

	go s.Serve(lis)

	go WathReload(srv.ReloadChan, cli)

	log.Debug("NodeCommuunications Server initialization done √")

	return cli, nil
}

func getCertificate(certFile string) []byte {
	cert, err := ioutil.ReadFile(certFile)
	cert = bytes.Trim(cert, "\x00")

	if err != nil {
		log.Errorf("get certificate: %s", err.Error())
		return []byte{}
	}
	if len(cert) > 1 {
		return cert
	}
	log.Errorf("get certificate: empty certificate")
	return []byte{}
}

func WathReload(reload chan struct{}, cli *NodeClient) {
	// func WathReload(reload chan struct{}, s *grpc.Server, srv *streamer.Server, lis net.Listener, conf *Configuration) {
	for {
		select {
		case _ = <-reload:
			ticker := time.NewTicker(1 * time.Second)

			err := cli.GRPCserver.Listener.Close()
			if err != nil {
				log.Errorf("WathReload:lis.Close %v", err.Error())
			}
			cli.GRPCserver.GRPCserver.Stop()
			log.Warnf("WathReload:Successfully stopped")
			for _ = range ticker.C {
				_, err := cli.Init(cli.Config)
				if err != nil {
					log.Errorf("WathReload:Init %v ", err)
					continue
				}

				log.Warnf("WathReload:Successfully reloaded")
				return
			}
		}
	}
}

func closeServerChannels(srv *streamer.Server) {
	close(srv.BtcCli.AddSpOut)
	close(srv.BtcCli.AddToMempool)
	close(srv.BtcCli.Block)
	close(srv.BtcCli.DeleteMempool)
	close(srv.BtcCli.DelSpOut)
	close(srv.BtcCli.ResyncCh)
	close(srv.BtcCli.TransactionsCh)
}
