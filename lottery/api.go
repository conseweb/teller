/*
Copyright Mojing Inc. 2016 All Rights Reserved.

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

	if !la.lottery.lotteryFlag {
		rsp.Error = pb.NewError(pb.ErrorType_NOT_IN_LOTTERY_INTERVAL, "now is not in lottery interval")
		goto RET
	}

	rsp.StartTime = la.lottery.lotteryStartTime
	rsp.EndTime = la.lottery.lotteryEndTime

RET:
	return rsp, nil
}
