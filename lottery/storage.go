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
	"errors"
	"sort"
	"sync"

	pb "github.com/conseweb/common/protos"
	"github.com/conseweb/teller/client"
)

// StorageCreator create a new instance of Storage interface
type StorageCreator func(string) Storage

// storage manager manage storage backend
// because lottery will persist previous lottery history, so we need a manager to handle
// after persist, storages can be deleted
type StorageManager struct {
	sync.Mutex
	storages map[string]Storage
	creator StorageCreator
}

// NewStorageManager return a storage manager with storage creator
func NewStorageManager(creator StorageCreator) *StorageManager {
	return &StorageManager{
		creator: creator,
		storages: make(map[string]Storage),
	}
}

// GetStorage
// if mgr's storage is nil, using creator create a new one
// if mgr's storage is not nil, just return it
func (mgr *StorageManager) GetStorage(name string) Storage {
	mgr.Lock()
	defer mgr.Unlock()

	storage, ok := mgr.storages[name]
	if ok {
		return storage
	}

	storage = mgr.creator(name)
	mgr.storages[name] = storage

	return storage
}

var (
	ErrAlreadyInLotteryPool = errors.New("already put fx into lottery pool")
)

type Storage interface {
	// put farmer random lottery number
	PutFx(string, uint64) (*pb.LotteryFxTicket, error)

	// put ledger random lottery number
	PutLx(string, uint64) (*pb.LotteryLxTicket, error)

	// vote candidate, won flag indicate the ledger has been selected
	VoteCandidate(winnerNum int)

	// persist current lottery history to a backend, maybe a db or something else
	Persist() error
}

type InMemoryStorage struct {
	name string

	// farmer area
	fids map[string]bool
	fxs  []*pb.LotteryFx

	r uint64
	sync.Mutex

	// ledger area
	lids map[string]bool
	lxs  []*pb.LotteryLx
}

// NewInMemoryStorage return a in memory storage
func NewInMemoryStorage(name string) Storage {
	storage := new(InMemoryStorage)

	storage.name = name
	storage.fids = make(map[string]bool)
	storage.fxs = make([]*pb.LotteryFx, 0)
	storage.r = 0
	storage.lids = make(map[string]bool)
	storage.lxs = make([]*pb.LotteryLx, 0)

	return storage
}

// PutFx record farmer lottery, and return a ticket
func (s *InMemoryStorage) PutFx(fid string, fx uint64) (*pb.LotteryFxTicket, error) {
	s.Lock()
	defer s.Unlock()

	if _, ok := s.fids[fid]; ok {
		return nil, ErrAlreadyInLotteryPool
	}

	s.fids[fid] = true
	s.r ^= fx
	s.fxs = append(s.fxs, &pb.LotteryFx{
		Fid:   fid,
		Value: fx,
		Mr:    s.r,
	})

	return &pb.LotteryFxTicket{
		Fid: fid,
		Fx:  fx,
		Mr:  s.r,
		Idx: int64(len(s.fxs) - 1),
	}, nil
}

// PutLx record ledger lottery, and return a ticket
func (s *InMemoryStorage) PutLx(lid string, lx uint64) (*pb.LotteryLxTicket, error) {
	s.Lock()
	defer s.Unlock()

	if _, ok := s.lids[lid]; ok {
		return nil, ErrAlreadyInLotteryPool
	}

	s.lids[lid] = true
	s.lxs = append(s.lxs, &pb.LotteryLx{
		Lid:   lid,
		Value: lx,
	})

	return &pb.LotteryLxTicket{
		Lid: lid,
		Lx:  lx,
	}, nil
}

// VoteCandidate select some(winnerNum) candidate to record the blockchain
// and give back coinbase transaction(lepuscoin) for a period
func (s *InMemoryStorage) VoteCandidate(winnerNum int) {
	s.Lock()
	defer s.Unlock()

	lenCandidate := len(s.lxs)

	// all the candidate are been selected
	if lenCandidate <= winnerNum {
		for idx, _ := range s.lxs {
			s.lxs[idx].Won = true
		}
	} else {
		for idx, lx := range s.lxs {
			s.lxs[idx].Dist = lx.Value ^ s.r
		}

		sort.Sort(client.LotteryLxs(s.lxs))
		for idx, _ := range s.lxs[:winnerNum-1] {
			s.lxs[idx].Won = true
		}
	}
}

// persist current lottery history to a backend, maybe a db or something else
func (s *InMemoryStorage) Persist() error {
	return nil
}