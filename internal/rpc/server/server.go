package server

import (
	"context"
	"fmt"
	"net"

	"go.uber.org/zap"
	"google.golang.org/grpc"

	"github.com/AlexeySadkovich/eldberg/config"
	"github.com/AlexeySadkovich/eldberg/internal/rpc/pb"
)

type Server struct {
	server *grpc.Server
	port   int

	logger *zap.SugaredLogger
}

func New(config *config.Config, logger *zap.SugaredLogger) *Server {
	rpcServer := grpc.NewServer()
	port := config.Node.Port

	server := &Server{
		server: rpcServer,
		port:   port,
	}

	instance := &NodeServiceController{}
	pb.RegisterNodeServiceServer(rpcServer, instance)

	return server
}

func (s *Server) Start(ctx context.Context) {
	addr := fmt.Sprintf(":%d", s.port)

	conn, err := net.Listen("tcp", addr)
	if err != nil {
		s.logger.Errorf("listen failed: %w", err)
		return
	}

	if err := s.server.Serve(conn); err != nil {
		s.logger.Errorf("serving failed: %w", err)
		return
	}
}

func (s *Server) Stop() {
	s.server.GracefulStop()
}