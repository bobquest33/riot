package fsm

import (
	"encoding/json"
	"io"
	"sync"
	"time"

	"github.com/hashicorp/go-msgpack/codec"
	"github.com/hashicorp/raft"
	"github.com/laohanlinux/go-logger/logger"
	"github.com/laohanlinux/riot/command"
)

// StorageFSM is an implememtation of the FSM interfacec, and just
// storage the key/value logs sequentially
type StorageFSM struct {
	sync.Mutex
	logs [][]byte

	cache map[string][]byte
}

// Apply is noly call in out with master leader
// log format: json
// {"cmd":op, "key":key, "value": value}
// TODO
// use protocol buffer instead of json format
func (s *StorageFSM) Apply(log *raft.Log) interface{} {
	s.Lock()
	defer s.Unlock()

	logger.Info("Excute StorageFSM.Apply ...")
	var cmdData command.Comand
	if err := json.Unmarshal(log.Data, &cmdData); err != nil {
		logger.Fatal(err)
	}

	switch cmdData.Op {
	// Get Operation should not in here
	//case command.CmdGet:
	case command.CmdSet:
		s.cache[cmdData.Key] = cmdData.Value
	case command.CmdDel:
		delete(s.cache, cmdData.Key)
	}

	// storage internel cache
	s.logs = append(s.logs, log.Data)
	return len(s.logs)
}

func (s *StorageFSM) Snapshot() (raft.FSMSnapshot, error) {
	s.Lock()
	defer s.Unlock()
	logger.Info("Excute StorageFSM.Snapshot ...")
	return &StorageSnapshot{s.logs, len(s.logs)}, nil
}

// restore data from persit location
func (s *StorageFSM) Restore(inp io.ReadCloser) error {
	logger.Info("Excute StorageFSN.Restore ...")
	s.Lock()
	defer s.Unlock()
	defer inp.Close()

	hd := codec.MsgpackHandle{}
	dec := codec.NewDecoder(inp, &hd)

	s.logs = nil
	return dec.Decode(&s.logs)
}

type StorageSnapshot struct {
	logs     [][]byte
	maxIndex int
}

func (s *StorageSnapshot) Persist(sink raft.SnapshotSink) error {
	logger.Info("Excute StorageSnapshot.Persist ... ")

	hd := codec.MsgpackHandle{}
	enc := codec.NewEncoder(sink, &hd)
	logger.Info(len(s.logs))

	if err := enc.Encode(s.logs[:s.maxIndex]); err != nil {
		sink.Close()
		return err
	}

	logger.Info(len(s.logs))
	sink.Close()
	return nil
}

func (s *StorageSnapshot) Release() {
	logger.Info("Excute StorageSnapshot.Release ...")
}

// Reeturn configurations optimized for in-memeory

func inmemConfig() *raft.Log {
	conf := raft.DefaultConfig()
	conf.HeartbeatTimeout = 50 * time.Millisecond
	conf.ElectionTimeout = 50 * time.Millisecond
	conf.LeaderLeaseTimeout = 50 * time.Millisecond
	conf.CommitTimeout = time.Millisecond
	//conf.Logger = log.New(&testLoggerAdapter, "", 0)
	return conf
}