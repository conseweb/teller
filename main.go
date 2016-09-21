/*
Copyright Mojing Inc. 2016 All Rights Reserved.
Written by mint.zhao.chiu@gmail.com. github.com: https://www.github.com/mintzhao

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

		 http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/
package main

import (
	"net"
	"runtime"

	"github.com/conseweb/common/config"
	"github.com/conseweb/common/exec"
	"github.com/conseweb/teller/lottery"
	"github.com/hyperledger/fabric/flogging"
	"github.com/op/go-logging"
	"github.com/spf13/viper"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

var (
	logger = logging.MustGetLogger("server")
	lot    *lottery.Lottery
)

func main() {
	if err := config.LoadConfig("TELLER", "teller", "github.com/conseweb/teller"); err != nil {
		logger.Fatalf("Load config error: %v", err)
	}
	flogging.LoggingInit("server")

	logger.Infof("Teller server version %s", viper.GetString("server.version"))
	runtime.GOMAXPROCS(viper.GetInt("server.gomaxprocs"))

	opts := make([]grpc.ServerOption, 0)
	if viper.GetBool("server.tls.enabled") {
		creds, err := credentials.NewServerTLSFromFile(viper.GetString("server.tls.cert.file"), viper.GetString("server.tls.key.file"))
		if err != nil {
			logger.Fatalf("create tls cert error: %v", err)
		}
		opts = append(opts, grpc.Creds(creds))
	}
	srv := grpc.NewServer(opts...)

	// lottery module
	lot = lottery.NewLottery()
	lot.Start(srv)

	addr := viper.GetString("server.port")
	lis, err := net.Listen("tcp", addr)
	if err != nil {
		logger.Fatalf("Fail to start Teller server: %v", err)
	}
	logger.Infof("Teller started at %v", addr)

	go srv.Serve(lis)
	exec.HandleSignal(lot.Stop)
}
