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
	"github.com/conseweb/common/semaphore"
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
	creator  StorageCreator
}

// NewStorageManager return a storage manager with storage creator
func NewStorageManager(creator StorageCreator) *StorageManager {
	return &StorageManager{
		creator:  creator,
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

// HandleLottery
func (mgr *StorageManager) HandleLottery(name string, winnerNum, worker int) {
	s := mgr.GetStorage(name)

	s.SelectCandidate(winnerNum)
	s.ChooseFarmerVote(worker)
	s.Persist()
}

// DisplayLotteryResult
func (mgr *StorageManager) DisplayLotteryResult(name string) {
	s := mgr.GetStorage(name)

	lotteryLogger.Infof("Display Lottery[%s] Result Start\n", name)
	lotteryLogger.Infof("END R: %d\n", s.GetEndR())
	ledgers := s.GetWonLedgers()
	farmers := s.GetFxs()
	lotteryLogger.Infof("Lottery Farmer vote display start==========================================================\n")
	for idx, fx := range farmers {
		lotteryLogger.Infof("farmer %d: <ID %s>, <Value %d>, <Candidate %s>", idx, fx.Fid, fx.Value, fx.Candidate)
	}
	lotteryLogger.Infof("Lottery Farmer vote display end============================================================\n")

	lotteryLogger.Infof("Lottery selected ledgers, count: %v\n", len(ledgers))
	for idx, lx := range ledgers {
		farmerCnt := 0
		for _, fx := range farmers {
			if fx.Candidate == lx.Lid {
				farmerCnt ++
			}
		}

		lotteryLogger.Infof("selected ledger %d: <ID %s>, <Value %d>, <Dist %d>, <Vote %d>\n", idx, lx.Lid, lx.Value, lx.Dist, farmerCnt)
	}

	lotteryLogger.Infof("Display Lottery[%s] Result End\n", name)
}

var (
	ErrAlreadyInLotteryPool = errors.New("already put fx into lottery pool")
)

type Storage interface {
	// put farmer random lottery number
	PutFx(string, uint64) (*pb.LotteryFxTicket, error)

	// put ledger random lottery number
	PutLx(string, uint64) (*pb.LotteryLxTicket, error)

	// GetEndR return finally R
	GetEndR() uint64

	// select candidate, won flag indicate the ledger has been selected
	// param[0] stands for how many candidate can be selected
	SelectCandidate(int)

	// GetWonLedgers return selected ledgers
	GetWonLedgers() []*pb.LotteryLx

	// choose farmer vote candidate, put into LotteryFx.Candidate
	// param[0] stands for how many workers work together at the same time
	ChooseFarmerVote(int)

	// GetFxs return farmer lottery list
	GetFxs() []*pb.LotteryFx

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

// GetEndR return finally R
func (s *InMemoryStorage) GetEndR() uint64 {
	s.Lock()
	defer s.Unlock()

	return s.r
}

// SelectCandidate select some(winnerNum) candidate to record the blockchain
// and give back coinbase transaction(lepuscoin) for a period
func (s *InMemoryStorage) SelectCandidate(winnerNum int) {
	s.Lock()
	defer s.Unlock()

	lenCandidate := len(s.lxs)
	lotteryLogger.Debugf("ledger candidates len: %d, ledger seat: %d", lenCandidate, winnerNum)

	// all the candidate are been selected
	if lenCandidate <= winnerNum {
		for idx, _ := range s.lxs {
			s.lxs[idx].Won = true
		}
	} else {
		for idx, _ := range s.lxs {
			s.lxs[idx].Dist = s.lxs[idx].Value ^ s.r
			lotteryLogger.Debugf("calc ledger %d: %s, %d, %d", idx, s.lxs[idx].Lid, s.lxs[idx].Value, s.lxs[idx].Dist)
		}

		sort.Sort(client.LotteryLxs(s.lxs))
		for idx := 0; idx < winnerNum; idx++ {
			s.lxs[idx].Won = true
		}
	}
}

// GetWonLedgers return selected ledgers
func (s *InMemoryStorage) GetWonLedgers() []*pb.LotteryLx {
	s.Lock()
	defer s.Unlock()

	lxs := make([]*pb.LotteryLx, 0)
	for _, lx := range s.lxs {
		if lx.Won {
			lxs = append(lxs, lx)
		}
	}

	return lxs
}

// ChooseFarmerVote
func (s *InMemoryStorage) ChooseFarmerVote(worker int) {
	s.Lock()
	defer s.Unlock()

	if len(s.lxs) <= 0 {
		return
	}

	sema := semaphore.NewSemaphore(worker)
	for idxF, fx := range s.fxs {
		sema.Acquire()
		go func(idxF int, fx *pb.LotteryFx) {
			defer sema.Release()

			tmpLxs := cloneLxs(s.lxs)
			for idxL, lx := range tmpLxs {
				tmpLxs[idxL].Dist = lx.Value ^ fx.Value
			}

			sort.Sort(client.LotteryLxs(tmpLxs))
			s.fxs[idxF].Candidate = tmpLxs[0].Lid
		}(idxF, fx)
	}
}

func (s *InMemoryStorage) GetFxs() []*pb.LotteryFx {
	s.Lock()
	defer s.Unlock()

	return s.fxs
}

// persist current lottery history to a backend, maybe a db or something else
func (s *InMemoryStorage) Persist() error {
	return nil
}

// deep copy form src to dst
func cloneLxs(src []*pb.LotteryLx) (dst []*pb.LotteryLx) {
	dst = make([]*pb.LotteryLx, 0)
	for _, lx := range src {
		dst = append(dst, &pb.LotteryLx{
			Lid:   lx.Lid,
			Value: lx.Value,
			Dist:  lx.Dist,
			Won:   lx.Won,
		})
	}

	return
}
