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
	"fmt"
	"time"

	pb "github.com/conseweb/common/protos"
	"github.com/hyperledger/fabric/flogging"
	"github.com/looplab/fsm"
	"github.com/op/go-logging"
	"github.com/spf13/viper"
	"google.golang.org/grpc"
)

var (
	lotteryLogger = logging.MustGetLogger("lottery")

	// lottery fsm states
	state_ulottery = "ulottery" // un lottery
	state_dlottery = "dlottery" // during lottery
	state_hlottery = "hlottery" // handle lottery result

	// lottery fsm events
	event_beginLottery  = "lottering"   // ulottery --> dlottery
	event_handleLottery = "calculating" // dlottery --> hlottery
	event_finishLottery = "finished"    // hlottery --> ulottery
)

// Lottery
type Lottery struct {
	gRPCServer *grpc.Server
	storageMgr *StorageManager

	lotteryInterval time.Duration
	lotteryLast     time.Duration

	lotteryStartTime int64  // lottery start time
	lotteryEndTime   int64  // lottery end time
	curLotteryName   string // current round of lottery name

	lotteryFSM *fsm.FSM
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
	lot.storageMgr = NewStorageManager(NewInMemoryStorage)

	// init fms
	lot.lotteryFSM = fsm.NewFSM(state_ulottery, []fsm.EventDesc{
		fsm.EventDesc{Name: event_beginLottery, Src: []string{state_ulottery}, Dst: state_dlottery},
		fsm.EventDesc{Name: event_handleLottery, Src: []string{state_dlottery}, Dst: state_hlottery},
		fsm.EventDesc{Name: event_finishLottery, Src: []string{state_hlottery}, Dst: state_ulottery},
	}, map[string]fsm.Callback{
		"before_event": lot.beforeEvent,
	})

	return lot
}

// fsm change logger
func (l *Lottery) beforeEvent(e *fsm.Event) {
	lotteryLogger.Debugf("lottery calling event[%s], state %s ==> %s", e.Event, e.Src, e.Dst)
}

// ListenLottery asyncly change lottery time
func (l *Lottery) listenLottery() {
	lotteryLogger.Debugf("begin to listen to lottery ticker")
	intervalTicker := time.NewTicker(l.lotteryInterval)

	for {
		select {
		case <-intervalTicker.C:
			nowTime := time.Now().UTC()

			l.StartNewLottery(nowTime, l.lotteryLast)
		}
	}
}

func (l *Lottery) lotteryLastCheck(ticker *time.Ticker) {
	<-ticker.C
	ticker.Stop()
	lotteryLogger.Debugf("lottery end at %v", time.Now().UTC())

	// handle lottery result
	if err := l.lotteryFSM.Event(event_handleLottery); err != nil {
		lotteryLogger.Errorf("cant handle lottery: %v", err)
		return
	}

	l.storageMgr.HandleLottery(l.curLotteryName, viper.GetInt("lottery.seat"), viper.GetInt("lottery.worker"))
	l.storageMgr.DisplayLotteryResult(l.curLotteryName)

	l.lotteryStartTime = 0
	l.lotteryEndTime = 0
	l.curLotteryName = ""

	// after handler lottery result
	if err := l.lotteryFSM.Event(event_finishLottery); err != nil {
		lotteryLogger.Errorf("cant finish lottery: %v", err)
		return
	}

	return
}

// Start a new round of lottery
func (l *Lottery) StartNewLottery(startTime time.Time, lastInterval time.Duration) error {
	timeNow := time.Now().UTC()
	if timeNow.Sub(startTime) > time.Minute {
		startTime = timeNow
	}
	lotteryLogger.Debugf("lottery begin at %v", startTime)

	if err := l.lotteryFSM.Event(event_beginLottery); err != nil {
		lotteryLogger.Errorf("can't begin lottery: %v", err)
		return fmt.Errorf("can't begin lottery: %v", err)
	}

	l.lotteryStartTime = startTime.Unix()
	l.lotteryEndTime = startTime.Add(lastInterval).Unix()
	l.curLotteryName = fmt.Sprintf("lottery_%v", startTime.UnixNano())

	go l.lotteryLastCheck(time.NewTicker(lastInterval))
	return nil
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
