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
	"testing"
	"time"
	"strconv"

	pb "github.com/conseweb/common/protos"
	"github.com/conseweb/common/snowflake"
	"github.com/spf13/viper"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
	"gopkg.in/check.v1"
)

func TestALL(t *testing.T) {
	check.TestingT(t)
}

type LotteryTest struct {
	api *lotteryAPI
}

func (t *LotteryTest) SetUpTest(c *check.C) {
	viper.Set("lottery.interval", "4s")
	viper.Set("lottery.last", "2s")
	viper.Set("lottery.ledger", 4)
	viper.Set("lottery.farmer", 16)
	viper.Set("logging.lottery", "debug")

	lottery := NewLottery()
	lottery.Start(grpc.NewServer())

	t.api = &lotteryAPI{lottery}
}

func (t *LotteryTest) TearDownTest(c *check.C) {
	t.api.lottery.Stop()
}

var _ = check.Suite(&LotteryTest{})

func (t *LotteryTest) TestNextLotteryInfo(c *check.C) {
	rsp1, err := t.api.NextLotteryInfo(context.Background(), &pb.NextLotteryInfoReq{})
	c.Check(err, check.IsNil)
	c.Check(rsp1.Error.OK(), check.Equals, false)
	c.Logf("rsp1: %+v", rsp1)

	time.Sleep(time.Second * 5)
	rsp2, err := t.api.NextLotteryInfo(context.Background(), &pb.NextLotteryInfoReq{})
	c.Check(err, check.IsNil)
	c.Check(rsp2.Error.OK(), check.Equals, true)
	c.Logf("rsp2: %+v", rsp2)
}

func (t *LotteryTest) TestSendLotteryFx(c *check.C) {
	rsp1, err := t.api.SendLotteryFx(context.Background(), &pb.SendLotteryFxReq{})
	c.Check(err, check.IsNil)
	c.Check(rsp1.Error.OK(), check.Equals, false)
	c.Logf("rsp1: %+v", rsp1)

	time.Sleep(time.Second * 5)
	rsp2, err := t.api.SendLotteryFx(context.Background(), &pb.SendLotteryFxReq{
		Fid: "123456789",
		Fx:  123456789,
	})
	c.Check(err, check.IsNil)
	c.Check(rsp2.Error.OK(), check.Equals, true)
	c.Check(rsp2.Ticket.Fid, check.Equals, "123456789")
	c.Check(rsp2.Ticket.Fx, check.Equals, uint64(123456789))
	c.Check(rsp2.Ticket.Mr, check.Equals, uint64(123456789))
	c.Check(rsp2.Ticket.Idx, check.Equals, int64(0))
	c.Logf("rsp2: %+v", rsp2)

	rsp3, err := t.api.SendLotteryFx(context.Background(), &pb.SendLotteryFxReq{
		Fid: "123456789",
		Fx:  123456789,
	})
	c.Check(err, check.IsNil)
	c.Check(rsp3.Error.OK(), check.Equals, false)
	c.Logf("rsp3: %+v", rsp3)

	rsp4, err := t.api.SendLotteryFx(context.Background(), &pb.SendLotteryFxReq{
		Fid: "1234567890",
		Fx:  123456789,
	})
	c.Check(err, check.IsNil)
	c.Check(rsp4.Error.OK(), check.Equals, true)
	c.Check(rsp4.Ticket.Fid, check.Equals, "1234567890")
	c.Check(rsp4.Ticket.Fx, check.Equals, uint64(123456789))
	c.Check(rsp4.Ticket.Mr, check.Equals, uint64(0))
	c.Check(rsp4.Ticket.Idx, check.Equals, int64(1))
	c.Logf("rsp4: %+v", rsp4)
}

func (t *LotteryTest) TestSendLotteryLx(c *check.C) {
	rsp1, err := t.api.SendLotteryLx(context.Background(), &pb.SendLotteryLxReq{})
	c.Check(err, check.IsNil)
	c.Check(rsp1.Error.OK(), check.Equals, false)
	c.Logf("rsp1: %+v", rsp1)

	time.Sleep(time.Second * 5)
	id, err := snowflake.NextID(int64(pb.DeviceFor_LEDGER), 0)
	c.Check(err, check.IsNil)
	lid := strconv.FormatUint(id, 16)
	rsp2, err := t.api.SendLotteryLx(context.Background(), &pb.SendLotteryLxReq{
		Lid: lid,
		Lx:  123456789,
	})
	c.Check(err, check.IsNil)
	c.Check(rsp2.Error.OK(), check.Equals, true)
	c.Check(rsp2.Ticket.Lid, check.Equals, lid)
	c.Check(rsp2.Ticket.Lx, check.Equals, uint64(123456789))
	c.Logf("rsp2: %+v", rsp2)
}

func (t *LotteryTest) TestManualStartLottery(c *check.C) {
	// param error
	rsp1, err := t.api.StartLottery(context.Background(), &pb.StartLotteryReq{})
	c.Check(err, check.IsNil)
	c.Check(rsp1.Error.OK(), check.Equals, false)
	c.Logf("rsp1: %+v", rsp1)


	time.Sleep(time.Second * 5)

	// in lottery, return err
	rsp2, err := t.api.StartLottery(context.Background(), &pb.StartLotteryReq{
		StartUTC: time.Now().UTC().Unix(),
		LastInterval: "2s",
	})
	c.Check(err, check.IsNil)
	c.Check(rsp2.Error.OK(), check.Equals, false)
	c.Logf("rsp2: %+v", rsp2)

	time.Sleep(time.Second * 2)

	// not in lottery, return ok
	rsp3, err := t.api.StartLottery(context.Background(), &pb.StartLotteryReq{
		StartUTC: time.Now().UTC().Unix(),
		LastInterval: "2s",
	})
	c.Check(err, check.IsNil)
	c.Check(rsp3.Error.OK(), check.Equals, true)
	c.Logf("rsp3: %+v", rsp3)
}
