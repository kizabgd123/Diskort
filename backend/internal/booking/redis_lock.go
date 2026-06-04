package booking

import (
	"bufio"
	"context"
	"fmt"
	"net"
	"os"
	"strconv"
	"strings"
	"time"
)

type RedisLockManager struct {
	Addr     string
	Password string
	Timeout  time.Duration
}

func NewRedisLockManagerFromEnv() *RedisLockManager {
	addr := os.Getenv("REDIS_URL")
	if addr == "" {
		addr = "127.0.0.1:6379"
	}
	addr = strings.TrimPrefix(addr, "redis://")
	return &RedisLockManager{Addr: addr, Password: os.Getenv("REDIS_PASSWORD"), Timeout: 2 * time.Second}
}

func (r *RedisLockManager) Acquire(ctx context.Context, key, owner string, ttl time.Duration) (bool, error) {
	result, err := r.command(ctx, "SET", key, owner, "NX", "PX", strconv.FormatInt(ttl.Milliseconds(), 10))
	if err != nil {
		return false, err
	}
	return result == "OK", nil
}

func (r *RedisLockManager) Release(ctx context.Context, key, owner string) error {
	const script = `if redis.call("get", KEYS[1]) == ARGV[1] then return redis.call("del", KEYS[1]) else return 0 end`
	_, err := r.command(ctx, "EVAL", script, "1", key, owner)
	return err
}

func (r *RedisLockManager) command(ctx context.Context, args ...string) (string, error) {
	dialer := net.Dialer{Timeout: r.Timeout}
	conn, err := dialer.DialContext(ctx, "tcp", r.Addr)
	if err != nil {
		return "", err
	}
	defer conn.Close()
	if deadline, ok := ctx.Deadline(); ok {
		_ = conn.SetDeadline(deadline)
	} else {
		_ = conn.SetDeadline(time.Now().Add(r.Timeout))
	}
	if r.Password != "" {
		if _, err := writeRESP(conn, "AUTH", r.Password); err != nil {
			return "", err
		}
		if _, err := readRESP(bufio.NewReader(conn)); err != nil {
			return "", err
		}
	}
	if _, err := writeRESP(conn, args...); err != nil {
		return "", err
	}
	return readRESP(bufio.NewReader(conn))
}

func writeRESP(conn net.Conn, args ...string) (int, error) {
	var b strings.Builder
	fmt.Fprintf(&b, "*%d\r\n", len(args))
	for _, arg := range args {
		fmt.Fprintf(&b, "$%d\r\n%s\r\n", len(arg), arg)
	}
	return conn.Write([]byte(b.String()))
}

func readRESP(r *bufio.Reader) (string, error) {
	prefix, err := r.ReadByte()
	if err != nil {
		return "", err
	}
	line, err := r.ReadString('\n')
	if err != nil {
		return "", err
	}
	line = strings.TrimSuffix(strings.TrimSuffix(line, "\n"), "\r")
	switch prefix {
	case '+', ':':
		return line, nil
	case '$':
		length, err := strconv.Atoi(line)
		if err != nil || length < 0 {
			return "", err
		}
		buf := make([]byte, length+2)
		if _, err := r.Read(buf); err != nil {
			return "", err
		}
		return string(buf[:length]), nil
	case '-':
		return "", fmt.Errorf("redis error: %s", line)
	default:
		return "", fmt.Errorf("unexpected redis response prefix %q", prefix)
	}
}
