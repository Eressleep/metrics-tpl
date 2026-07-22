package agent

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/Eressleep/metrics-tpl/internal/proto"
	"github.com/Eressleep/metrics-tpl/internal/utils"
	"google.golang.org/grpc"
	"google.golang.org/grpc/connectivity"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
)

type GRPCClient struct {
	conn       *grpc.ClientConn
	client     proto.MetricsClient
	serverAddr string
	localIP    string
	timeout    time.Duration
}

func NewGRPCClient(serverAddr string, timeout time.Duration) (*GRPCClient, error) {
	// Создаем клиент с NewClient
	conn, err := grpc.NewClient(
		serverAddr,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create gRPC client: %w", err)
	}

	// Ждем, пока соединение будет готово
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	// Проверяем состояние соединения
	for {
		state := conn.GetState()
		if state == connectivity.Ready {
			break
		}
		if state == connectivity.TransientFailure || state == connectivity.Shutdown {
			conn.Close()
			return nil, fmt.Errorf("connection failed: state=%s", state)
		}
		if !conn.WaitForStateChange(ctx, state) {
			conn.Close()
			return nil, fmt.Errorf("connection timeout after %v", timeout)
		}
	}

	return &GRPCClient{
		conn:       conn,
		client:     proto.NewMetricsClient(conn),
		serverAddr: serverAddr,
		localIP:    utils.GetOutboundIP(),
		timeout:    timeout,
	}, nil
}

func (c *GRPCClient) SendMetrics(metrics []proto.Metric) error {
	ctx, cancel := context.WithTimeout(context.Background(), c.timeout)
	defer cancel()

	// Добавляем IP адрес в метаданные
	md := metadata.Pairs("x-real-ip", c.localIP)
	ctx = metadata.NewOutgoingContext(ctx, md)

	// Конвертируем слайс значений в слайс указателей
	metricsPtr := make([]*proto.Metric, len(metrics))
	for i := range metrics {
		metricsPtr[i] = &metrics[i]
	}

	req := &proto.UpdateMetricsRequest{
		Metrics: metricsPtr,
	}

	resp, err := c.client.UpdateMetrics(ctx, req)
	if err != nil {
		return fmt.Errorf("failed to send metrics via gRPC: %w", err)
	}

	log.Printf("Successfully sent %d metrics via gRPC", len(metrics))
	_ = resp

	return nil
}

func (c *GRPCClient) Close() error {
	if c.conn != nil {
		return c.conn.Close()
	}
	return nil
}
