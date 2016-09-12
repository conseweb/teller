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
	"errors"
	pb "github.com/conseweb/common/protos"
)

var (
	ErrAlreadyInLotteryPool = errors.New("already put fx into lottery pool")
)

type Storage interface {
	// put farmer random lottery tickets
	PutFx(string, uint64) (*pb.LotteryTicket, error)
	//
}

type lotterFx struct {
	fid   string
	value uint64
	mr    uint64
}

type InMemoryStorage struct {
	fids map[string]bool
	fxs  []*lotterFx
	r    uint64
}

// NewDefaultStorage return a in memory storage
func NewDefaultStorage() Storage {
	storage := new(InMemoryStorage)

	storage.fids = make(map[string]bool)
	storage.fxs = make([]*lotterFx, 0)
	storage.r = 0

	return storage
}

// PutFx record farmer lottery, and return a ticket
func (s *InMemoryStorage) PutFx(fid string, fx uint64) (*pb.LotteryTicket, error) {
	if _, ok := s.fids[fid]; ok {
		return nil, ErrAlreadyInLotteryPool
	}

	s.fids[fid] = true
	s.r ^= fx
	s.fxs = append(s.fxs, &lotterFx{
		fid:   fid,
		value: fx,
		mr:    s.r,
	})

	return &pb.LotteryTicket{
		Fid: fid,
		Fx:  fx,
		Mr:  s.r,
		Idx: int64(len(s.fxs) - 1),
	}, nil
}
