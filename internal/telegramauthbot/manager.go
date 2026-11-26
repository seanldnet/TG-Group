package telegramauthbot

import (
	"context"
	"log/slog"
	"strings"
	"sync"

	"github.com/seanldnet/tg-group/internal/auth"
)

type Manager struct {
	mu sync.Mutex
	auth *auth.Service
	logger *slog.Logger
	cancel context.CancelFunc
	username string
	running bool
	generation uint64
}

func NewManager(service *auth.Service,logger *slog.Logger)*Manager{
	if logger==nil{logger=slog.Default()}
	return &Manager{auth:service,logger:logger}
}

func (m *Manager) Configure(token,username string){
	m.mu.Lock()
	m.generation++
	generation:=m.generation
	if m.cancel!=nil {m.cancel();m.cancel=nil}
	m.username=strings.TrimPrefix(strings.TrimSpace(username),"@")
	m.running=false
	token=strings.TrimSpace(token)
	if token!="" {
		ctx,cancel:=context.WithCancel(context.Background())
		m.cancel=cancel
		m.running=true
		poller:=New(token,m.auth,m.logger)
		go func(){poller.Run(ctx);m.mu.Lock();if m.generation==generation{m.running=false;m.cancel=nil};m.mu.Unlock()}()
	}
	m.mu.Unlock()
}

func (m *Manager) Stop(){
	m.mu.Lock()
	m.generation++
	if m.cancel!=nil {m.cancel();m.cancel=nil}
	m.running=false
	m.username=""
	m.mu.Unlock()
}

func (m *Manager) Status()(username string,running bool){
	m.mu.Lock();defer m.mu.Unlock()
	return m.username,m.running
}
