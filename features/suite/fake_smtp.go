//go:build bdd

package suite

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"net"
	"strings"
	"sync"
	"time"
)

// ReceivedEmail описывает письмо, принятое FakeSMTPServer.
type ReceivedEmail struct {
	From      string
	To        []string
	Data      string
	Timestamp time.Time
}

// FakeSMTPServer — минимальный SMTP-сервер для BDD тестов.
// Принимает плейн-текст SMTP диалог (HELO/EHLO, MAIL FROM, RCPT TO, DATA, QUIT)
// и сохраняет каждое письмо в памяти.
type FakeSMTPServer struct {
	listener net.Listener
	host     string
	port     int

	mu       sync.Mutex
	messages []ReceivedEmail
	closed   bool
	wg       sync.WaitGroup
}

// NewFakeSMTPServer поднимает SMTP-сервер на свободном порту 127.0.0.1.
func NewFakeSMTPServer() (*FakeSMTPServer, error) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return nil, fmt.Errorf("listen smtp: %w", err)
	}
	addr := ln.Addr().(*net.TCPAddr)

	s := &FakeSMTPServer{
		listener: ln,
		host:     addr.IP.String(),
		port:     addr.Port,
	}

	s.wg.Add(1)
	go s.acceptLoop()

	return s, nil
}

// Host возвращает hostname слушающего сокета.
func (s *FakeSMTPServer) Host() string { return s.host }

// Port возвращает TCP-порт слушающего сокета.
func (s *FakeSMTPServer) Port() int { return s.port }

// Messages возвращает копию принятых писем.
func (s *FakeSMTPServer) Messages() []ReceivedEmail {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]ReceivedEmail, len(s.messages))
	copy(out, s.messages)
	return out
}

// Reset очищает накопленные письма.
func (s *FakeSMTPServer) Reset() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.messages = nil
}

// Stop останавливает listener и ждёт завершения handler-горутин.
func (s *FakeSMTPServer) Stop() error {
	s.mu.Lock()
	if s.closed {
		s.mu.Unlock()
		return nil
	}
	s.closed = true
	s.mu.Unlock()

	err := s.listener.Close()
	s.wg.Wait()
	if err != nil && !errors.Is(err, net.ErrClosed) {
		return err
	}
	return nil
}

func (s *FakeSMTPServer) acceptLoop() {
	defer s.wg.Done()
	for {
		conn, err := s.listener.Accept()
		if err != nil {
			s.mu.Lock()
			done := s.closed
			s.mu.Unlock()
			if done || errors.Is(err, net.ErrClosed) {
				return
			}
			// Транзиентная ошибка — продолжаем
			continue
		}
		s.wg.Add(1)
		go func(c net.Conn) {
			defer s.wg.Done()
			s.handleConn(c)
		}(conn)
	}
}

func (s *FakeSMTPServer) handleConn(conn net.Conn) {
	defer func() {
		//nolint:errcheck // best-effort close
		_ = conn.Close()
	}()

	reader := bufio.NewReader(conn)
	writer := bufio.NewWriter(conn)

	write := func(line string) error {
		if _, err := writer.WriteString(line + "\r\n"); err != nil {
			return err
		}
		return writer.Flush()
	}

	if err := write("220 fake-smtp ready"); err != nil {
		return
	}

	var (
		from string
		to   []string
	)

	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			if !errors.Is(err, io.EOF) {
				// log? в тестах не критично
			}
			return
		}
		cmd := strings.TrimRight(line, "\r\n")
		upper := strings.ToUpper(cmd)

		switch {
		case strings.HasPrefix(upper, "HELO"), strings.HasPrefix(upper, "EHLO"):
			if err := write("250 fake-smtp"); err != nil {
				return
			}
		case strings.HasPrefix(upper, "MAIL FROM"):
			from = extractAddress(cmd)
			to = nil
			if err := write("250 OK"); err != nil {
				return
			}
		case strings.HasPrefix(upper, "RCPT TO"):
			to = append(to, extractAddress(cmd))
			if err := write("250 OK"); err != nil {
				return
			}
		case upper == "DATA":
			if err := write("354 End data with <CR><LF>.<CR><LF>"); err != nil {
				return
			}
			data, err := readData(reader)
			if err != nil {
				return
			}
			s.mu.Lock()
			s.messages = append(s.messages, ReceivedEmail{
				From:      from,
				To:        append([]string(nil), to...),
				Data:      data,
				Timestamp: time.Now(),
			})
			s.mu.Unlock()
			if err := write("250 OK"); err != nil {
				return
			}
			from = ""
			to = nil
		case upper == "RSET":
			from = ""
			to = nil
			if err := write("250 OK"); err != nil {
				return
			}
		case upper == "NOOP":
			if err := write("250 OK"); err != nil {
				return
			}
		case upper == "QUIT":
			//nolint:errcheck // закрываем после QUIT
			_ = write("221 bye")
			return
		default:
			if err := write("250 OK"); err != nil {
				return
			}
		}
	}
}

// extractAddress берёт адрес из строк вида "MAIL FROM:<a@b>" или "RCPT TO:<a@b>".
func extractAddress(line string) string {
	if i := strings.Index(line, "<"); i >= 0 {
		if j := strings.Index(line[i:], ">"); j > 0 {
			return line[i+1 : i+j]
		}
	}
	if i := strings.Index(line, ":"); i >= 0 {
		return strings.TrimSpace(line[i+1:])
	}
	return ""
}

// readData читает тело письма до CRLF "." CRLF.
func readData(r *bufio.Reader) (string, error) {
	var b strings.Builder
	for {
		line, err := r.ReadString('\n')
		if err != nil {
			return "", err
		}
		trimmed := strings.TrimRight(line, "\r\n")
		if trimmed == "." {
			return b.String(), nil
		}
		// dot-stuffing: ведущая точка экранируется удвоением
		if strings.HasPrefix(trimmed, "..") {
			trimmed = trimmed[1:]
		}
		b.WriteString(trimmed)
		b.WriteString("\r\n")
	}
}
