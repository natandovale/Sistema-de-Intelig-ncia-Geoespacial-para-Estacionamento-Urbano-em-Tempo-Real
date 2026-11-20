package redis

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/spot-sense/backend/internal/domain"
)

type cacheEntry struct {
	data      []byte
	expiresAt time.Time
}

// MemorySpotCache implementa domain.SpotCache em memória RAM com expiração por TTL.
type MemorySpotCache struct {
	mu    sync.RWMutex
	ttl   time.Duration
	items map[string]cacheEntry
}

// NewMemorySpotCache cria um cache em memória thread-safe com o TTL informado.
func NewMemorySpotCache(ttl time.Duration) *MemorySpotCache {
	if ttl <= 0 {
		ttl = 10 * time.Minute
	}
	return &MemorySpotCache{
		ttl:   ttl,
		items: make(map[string]cacheEntry),
	}
}

func (m *MemorySpotCache) Get(_ context.Context, key string) (*domain.ZoneAssessment, bool, error) {
	m.mu.RLock()
	entry, ok := m.items[key]
	m.mu.RUnlock()

	if !ok || time.Now().After(entry.expiresAt) {
		return nil, false, nil
	}

	var assessment domain.ZoneAssessment
	if err := json.Unmarshal(entry.data, &assessment); err != nil {
		return nil, false, err
	}
	return &assessment, true, nil
}

func (m *MemorySpotCache) Set(_ context.Context, key string, value domain.ZoneAssessment) error {
	payload, err := json.Marshal(value)
	if err != nil {
		return err
	}

	m.mu.Lock()
	m.items[key] = cacheEntry{
		data:      payload,
		expiresAt: time.Now().Add(m.ttl),
	}
	m.mu.Unlock()
	return nil
}

// DialFunc define a função de conexão para permitir testes com net.Pipe sem sockets de rede.
type DialFunc func(ctx context.Context, network, addr string) (net.Conn, error)

// RedisSpotCache implementa domain.SpotCache comunicando-se com o Redis via protocolo RESP2.
type RedisSpotCache struct {
	addr    string
	ttl     time.Duration
	timeout time.Duration
	dial    DialFunc
}

// NewRedisSpotCache cria um cliente de cache Redis apontando para host:port (ex: "localhost:6379").
func NewRedisSpotCache(addr string, ttl time.Duration) *RedisSpotCache {
	addr = strings.TrimPrefix(addr, "redis://")
	if idx := strings.Index(addr, "/"); idx != -1 {
		addr = addr[:idx]
	}
	if ttl <= 0 {
		ttl = 10 * time.Minute
	}
	timeout := 2 * time.Second
	dialer := &net.Dialer{Timeout: timeout}
	return &RedisSpotCache{
		addr:    addr,
		ttl:     ttl,
		timeout: timeout,
		dial:    dialer.DialContext,
	}
}

// WithDialer permite substituir o dialer TCP (útil para testes unitários isolados via net.Pipe).
func (r *RedisSpotCache) WithDialer(dial DialFunc) *RedisSpotCache {
	r.dial = dial
	return r
}

func (r *RedisSpotCache) Get(ctx context.Context, key string) (*domain.ZoneAssessment, bool, error) {
	conn, err := r.dial(ctx, "tcp", r.addr)
	if err != nil {
		return nil, false, err
	}
	defer conn.Close()
	_ = conn.SetDeadline(time.Now().Add(r.timeout))

	// Comando RESP: *2\r\n$3\r\nGET\r\n$<len>\r\n<key>\r\n
	cmd := fmt.Sprintf("*2\r\n$3\r\nGET\r\n$%d\r\n%s\r\n", len(key), key)
	if _, err := conn.Write([]byte(cmd)); err != nil {
		return nil, false, err
	}

	reader := bufio.NewReader(conn)
	header, err := reader.ReadString('\n')
	if err != nil {
		return nil, false, err
	}
	header = strings.TrimSpace(header)
	if header == "$-1" || !strings.HasPrefix(header, "$") {
		return nil, false, nil
	}

	length, err := strconv.Atoi(header[1:])
	if err != nil || length <= 0 {
		return nil, false, err
	}

	buf := make([]byte, length+2) // inclui \r\n
	if _, err := io.ReadFull(reader, buf); err != nil {
		return nil, false, err
	}

	var assessment domain.ZoneAssessment
	if err := json.Unmarshal(buf[:length], &assessment); err != nil {
		return nil, false, err
	}
	return &assessment, true, nil
}

func (r *RedisSpotCache) Set(ctx context.Context, key string, value domain.ZoneAssessment) error {
	payload, err := json.Marshal(value)
	if err != nil {
		return err
	}

	conn, err := r.dial(ctx, "tcp", r.addr)
	if err != nil {
		return err
	}
	defer conn.Close()
	_ = conn.SetDeadline(time.Now().Add(r.timeout))

	ttlSeconds := strconv.Itoa(int(r.ttl.Seconds()))
	if ttlSeconds == "0" {
		ttlSeconds = "1"
	}

	// Comando RESP: *5\r\n$3\r\nSET\r\n$<klen>\r\n<key>\r\n$<vlen>\r\n<val>\r\n$2\r\nEX\r\n$<tlen>\r\n<ttl>\r\n
	cmd := fmt.Sprintf(
		"*5\r\n$3\r\nSET\r\n$%d\r\n%s\r\n$%d\r\n%s\r\n$2\r\nEX\r\n$%d\r\n%s\r\n",
		len(key), key,
		len(payload), string(payload),
		len(ttlSeconds), ttlSeconds,
	)

	if _, err := conn.Write([]byte(cmd)); err != nil {
		return err
	}

	reader := bufio.NewReader(conn)
	resp, err := reader.ReadString('\n')
	if err != nil {
		return err
	}
	if !strings.HasPrefix(resp, "+OK") {
		return fmt.Errorf("unexpected redis SET response: %s", strings.TrimSpace(resp))
	}
	return nil
}
