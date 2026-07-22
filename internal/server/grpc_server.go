package server

import (
	"context"
	"fmt"
	"net"
	"strings"

	"github.com/Eressleep/metrics-tpl/internal/proto"
	"github.com/Eressleep/metrics-tpl/internal/storage"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/peer"
	"google.golang.org/grpc/status"
)

type GRPCServer struct {
	proto.UnimplementedMetricsServer
	storage       storage.Storage
	logger        *zap.Logger
	trustedSubnet string
	ipNet         *net.IPNet // Предварительно распарсенный CIDR
	grpcServer    *grpc.Server
	listener      net.Listener
}

func NewGRPCServer(storage storage.Storage, logger *zap.Logger, trustedSubnet string) *GRPCServer {
	var ipNet *net.IPNet
	if trustedSubnet != "" {
		_, parsed, err := net.ParseCIDR(trustedSubnet)
		if err != nil {
			logger.Error("Failed to parse trusted subnet CIDR",
				zap.String("subnet", trustedSubnet),
				zap.Error(err))
		} else {
			ipNet = parsed
			logger.Info("Trusted subnet parsed successfully",
				zap.String("subnet", trustedSubnet))
		}
	}

	s := &GRPCServer{
		storage:       storage,
		logger:        logger,
		trustedSubnet: trustedSubnet,
		ipNet:         ipNet,
	}

	s.grpcServer = grpc.NewServer(
		grpc.UnaryInterceptor(s.unaryInterceptor),
	)

	return s
}

// unaryInterceptor - перехватчик для проверки доверенной подсети
func (s *GRPCServer) unaryInterceptor(
	ctx context.Context,
	req interface{},
	info *grpc.UnaryServerInfo,
	handler grpc.UnaryHandler,
) (interface{}, error) {
	if err := s.checkTrustedSubnet(ctx); err != nil {
		return nil, err
	}

	s.logger.Debug("gRPC request",
		zap.String("method", info.FullMethod))

	return handler(ctx, req)
}

// checkTrustedSubnet проверяет, что IP адрес клиента входит в доверенную подсеть
func (s *GRPCServer) checkTrustedSubnet(ctx context.Context) error {
	if s.trustedSubnet == "" || s.ipNet == nil {
		return nil
	}

	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		s.logger.Warn("No metadata in gRPC request")
		return status.Errorf(codes.PermissionDenied, "metadata not found")
	}

	xRealIPs := md.Get("x-real-ip")
	if len(xRealIPs) == 0 {
		s.logger.Warn("X-Real-IP header missing in gRPC request")
		return status.Errorf(codes.PermissionDenied, "X-Real-IP header required")
	}
	clientIP := xRealIPs[0]

	if clientIP == "" {
		p, ok := peer.FromContext(ctx)
		if ok && p.Addr != nil {
			host, _, err := net.SplitHostPort(p.Addr.String())
			if err == nil {
				clientIP = host
			} else {
				clientIP = strings.Split(p.Addr.String(), ":")[0]
			}
		}
	}

	if clientIP == "" {
		s.logger.Warn("Unable to determine client IP")
		return status.Errorf(codes.PermissionDenied, "unable to determine client IP")
	}

	ip := net.ParseIP(clientIP)
	if ip == nil {
		s.logger.Warn("Invalid client IP", zap.String("ip", clientIP))
		return status.Errorf(codes.PermissionDenied, "invalid client IP: %s", clientIP)
	}

	if !s.ipNet.Contains(ip) {
		s.logger.Warn("IP not in trusted subnet",
			zap.String("ip", clientIP),
			zap.String("subnet", s.trustedSubnet))
		return status.Errorf(codes.PermissionDenied, "IP %s not in trusted subnet %s", clientIP, s.trustedSubnet)
	}

	s.logger.Debug("IP verified in trusted subnet",
		zap.String("ip", clientIP),
		zap.String("subnet", s.trustedSubnet))

	return nil
}

// UpdateMetrics реализует метод gRPC сервиса
func (s *GRPCServer) UpdateMetrics(ctx context.Context, req *proto.UpdateMetricsRequest) (*proto.UpdateMetricsResponse, error) {
	if len(req.Metrics) == 0 {
		return &proto.UpdateMetricsResponse{}, nil
	}

	storageMetrics := make([]storage.Metrics, len(req.Metrics))
	for i, m := range req.Metrics {
		var delta *int64
		var value *float64

		switch m.Type {
		case proto.Metric_GAUGE:
			val := m.Value
			value = &val
		case proto.Metric_COUNTER:
			val := m.Delta
			delta = &val
		}

		storageMetrics[i] = storage.Metrics{
			ID:    m.Id,
			MType: strings.ToLower(m.Type.String()),
			Delta: delta,
			Value: value,
		}
	}

	if err := s.storage.BatchUpdate(ctx, storageMetrics); err != nil {
		s.logger.Error("Failed to update metrics via gRPC",
			zap.Error(err),
			zap.Int("count", len(storageMetrics)))
		return nil, status.Errorf(codes.Internal, "failed to update metrics: %v", err)
	}

	s.logger.Info("Metrics updated via gRPC",
		zap.Int("count", len(storageMetrics)))

	return &proto.UpdateMetricsResponse{}, nil
}

// Start запускает gRPC сервер
func (s *GRPCServer) Start(addr string) error {
	lis, err := net.Listen("tcp", addr)
	if err != nil {
		return fmt.Errorf("failed to listen: %w", err)
	}
	s.listener = lis

	proto.RegisterMetricsServer(s.grpcServer, s)

	s.logger.Info("Starting gRPC server",
		zap.String("address", addr))

	return s.grpcServer.Serve(lis)
}

// Stop останавливает gRPC сервер
func (s *GRPCServer) Stop() {
	if s.grpcServer != nil {
		s.logger.Info("Stopping gRPC server...")
		s.grpcServer.GracefulStop()
		s.logger.Info("gRPC server stopped")
	}
}

// GetListener возвращает listener для интеграции с HTTP сервером
func (s *GRPCServer) GetListener() net.Listener {
	return s.listener
}
