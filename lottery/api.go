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
	pb "github.com/conseweb/common/protos"
	"golang.org/x/net/context"
)

type lotteryAPI struct {
	lottery *Lottery
}

func (la *lotteryAPI) NextLotteryInfo(ctx context.Context, req *pb.NextLotteryInfoReq) (*pb.NextLotteryInfoRsp, error) {
	rsp := &pb.NextLotteryInfoRsp{
		Error: pb.ResponseOK(),
	}

	// if not during lottery, return error
	if !la.lottery.lotteryFlag {
		rsp.Error = pb.NewError(pb.ErrorType_NOT_IN_LOTTERY_INTERVAL, "now is not in lottery interval")
		goto RET
	}

	rsp.StartTime = la.lottery.lotteryStartTime
	rsp.EndTime = la.lottery.lotteryEndTime

RET:
	return rsp, nil
}

func (la *lotteryAPI) SendLotteryNum(ctx context.Context, req *pb.SendLotteryNumReq) (*pb.SendLotteryNumRsp, error) {
	rsp := &pb.SendLotteryNumRsp{
		Error: pb.ResponseOK(),
	}

	var ticket *pb.LotteryTicket
	var err error

	// if not during lottery, return error
	if !la.lottery.lotteryFlag {
		rsp.Error = pb.NewError(pb.ErrorType_NOT_IN_LOTTERY_INTERVAL, "now is not in lottery interval, reject all the lottery")
		goto RET
	}

	ticket, err = la.lottery.storage.PutFx(req.Fid, req.Fx)
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
