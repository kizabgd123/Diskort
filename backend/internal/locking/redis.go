package locking

import (
	"bufio"
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"net"
	"net/url"
	"strconv"
	"strings"
	"time"
)

type RedisLocker struct {
	addr     string
	password string
	useTLS   bool
}

func NewRedisLocker(rawURL string) (*RedisLocker, error) {
	parsed, err := url.Parse(rawURL)
	if err != nil {
		return nil, err
	}
	if parsed.Scheme != "redis" && parsed.Scheme != "rediss" {
		return nil, errors.New("redis url must use redis:// or rediss://")
	}
	password, _ := parsed.User.Password()
	return &RedisLocker{addr: parsed.Host, password: password, useTLS: parsed.Scheme == "rediss"}, nil
}

func (l *RedisLocker) Acquire(ctx context.Context, key, token string, ttl time.Duration) error {
	resp, err := l.command(ctx, "SET", key, token, "NX", "PX", strconv.FormatInt(ttl.Milliseconds(), 10))
	if err != nil {
		return err
	}
	if resp == "$-1" {
		return fmt.Errorf("%w: %s", ErrLockHeld, key)
	}
	if !strings.Contains(resp, "+OK") {
		return fmt.Errorf("unexpected redis lock response: %s", resp)
	}
	return nil
}

func (l *RedisLocker) Release(ctx context.Context, key, token string) error {
	const script = `if redis.call("get", KEYS[1]) == ARGV[1] then return redis.call("del", KEYS[1]) else return 0 end`
	_, err := l.command(ctx, "EVAL", script, "1", key, token)
	return err
}

func (l *RedisLocker) command(ctx context.Context, args ...string) (string, error) {
	dialer := net.Dialer{Timeout: 2 * time.Second}
	conn, err := dialer.DialContext(ctx, "tcp", l.addr)
	if err != nil {
		return "", err
	}
	defer conn.Close()
	if l.useTLS {
		conn = tls.Client(conn, &tls.Config{ServerName: strings.Split(l.addr, ":")[0], MinVersion: tls.VersionTLS12})
	}
	if deadline, ok := ctx.Deadline(); ok {
		_ = conn.SetDeadline(deadline)
	} else {
		_ = conn.SetDeadline(time.Now().Add(3 * time.Second))
	}
	reader := bufio.NewReader(conn)
	if l.password != "" {
		if _, err := conn.Write([]byte(respCommand("AUTH", l.password))); err != nil {
			return "", err
		}
		if line, err := reader.ReadString('\n'); err != nil || !strings.HasPrefix(line, "+OK") {
			return "", fmt.Errorf("redis auth failed: %s %w", strings.TrimSpace(line), err)
		}
	}
	if _, err := conn.Write([]byte(respCommand(args...))); err != nil {
		return "", err
	}
	line, err := reader.ReadString('\n')
	if err != nil {
		return "", err
	}
	line = strings.TrimSpace(line)
	if strings.HasPrefix(line, "-ERR") {
		return "", errors.New(line)
	}
	return line, nil
}

func respCommand(args ...string) string {
	var b strings.Builder
	b.WriteString("*")
	b.WriteString(strconv.Itoa(len(args)))
	b.WriteString("\r\n")
	for _, arg := range args {
		b.WriteString("$")
		b.WriteString(strconv.Itoa(len(arg)))
		b.WriteString("\r\n")
		b.WriteString(arg)
		b.WriteString("\r\n")
	}
	return b.String()
}
