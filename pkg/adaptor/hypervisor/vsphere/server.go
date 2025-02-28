package vsphere

import (
	"context"
	"log"
	"net"
	"os"
	"path/filepath"
	"sync"

	"github.com/confidential-containers/cloud-api-adaptor/pkg/adaptor/hypervisor"
	"github.com/confidential-containers/cloud-api-adaptor/pkg/podnetwork"
	"github.com/containerd/ttrpc"
	"github.com/vmware/govmomi"

	pb "github.com/kata-containers/kata-containers/src/runtime/protocols/hypervisor"
)

var logger = log.New(log.Writer(), "[helper/hypervisor] ", log.LstdFlags|log.Lmsgprefix)

type server struct {
	socketPath string

	ttRpc   *ttrpc.Server
	service pb.HypervisorService

	client *govmomi.Client

	workerNode podnetwork.WorkerNode

	readyCh  chan struct{}
	stopCh   chan struct{}
	stopOnce sync.Once
}

func NewServer(cfg hypervisor.Config, vmcfg Config, workerNode podnetwork.WorkerNode, daemonPort string) hypervisor.Server {

	logger.Printf("hypervisor config %v", cfg)
	logger.Printf("cloud config %v", vmcfg)

	govmomiClient, err := NewGovmomiClient(vmcfg)
	if err != nil {
		return nil
	}

	return &server{
		socketPath: cfg.SocketPath,
		service:    newService(govmomiClient, &vmcfg, &cfg, workerNode, cfg.PodsDir, daemonPort),
		client:     govmomiClient,
		workerNode: workerNode,
		readyCh:    make(chan struct{}),
		stopCh:     make(chan struct{}),
	}
}

func (s *server) Start(ctx context.Context) (err error) {

	ttRpc, err := ttrpc.NewServer()
	if err != nil {
		return err
	}
	s.ttRpc = ttRpc
	if err = os.MkdirAll(filepath.Dir(s.socketPath), os.ModePerm); err != nil {
		return err
	}
	if err := os.RemoveAll(s.socketPath); err != nil { // just in case socket wasn't cleaned
		return err
	}
	pb.RegisterHypervisorService(s.ttRpc, s.service)
	listener, err := net.Listen("unix", s.socketPath)
	if err != nil {
		return err
	}

	ttRpcErr := make(chan error)
	go func() {
		defer close(ttRpcErr)
		if err = s.ttRpc.Serve(ctx, listener); err != nil {
			ttRpcErr <- err
		}
	}()
	defer func() {
		newErr := s.ttRpc.Shutdown(context.Background())
		if newErr != nil && err == nil {
			err = newErr
		}
	}()

	close(s.readyCh)

	select {
	case <-ctx.Done():
		err = s.Shutdown()
	case <-s.stopCh:
	case err = <-ttRpcErr:
	}
	return err
}

func (s *server) Shutdown() error {

	DeleteGovmomiClient(s.client)

	s.stopOnce.Do(func() {
		close(s.stopCh)
	})

	return nil
}

func (s *server) Ready() chan struct{} {
	return s.readyCh
}
