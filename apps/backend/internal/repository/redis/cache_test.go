package redis_test

import (
	"bufio"
	"context"
	"fmt"
	"net"
	"strings"
	"testing"
	"time"

	"github.com/spot-sense/backend/internal/domain"
	redisrepo "github.com/spot-sense/backend/internal/repository/redis"
)

func TestMemoryCache_SetAndGet_PreservesZoneAssessment(t *testing.T) {
	cache := redisrepo.NewMemorySpotCache(10 * time.Minute)
	ctx := context.Background()

	coords, _ := domain.NewCoordinates(-3.729, -38.523)
	expected := domain.ZoneAssessment{
		GridCoordinates: coords,
		RadiusMeters:    1000,
		Score:           0.85,
		Probability:     domain.ProbabilityHigh,
		TotalSupply:     120,
		EstimatedDemand: 18,
		ActiveEvents:    0,
		Roads: []domain.RoadSegment{
			{ID: "r1", Name: "Av. Beira Mar", EstimatedSpots: 120, ParkingAllowed: true},
		},
	}

	key := "spots:v1:grid:-3.729:-38.523:r:1000"

	// 1. Antes do Set -> Cache Miss
	got, found, err := cache.Get(ctx, key)
	if err != nil {
		t.Fatalf("unexpected error on cache miss: %v", err)
	}
	if found || got != nil {
		t.Fatalf("expected cache miss before Set")
	}

	// 2. Executa Set e recupera -> Cache Hit
	if err := cache.Set(ctx, key, expected); err != nil {
		t.Fatalf("unexpected error on Set: %v", err)
	}

	got, found, err = cache.Get(ctx, key)
	if err != nil {
		t.Fatalf("unexpected error on Get: %v", err)
	}
	if !found || got == nil {
		t.Fatalf("expected cache hit after Set")
	}

	if got.GridCoordinates.Latitude() != -3.729 || got.GridCoordinates.Longitude() != -38.523 {
		t.Errorf("expected coordinates (-3.729, -38.523), got (%f, %f)",
			got.GridCoordinates.Latitude(), got.GridCoordinates.Longitude())
	}
	if got.Score != expected.Score || got.Probability != expected.Probability {
		t.Errorf("expected score %f (%s), got %f (%s)",
			expected.Score, expected.Probability, got.Score, got.Probability)
	}
	if len(got.Roads) != 1 || got.Roads[0].Name != "Av. Beira Mar" {
		t.Errorf("expected road 'Av. Beira Mar', got %+v", got.Roads)
	}
}

func TestMemoryCache_ExpiredEntry_ReturnsCacheMiss(t *testing.T) {
	cache := redisrepo.NewMemorySpotCache(15 * time.Millisecond)
	ctx := context.Background()

	coords, _ := domain.NewCoordinates(-3.729, -38.523)
	key := "spots:v1:grid:-3.729:-38.523:r:1000"

	_ = cache.Set(ctx, key, domain.ZoneAssessment{
		GridCoordinates: coords,
		RadiusMeters:    1000,
		Score:           0.75,
	})

	time.Sleep(30 * time.Millisecond)

	_, found, err := cache.Get(ctx, key)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if found {
		t.Errorf("expected expired cache item to return found=false")
	}
}

func TestRedisSpotCache_SetAndGet_ViaRESP(t *testing.T) {
	store := make(map[string]string)

	pipeDialer := func(_ context.Context, _, _ string) (net.Conn, error) {
		clientConn, serverConn := net.Pipe()
		go func(c net.Conn) {
			defer c.Close()
			r := bufio.NewReader(c)
			line, err := r.ReadString('\n')
			if err != nil {
				return
			}
			if strings.HasPrefix(line, "*2") {
				// GET key
				_, _ = r.ReadString('\n') // $3
				_, _ = r.ReadString('\n') // GET
				_, _ = r.ReadString('\n') // $len
				k, _ := r.ReadString('\n')
				key := strings.TrimSpace(k)
				val, ok := store[key]
				if !ok {
					_, _ = fmt.Fprint(c, "$-1\r\n")
					return
				}
				_, _ = fmt.Fprintf(c, "$%d\r\n%s\r\n", len(val), val)
			} else if strings.HasPrefix(line, "*5") {
				// SET key val EX ttl
				_, _ = r.ReadString('\n') // $3
				_, _ = r.ReadString('\n') // SET
				_, _ = r.ReadString('\n') // $klen
				k, _ := r.ReadString('\n')
				var vlen int
				_, _ = fmt.Fscanf(r, "$%d\r\n", &vlen)
				buf := make([]byte, vlen+2)
				_, _ = r.Read(buf)
				_, _ = r.ReadString('\n') // $2
				_, _ = r.ReadString('\n') // EX
				_, _ = r.ReadString('\n') // $tlen
				_, _ = r.ReadString('\n') // ttl
				store[strings.TrimSpace(k)] = string(buf[:vlen])
				_, _ = fmt.Fprint(c, "+OK\r\n")
			}
		}(serverConn)
		return clientConn, nil
	}

	cache := redisrepo.NewRedisSpotCache("redis://localhost:6379/0", 10*time.Minute).WithDialer(pipeDialer)
	ctx := context.Background()

	coords, _ := domain.NewCoordinates(-3.729, -38.523)
	key := "spots:v1:grid:-3.729:-38.523:r:1000"

	// 1. Cache Miss
	_, found, err := cache.Get(ctx, key)
	if err != nil || found {
		t.Fatalf("expected cache miss on empty redis, found=%v, err=%v", found, err)
	}

	// 2. Set + Cache Hit
	err = cache.Set(ctx, key, domain.ZoneAssessment{
		GridCoordinates: coords,
		RadiusMeters:    1000,
		Score:           0.90,
		Probability:     domain.ProbabilityHigh,
	})
	if err != nil {
		t.Fatalf("unexpected error on Redis Set: %v", err)
	}

	got, found, err := cache.Get(ctx, key)
	if err != nil || !found || got == nil {
		t.Fatalf("expected cache hit on Redis Get, found=%v, err=%v", found, err)
	}
	if got.Score != 0.90 || got.GridCoordinates.Latitude() != -3.729 {
		t.Errorf("unexpected cached assessment: %+v", got)
	}
}

