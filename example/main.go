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
	"fmt"
	"math/rand"
	"strconv"
	"time"

	"github.com/conseweb/common/clientconn"
	pb "github.com/conseweb/common/protos"
	"github.com/conseweb/common/semaphore"
	"github.com/conseweb/common/snowflake"
	"golang.org/x/net/context"
)

var (
	ticker  = time.NewTicker(time.Second * 10)
	address = ":7058"
)

func main() {
	for {
		select {
		case <-ticker.C:
			conn, err := clientconn.NewClientConnectionWithAddress(address, true, false, nil)
			if err != nil {
				panic(err)
			}
			defer conn.Close()

			cli := pb.NewLotteryAPIClient(conn)

			// call teller to ask next lottery info
			rspNextLotteryInfo, err := cli.NextLotteryInfo(context.Background(), &pb.NextLotteryInfoReq{})
			if err != nil {
				panic(err)
			}

			if !rspNextLotteryInfo.Error.OK() {
				fmt.Println("not in lottery round, please waiting...")
				continue
			}
			ticker.Stop()

			// call sendLotteryFx 1000 times, means there are 1000 farmer
			go func() {
				semaF := semaphore.NewSemaphore(16)
				for i := 0; i < 10000; i++ {
					semaF.Acquire()
					go func() {
						defer semaF.Release()

						id, err := snowflake.NextID(int64(pb.DeviceFor_FARMER), 0)
						if err != nil {
							panic(err)
						}
						fid := strconv.FormatUint(id, 16)
						rspFx, err := cli.SendLotteryFx(context.Background(), &pb.SendLotteryFxReq{
							Fid: fid,
							Fx:  uint64(rand.Int63n(1000)),
						})
						if err != nil {
							panic(err)
						}

						fmt.Printf("sendLotteryFx: %+v\n", rspFx)
					}()
				}
			}()

			// call sendLotteryLx 100 times, means there are 100 ledger
			go func() {
				semaL := semaphore.NewSemaphore(10)
				for i := 0; i < 100; i++ {
					semaL.Acquire()
					go func() {
						defer semaL.Release()

						id, err := snowflake.NextID(int64(pb.DeviceFor_LEDGER), 0)
						if err != nil {
							panic(err)
						}
						lid := strconv.FormatUint(id, 16)
						rspLx, err := cli.SendLotteryLx(context.Background(), &pb.SendLotteryLxReq{
							Lid: lid,
							Lx:  uint64(rand.Int63n(1000)),
						})
						if err != nil {
							panic(err)
						}

						fmt.Printf("sendLotteryLx: %+v\n", rspLx)
					}()
				}
			}()
		}
	}

	select {}
}
