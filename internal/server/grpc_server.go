package server

import (
	"context"
	"fmt"
	"net"
	"strings"

	"github.com/Eressleep/metrics-tpl/internal/proto"
	"github.com/Eressleep/metrics-tpl/internal/storage"
	"github.com/Eressleep/metrics-tpl/internal/utils"
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
	grpcServer    *grpc.Server
	listener      net.Listener
}

func NewGRPCServer(storage storage.Storage, logger *zap.Logger, trustedSubnet string) *GRPCServer {
	return &GRPCServer{
		storage:       storage,
		logger:        logger,
		trustedSubnet: trustedSubnet,
	}
}

// UpdateMetrics реализует метод gRPC сервиса
func (s *GRPCServer) UpdateMetrics(ctx context.Context, req *proto.UpdateMetricsRequest) (*proto.UpdateMetricsResponse, error) {
	// Проверяем доверенную подсеть
	if err := s.checkTrustedSubnet(ctx); err != nil {
		return nil, err
	}

	if len(req.Metrics) == 0 {
		return &proto.UpdateMetricsResponse{}, nil
	}

	// Конвертируем proto метрики в storage метрики
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

	// Сохраняем метрики
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

// checkTrustedSubnet проверяет, что IP адрес клиента входит в доверенную подсеть
func (s *GRPCServer) checkTrustedSubnet(ctx context.Context) error {
	// Если доверенная подсеть не указана, пропускаем проверку
	if s.trustedSubnet == "" {
		return nil
	}

	// Получаем IP адрес из метаданных
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		s.logger.Warn("No metadata in gRPC request")
		return status.Errorf(codes.PermissionDenied, "metadata not found")
	}

	// Проверяем X-Real-IP из метаданных
	xRealIPs := md.Get("x-real-ip")
	if len(xRealIPs) == 0 {
		s.logger.Warn("X-Real-IP header missing in gRPC request")
		return status.Errorf(codes.PermissionDenied, "X-Real-IP header required")
	}
	clientIP := xRealIPs[0]

	// Если IP не получен, пытаемся получить его из peer
	if clientIP == "" {
		p, ok := peer.FromContext(ctx)
		if ok && p.Addr != nil {
			// Извлекаем IP из адреса
			clientIP = strings.Split(p.Addr.String(), ":")[0]
		}
	}

	if clientIP == "" {
		s.logger.Warn("Unable to determine client IP")
		return status.Errorf(codes.PermissionDenied, "unable to determine client IP")
	}

	// Проверяем IP в доверенной подсети
	allowed, err := utils.IPInCIDR(clientIP, s.trustedSubnet)
	if err != nil {
		s.logger.Error("Failed to check IP in CIDR",
			zap.String("ip", clientIP),
			zap.String("subnet", s.trustedSubnet),
			zap.Error(err))
		return status.Errorf(codes.Internal, "failed to check IP: %v", err)
	}

	if !allowed {
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

// Start запускает gRPC сервер
func (s *GRPCServer) Start(addr string) error {
	lis, err := net.Listen("tcp", addr)
	if err != nil {
		return fmt.Errorf("failed to listen: %w", err)
	}
	s.listener = lis

	s.grpcServer = grpc.NewServer(
		grpc.UnaryInterceptor(s.unaryInterceptor),
	)
	proto.RegisterMetricsServer(s.grpcServer, s)

	s.logger.Info("Starting gRPC server",
		zap.String("address", addr))

	return s.grpcServer.Serve(lis)
}

// unaryInterceptor перехватывает и логирует gRPC запросы
func (s *GRPCServer) unaryInterceptor(
	ctx context.Context,
	req interface{},
	info *grpc.UnaryServerInfo,
	handler grpc.UnaryHandler,
) (interface{}, error) {
	// Логируем запрос
	s.logger.Debug("gRPC request",
		zap.String("method", info.FullMethod))

	// Вызываем обработчик
	return handler(ctx, req)
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
