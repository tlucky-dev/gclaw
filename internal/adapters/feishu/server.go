package feishu

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"gclaw/pkg/types"
)

// Server 飞书 HTTP 服务器
type Server struct {
	adapter  *Adapter
	server   *http.Server
	addr     string
}

// NewServer 创建飞书 HTTP 服务器
func NewServer(adapter *Adapter, addr string) *Server {
	return &Server{
		adapter: adapter,
		addr:    addr,
	}
}

// Start 启动 HTTP 服务器
func (s *Server) Start(ctx context.Context) error {
	mux := http.NewServeMux()
	
	// 注册 webhook 处理路由
	mux.HandleFunc("/webhook/feishu", s.adapter.HandleWebhook)
	
	s.server = &http.Server{
		Addr:         s.addr,
		Handler:      mux,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	go func() {
		<-ctx.Done()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := s.server.Shutdown(shutdownCtx); err != nil {
			fmt.Printf("关闭服务器错误：%v\n", err)
		}
	}()

	fmt.Printf("飞书适配器 HTTP 服务器启动在 %s\n", s.addr)
	fmt.Printf("Webhook 端点：http://%s/webhook/feishu\n", s.addr)

	if err := s.server.ListenAndServe(); err != http.ErrServerClosed {
		return fmt.Errorf("HTTP 服务器错误：%w", err)
	}

	return nil
}

// Stop 停止 HTTP 服务器
func (s *Server) Stop(ctx context.Context) error {
	if s.server == nil {
		return nil
	}
	
	shutdownCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	
	return s.server.Shutdown(shutdownCtx)
}

// GetAdapter 获取适配器实例
func (s *Server) GetAdapter() *Adapter {
	return s.adapter
}

// ProcessMessages 处理消息循环
func (s *Server) ProcessMessages(ctx context.Context, handler func(types.Message) error) {
	msgChan := s.adapter.StartMessageChannel(ctx)
	
	for {
		select {
		case <-ctx.Done():
			fmt.Println("消息处理循环退出")
			return
		case msg, ok := <-msgChan:
			if !ok {
				fmt.Println("消息通道已关闭")
				return
			}
			
			if err := handler(msg); err != nil {
				fmt.Printf("处理消息错误：%v\n", err)
			}
		}
	}
}
