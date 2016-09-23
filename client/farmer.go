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

// relate to farmer lottery something together
package client

import (
	"sort"

	pb "github.com/conseweb/common/protos"
)

var (
	_ sort.Interface
)

type LotteryFxs []*pb.LotteryFx

func (fxs LotteryFxs) Len() int           { return len(fxs) }
func (fxs LotteryFxs) Less(i, j int) bool { return fxs[i].Dist < fxs[j].Dist }
func (fxs LotteryFxs) Swap(i, j int)      { fxs[i], fxs[j] = fxs[j], fxs[i] }
