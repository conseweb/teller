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
	"strconv"
	"time"

	pb "github.com/conseweb/common/protos"
	"github.com/conseweb/common/snowflake"
	"golang.org/x/net/context"
)

type lotteryAPI struct {
	lottery *Lottery
}

// returns next lottery info, something about time begin, end etc...
func (la *lotteryAPI) NextLotteryInfo(ctx context.Context, req *pb.NextLotteryInfoReq) (*pb.NextLotteryInfoRsp, error) {
	lotteryLogger.Debugf("lotteryAPI.NextLotteryInfo called.")

	rsp := &pb.NextLotteryInfoRsp{
		Error: pb.ResponseOK(),
	}

	// if not during lottery, return error
	if !la.lottery.lotteryFSM.Is(state_dlottery) {
		rsp.Error = pb.NewError(pb.ErrorType_NOT_IN_LOTTERY_INTERVAL, "now is not in lottery interval")
		goto RET
	}

	rsp.StartTime = la.lottery.lotteryStartTime
	rsp.EndTime = la.lottery.lotteryEndTime

RET:
	return rsp, nil
}

// receive lottery number form farmer
func (la *lotteryAPI) SendLotteryFx(ctx context.Context, req *pb.SendLotteryFxReq) (*pb.SendLotteryFxRsp, error) {
	lotteryLogger.Debugf("lotteryAPI.SendLotteryFx called.")

	rsp := &pb.SendLotteryFxRsp{
		Error: pb.ResponseOK(),
	}

	var ticket *pb.LotteryFxTicket
	var err error
	var role pb.DeviceFor

	fid, err := strconv.ParseUint(req.Fid, 16, 64)
	if err != nil {
		rsp.Error = pb.NewError(pb.ErrorType_INVALID_PARAM, err.Error())
		goto RET
	}

	role = pb.DeviceFor(snowflake.ParseRole(fid))
	lotteryLogger.Debugf("request: %+v, fid: %v, role: %v", req, fid, role)
	if role != pb.DeviceFor_FARMER {
		rsp.Error = pb.NewError(pb.ErrorType_INAPPROPRIATE_DEVICE_ROLE, "device must be a farmer")
		goto RET
	}

	// if not during lottery, return error
	if !la.lottery.lotteryFSM.Is(state_dlottery) {
		rsp.Error = pb.NewError(pb.ErrorType_NOT_IN_LOTTERY_INTERVAL, "now is not in lottery interval, reject all the lottery")
		goto RET
	}

	ticket, err = la.lottery.storageMgr.GetStorage(la.lottery.curLotteryName).PutFx(req.Fid, req.Fx)
	if err == ErrAlreadyInLotteryPool {
		rsp.Error = pb.NewError(pb.ErrorType_ALREADY_RECEIVED_LOTTERY, err.Error())
		goto RET
	} else if err != nil {
		rsp.Error = pb.NewError(pb.ErrorType_INTERNAL_ERROR, err.Error())
		goto RET
	}
	rsp.Ticket = ticket

RET:
	return rsp, nil
}

// receive lottery number form ledger
func (la *lotteryAPI) SendLotteryLx(ctx context.Context, req *pb.SendLotteryLxReq) (*pb.SendLotteryLxRsp, error) {
	lotteryLogger.Debugf("lotteryAPI.SendLotteryLx called.")

	rsp := &pb.SendLotteryLxRsp{
		Error: pb.ResponseOK(),
	}

	var ticket *pb.LotteryLxTicket
	var err error
	var role pb.DeviceFor

	lid, err := strconv.ParseUint(req.Lid, 16, 64)
	if err != nil {
		rsp.Error = pb.NewError(pb.ErrorType_INVALID_PARAM, err.Error())
		goto RET
	}

	role = pb.DeviceFor(snowflake.ParseRole(lid))
	lotteryLogger.Debugf("request: %+v, lid: %v, role: %v", req, lid, role)
	if role != pb.DeviceFor_LEDGER {
		rsp.Error = pb.NewError(pb.ErrorType_INAPPROPRIATE_DEVICE_ROLE, "device must be a ledger")
		goto RET
	}

	// if not during lottery, return error
	if !la.lottery.lotteryFSM.Is(state_dlottery) {
		rsp.Error = pb.NewError(pb.ErrorType_NOT_IN_LOTTERY_INTERVAL, "now is not in lottery interval, reject all the lottery")
		goto RET
	}

	ticket, err = la.lottery.storageMgr.GetStorage(la.lottery.curLotteryName).PutLx(req.Lid, req.Lx)
	if err == ErrAlreadyInLotteryPool {
		rsp.Error = pb.NewError(pb.ErrorType_ALREADY_RECEIVED_LOTTERY, err.Error())
		goto RET
	} else if err != nil {
		rsp.Error = pb.NewError(pb.ErrorType_INTERNAL_ERROR, err.Error())
		goto RET
	}
	rsp.Ticket = ticket

RET:
	return rsp, nil
}

// send a command to start new round of lottery immediately
func (la *lotteryAPI) StartLottery(ctx context.Context, req *pb.StartLotteryReq) (*pb.StartLotteryRsp, error) {
	lotteryLogger.Debugf("lotteryAPI.StartLottery called.")

	rsp := &pb.StartLotteryRsp{
		Error: pb.ResponseOK(),
	}

	startTime := time.Unix(req.StartUTC, 0).UTC()
	lastInterval, err := time.ParseDuration(req.LastInterval)
	if err != nil {
		lotteryLogger.Errorf("parse lastInterval return error: %v", err)
		rsp.Error = pb.NewErrorf(pb.ErrorType_INVALID_PARAM, "parse lastInterval return error: %v", err)
		goto RET
	}

	// if not ulottery, return error
	if !la.lottery.lotteryFSM.Is(state_ulottery) {
		lotteryLogger.Warning("now is in lottery interval, please waiting...")
		rsp.Error = pb.NewError(pb.ErrorType_IN_LOTTERY_INTERVAL, "now is in lottery interval, please waiting")
		goto RET
	}

	if err := la.lottery.StartNewLottery(startTime, lastInterval); err != nil {
		lotteryLogger.Errorf("manual start lottery return error: %v", err)
		rsp.Error = pb.NewError(pb.ErrorType_INTERNAL_ERROR, err.Error())
		goto RET
	}

RET:
	return rsp, nil
}

// get lottery result to verify or someelse use
func (la *lotteryAPI) GetLotteryResult(ctx context.Context, req *pb.GetLotteryResultReq) (*pb.GetLotteryResultRsp, error) {
	rsp := &pb.GetLotteryResultRsp{
		Error: pb.ResponseOK(),
	}

	s := la.lottery.storageMgr.GetStorage(req.LotteryName)
	rsp.Result = &pb.LotteryResult{
		Name: s.GetName(),
		Fxs:  s.GetFxs(),
		Lxs:  s.GetLxs(),
	}

	return rsp, nil
}
