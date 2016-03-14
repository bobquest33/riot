package cluster

import (
	"io/ioutil"
	"time"

	"github.com/hashicorp/raft"
	"github.com/laohanlinux/go-logger/logger"
	"github.com/laohanlinux/riot/config"
	"github.com/laohanlinux/riot/fsm"
)

type Cluster struct {
	Dir         string
	R           *raft.Raft
	Stores      *raft.InmemStore
	FSM         *fsm.StorageFSM
	Snap        raft.SnapshotStore
	Tran        raft.Transport
	PeerStorage raft.PeerStore
}

var rCluster *Cluster

// SingleCluster .
func SingleCluster() *Cluster {
	return rCluster
}

// NewCluster ...
func NewCluster(cfg *config.Configure, conf *raft.Config) *Cluster {
	if rCluster != nil {
		return rCluster
	}
	rCluster = &Cluster{}

	// Setup the restores and transports
	dir, err := ioutil.TempDir("", "raft")
	if err != nil {
		logger.Fatal(err)
	}

	// create log store dir, may be use disk
	store := raft.NewInmemStore()
	rCluster.Dir = dir
	// for log and config storage
	rCluster.Stores = store
	rCluster.FSM = fsm.NewStorageFSM()

	//create snap dir
	_, snap := fileSnap()
	rCluster.Snap = snap

	// create transport
	tran, err := raft.NewTCPTransport(cfg.RaftC.Addr+":"+cfg.RaftC.Port, nil, 3, 2*time.Second, nil)
	if err != nil {
		logger.Fatal(err)
	}
	rCluster.Tran = tran

	// NewJSONPeers create a new JSONPees store
	peerStorage := raft.NewJSONPeers(cfg.RaftC.PeerStorage, tran)
	ps, err := peerStorage.Peers()
	if err != nil {
		logger.Fatal(err)
	}
	for _, peer := range cfg.RaftC.Peers {
		ps = raft.AddUniquePeer(ps, peer)
	}
	if cfg.RaftC.EnableSingleNode && len(ps) > 0 {
		conf.EnableSingleNode = false
	}
	peerStorage.SetPeers(ps)
	rCluster.PeerStorage = peerStorage
	// Wait the transport
	r, err := raft.NewRaft(conf, rCluster.FSM, store, store, snap, peerStorage, tran)
	if err != nil {
		logger.Fatal(err)
	}
	rCluster.R = r

	go rCluster.LeaderChange()
	return rCluster
}

func (c *Cluster) Join() {

}

func (c *Cluster) LeaderChange() {
	for {
		logger.Info("leader is: ", c.R.Leader())
		<-c.R.LeaderCh()
		logger.Info("leader change to ", c.R.Leader())
	}
}

func (c *Cluster) Leader() string {
	return c.Leader()
}

func (c *Cluster) Get(key string) ([]byte, error) {
	return c.FSM.Get(key)
}
func fileSnap() (string, *raft.FileSnapshotStore) {
	dir, err := ioutil.TempDir("", "raft")
	if err != nil {
		panic(err)
	}

	logger.Info("snap save dir:", dir)
	snap, err := raft.NewFileSnapshotStore(dir, 3, nil)
	if err != nil {
		panic(err)
	}

	return dir, snap
}
