package autotemplate

import (
	"bufio"
	"os"
	"strings"
	"testing"
)

// BenchmarkRealLogs tests performance with real-world-like log messages
func BenchmarkRealLogs(b *testing.B) {
	cfg := DefaultConfig()
	cfg.Shards = 4
	miner := NewShardedMiner(cfg)
	
	// Real-world-like log messages with various patterns
	messages := []string{
		"INFO [2025-01-15 10:23:45] user admin logged in from 192.168.1.100",
		"INFO [2025-01-15 10:23:46] user john.doe logged in from 10.0.0.23",
		"ERROR [2025-01-15 10:23:47] failed to connect to database server db-prod-01 after 3 retries",
		"ERROR [2025-01-15 10:23:48] failed to connect to database server db-prod-02 after 5 retries",
		"DEBUG [2025-01-15 10:23:49] cache hit for key user:session:abc123def456",
		"DEBUG [2025-01-15 10:23:50] cache hit for key user:session:xyz789ghi012",
		"INFO [2025-01-15 10:23:51] HTTP GET /api/v1/users/12345 200 OK 45ms",
		"INFO [2025-01-15 10:23:52] HTTP GET /api/v1/users/67890 200 OK 38ms",
		"INFO [2025-01-15 10:23:53] HTTP POST /api/v1/orders 201 Created 123ms",
		"WARN [2025-01-15 10:23:54] rate limit exceeded for client 192.168.1.150 endpoint /api/v1/search",
		"WARN [2025-01-15 10:23:55] rate limit exceeded for client 10.0.5.72 endpoint /api/v1/query",
		"INFO [2025-01-15 10:23:56] background job invoice-processing-worker-3 started",
		"INFO [2025-01-15 10:23:57] background job email-sender-worker-7 started",
		"ERROR [2025-01-15 10:23:58] payment gateway timeout for transaction txn_9f8e7d6c5b4a after 30000ms",
		"ERROR [2025-01-15 10:23:59] payment gateway timeout for transaction txn_1a2b3c4d5e6f after 30000ms",
		"DEBUG [2025-01-15 10:24:00] SQL query SELECT * FROM users WHERE id=? took 12ms",
		"DEBUG [2025-01-15 10:24:01] SQL query SELECT * FROM orders WHERE status=? took 89ms",
		"INFO [2025-01-15 10:24:02] message received on queue kafka://orders-topic partition 5 offset 123456",
		"INFO [2025-01-15 10:24:03] message received on queue kafka://orders-topic partition 2 offset 789012",
		"WARN [2025-01-15 10:24:04] authentication failed for user@example.com invalid password",
		"WARN [2025-01-15 10:24:05] authentication failed for admin@test.com invalid password",
		"INFO [2025-01-15 10:24:06] SSL certificate expires in 45 days for domain api.example.com",
		"INFO [2025-01-15 10:24:07] SSL certificate expires in 12 days for domain cdn.example.com",
		"ERROR [2025-01-15 10:24:08] disk usage 95% on volume /mnt/data01 threshold exceeded",
		"ERROR [2025-01-15 10:24:09] disk usage 97% on volume /mnt/data05 threshold exceeded",
	}
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		msg := messages[i%len(messages)]
		miner.Add(msg)
	}
	
	// Report EPS
	b.ReportMetric(float64(b.N)/b.Elapsed().Seconds(), "eps")
}

// BenchmarkLargeDataset tests with a larger variety of messages
func BenchmarkLargeDataset(b *testing.B) {
	cfg := DefaultConfig()
	cfg.Shards = 8
	miner := NewShardedMiner(cfg)
	
	// Generate 1000 diverse messages
	messages := make([]string, 1000)
	severities := []string{"DEBUG", "INFO", "WARN", "ERROR"}
	services := []string{"api-server", "auth-service", "payment-gateway", "email-worker", "cache-manager"}
	actions := []string{"started", "stopped", "failed", "completed", "retrying"}
	
	for i := 0; i < 1000; i++ {
		sev := severities[i%len(severities)]
		svc := services[(i/10)%len(services)]
		act := actions[(i/25)%len(actions)]
		num := i * 123
		messages[i] = sev + " service " + svc + " " + act + " with code " + string(rune('0'+num%10))
	}
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		msg := messages[i%len(messages)]
		miner.Add(msg)
	}
	
	b.ReportMetric(float64(b.N)/b.Elapsed().Seconds(), "eps")
}

// BenchmarkConcurrentAdd tests concurrent processing
func BenchmarkConcurrentAdd(b *testing.B) {
	cfg := DefaultConfig()
	cfg.Shards = 8
	miner := NewShardedMiner(cfg)
	
	messages := []string{
		"INFO user logged in from 192.168.1.1",
		"ERROR database connection failed",
		"DEBUG cache miss for key abc123",
		"WARN rate limit exceeded for client",
	}
	
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			msg := messages[i%len(messages)]
			miner.Add(msg)
			i++
		}
	})
	
	b.ReportMetric(float64(b.N)/b.Elapsed().Seconds(), "eps")
}

// TestLoadFromFile tests loading a real log file if available
func TestLoadFromFile(t *testing.T) {
	// Skip in CI or if file doesn't exist
	testFile := "testdata/sample.log"
	if _, err := os.Stat(testFile); os.IsNotExist(err) {
		t.Skip("test log file not available")
	}
	
	cfg := DefaultConfig()
	cfg.Shards = 4
	miner := NewShardedMiner(cfg)
	
	file, err := os.Open(testFile)
	if err != nil {
		t.Fatalf("failed to open test file: %v", err)
	}
	defer file.Close()
	
	scanner := bufio.NewScanner(file)
	count := 0
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		miner.Add(line)
		count++
	}
	
	if err := scanner.Err(); err != nil {
		t.Fatalf("error reading file: %v", err)
	}
	
	stats := miner.Stats()
	t.Logf("Processed %d lines, created %d clusters", count, stats["clusters"])
}
