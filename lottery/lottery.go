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
package lottery

import (
	"time"

	pb "github.com/conseweb/common/protos"
	"github.com/hyperledger/fabric/flogging"
	"github.com/op/go-logging"
	"github.com/spf13/viper"
	"google.golang.org/grpc"
)

var (
	lotteryLogger = logging.MustGetLogger("lottery")
)

type Lottery struct {
	gRPCServer *grpc.Server
	storage    Storage

	lotteryInterval time.Duration
	lotteryLast     time.Duration

	// whether in lottery mode
	lotteryFlag bool
	// lottery start time
	lotteryStartTime int64
	// lottery end time
	lotteryEndTime int64
}

// NewLottery created a new lottery instance
func NewLottery() *Lottery {
	flogging.LoggingInit("lottery")

	lot := new(Lottery)

	// lottery interval
	lotteryInterval, err := time.ParseDuration(viper.GetString("lottery.interval"))
	if err != nil {
		lotteryLogger.Fatalf("get lottery.interval return error: %v, exitting...", err)
	}
	lotteryLogger.Infof("get lottery.interval: %v", lotteryInterval)
	lot.lotteryInterval = lotteryInterval

	// lottery last
	lotteryLast, err := time.ParseDuration(viper.GetString("lottery.last"))
	if err != nil {
		lotteryLogger.Fatalf("get lottery.last return error: %v, exitting...", err)
	}
	lotteryLogger.Infof("get lottery.last: %v", lotteryLast)
	lot.lotteryLast = lotteryLast

	if lotteryLast >= lotteryInterval {
		lotteryLogger.Fatalf("one round lottery last time bigger than two round lottery interval")
	}

	// init storage
	lot.storage = NewDefaultStorage()

	return lot
}

// ListenLottery asyncly change lottery time
func (l *Lottery) listenLottery() {
	lotteryLogger.Debugf("begin to listen to lottery ticker")

	intervalTicker := time.NewTicker(l.lotteryInterval)

	lotteryLastCheck := func(ticker *time.Ticker) {
		for {
			select {
			case <-ticker.C:
				lotteryLogger.Debugf("lottery end at %v", time.Now().UTC())

				l.lotteryFlag = false
				l.lotteryStartTime = 0
				l.lotteryEndTime = 0
				ticker.Stop()

				// handle lottery result

				return
			}
		}
	}

	for {
		select {
		case <-intervalTicker.C:
			nowTime := time.Now().UTC()
			lotteryLogger.Debugf("new round of lottery begin at %v", nowTime)

			l.lotteryFlag = true
			l.lotteryStartTime = nowTime.Unix()
			l.lotteryEndTime = nowTime.Add(l.lotteryLast).Unix()

			go lotteryLastCheck(time.NewTicker(l.lotteryLast))
		}
	}
}

// Start start Lottery api
func (l *Lottery) Start(srv *grpc.Server) {
	lotteryLogger.Info("lottery service starting...")

	l.gRPCServer = srv
	pb.RegisterLotteryAPIServer(srv, &lotteryAPI{l})
	go l.listenLottery()

	lotteryLogger.Info("lottery service started")
}

// Stop stop Lottery api
func (l *Lottery) Stop() error {
	lotteryLogger.Info("lottery service stopping...")

	if l.gRPCServer != nil {
		l.gRPCServer.Stop()
	}

	lotteryLogger.Info("lottery service stoped")

	return nil
}
