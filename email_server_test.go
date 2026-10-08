package main

import (
	"bufio"
	"fmt"
	"net"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"
)

type fakeSMTP struct {
	ln      net.Listener
	mu      sync.Mutex
	message string
}

func startFakeSMTP(t *testing.T) *fakeSMTP {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	s := &fakeSMTP{ln: ln}
	go s.serve()
	t.Cleanup(func() { _ = ln.Close() })
	return s
}

func (s *fakeSMTP) address() (string, int) {
	host, portText, _ := net.SplitHostPort(s.ln.Addr().String())
	port, _ := strconv.Atoi(portText)
	return host, port
}

func (s *fakeSMTP) serve() {
	conn, err := s.ln.Accept()
	if err != nil {
		return
	}
	defer conn.Close()
	r := bufio.NewReader(conn)
	w := bufio.NewWriter(conn)
	fmt.Fprint(w, "220 fake.local ESMTP\r\n")
	w.Flush()

	inData := false
	var data strings.Builder
	for {
		line, err := r.ReadString('\n')
		if err != nil {
			return
		}
		trimmed := strings.TrimRight(line, "\r\n")
		if inData {
			if trimmed == "." {
				s.mu.Lock()
				s.message = data.String()
				s.mu.Unlock()
				fmt.Fprint(w, "250 queued\r\n")
				w.Flush()
				inData = false
				continue
			}
			data.WriteString(line)
			continue
		}
		upper := strings.ToUpper(trimmed)
		switch {
		case strings.HasPrefix(upper, "EHLO"):
			fmt.Fprint(w, "250-fake.local\r\n250 8BITMIME\r\n")
		case strings.HasPrefix(upper, "HELO"):
			fmt.Fprint(w, "250 fake.local\r\n")
		case strings.HasPrefix(upper, "MAIL FROM:"):
			fmt.Fprint(w, "250 ok\r\n")
		case strings.HasPrefix(upper, "RCPT TO:"):
			fmt.Fprint(w, "250 ok\r\n")
		case upper == "DATA":
			fmt.Fprint(w, "354 end with .\r\n")
			inData = true
		case upper == "QUIT":
			fmt.Fprint(w, "221 bye\r\n")
			w.Flush()
			return
		default:
			fmt.Fprint(w, "250 ok\r\n")
		}
		w.Flush()
	}
}

func (s *fakeSMTP) waitMessage(t *testing.T) string {
	t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		s.mu.Lock()
		v := s.message
		s.mu.Unlock()
		if v != "" {
			return v
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatal("mensagem SMTP não recebida")
	return ""
}

func TestSMTPMentorCredentialDelivery(t *testing.T) {
	fake := startFakeSMTP(t)
	host, port := fake.address()

	t.Setenv("JED_SMTP_CONFIG", t.TempDir()+"/nao-existe.json")
	t.Setenv("JED_SMTP_HOST", host)
	t.Setenv("JED_SMTP_PORT", strconv.Itoa(port))
	t.Setenv("JED_SMTP_FROM", "jed@example.com")
	t.Setenv("JED_SMTP_FROM_NAME", "JED Simulador")
	t.Setenv("JED_SMTP_SECURITY", "plain")
	t.Setenv("JED_SMTP_USER", "")
	t.Setenv("JED_SMTP_PASSWORD", "")

	cfg, configured, err := loadServerEmailConfig()
	if err != nil {
		t.Fatal(err)
	}
	if !configured {
		t.Fatal("SMTP deveria estar configurado")
	}

	inv := serverMentorInvitation{
		Code:        "MTR-ABCD-EF12",
		Name:        "Mentora Teste",
		Email:       "mentor@example.com",
		Institution: "Escola Teste",
	}
	subject, body := mentorCredentialEmail(inv)
	if err := sendSMTPMessage(cfg, inv.Email, subject, body); err != nil {
		t.Fatal(err)
	}
	msg := fake.waitMessage(t)
	if !strings.Contains(msg, "MTR-ABCD-EF12") {
		t.Fatalf("credencial não encontrada no e-mail:\n%s", msg)
	}
	if !strings.Contains(msg, "mentor@example.com") {
		t.Fatalf("destinatário não encontrado no e-mail:\n%s", msg)
	}
}

func TestMentorEmailFailureDoesNotRemoveCredential(t *testing.T) {
	t.Setenv("JED_SMTP_CONFIG", t.TempDir()+"/nao-existe.json")
	t.Setenv("JED_SMTP_HOST", "127.0.0.1")
	t.Setenv("JED_SMTP_PORT", "1")
	t.Setenv("JED_SMTP_FROM", "jed@example.com")
	t.Setenv("JED_SMTP_SECURITY", "plain")
	t.Setenv("JED_SMTP_USER", "")
	t.Setenv("JED_SMTP_PASSWORD", "")

	st := newServerState(t.TempDir())
	admin, err := st.createInitialAdmin("Admin", "admin@example.com", "SenhaAdmin123")
	if err != nil {
		t.Fatal(err)
	}
	inv, err := st.createMentorInvitation(admin, "Mentor", "mentor@example.com", "Escola", "")
	if err != nil {
		t.Fatal(err)
	}
	delivered := st.deliverMentorInvitationEmail(inv)
	if delivered.EmailStatus != "failed" {
		t.Fatalf("status esperado failed; recebido %q", delivered.EmailStatus)
	}
	if _, ok := st.MentorInvitations[inv.Code]; !ok {
		t.Fatal("credencial foi perdida após falha de SMTP")
	}
}

func TestAccountCreatedEmailUsesTemporaryPassword(t *testing.T) {
	fake := startFakeSMTP(t)
	host, port := fake.address()
	t.Setenv("JED_SMTP_CONFIG", t.TempDir()+"/nao-existe.json")
	t.Setenv("JED_SMTP_HOST", host)
	t.Setenv("JED_SMTP_PORT", strconv.Itoa(port))
	t.Setenv("JED_SMTP_FROM", "jed@example.com")
	t.Setenv("JED_SMTP_FROM_NAME", "JED Simulador")
	t.Setenv("JED_SMTP_SECURITY", "plain")
	t.Setenv("JED_SMTP_USER", "")
	t.Setenv("JED_SMTP_PASSWORD", "")

	st := newServerState(t.TempDir())
	admin, err := st.createInitialAdmin("Admin", "admin@example.com", "SenhaAdmin123")
	if err != nil {
		t.Fatal(err)
	}
	student, err := st.createProvisionedUser(admin, "Aluno Teste", "aluno@example.com", "aluno", "Escola", "A-01")
	if err != nil {
		t.Fatal(err)
	}
	if student.EmailStatus != "sent" {
		t.Fatalf("status de e-mail esperado sent; recebido %q", student.EmailStatus)
	}
	msg := fake.waitMessage(t)
	if !strings.Contains(msg, "acbd1234") {
		t.Fatalf("senha temporária não encontrada no e-mail:\n%s", msg)
	}
	if !strings.Contains(msg, "primeiro acesso") && !strings.Contains(msg, "primeiro=20acesso") {
		t.Fatalf("instrução de primeiro acesso ausente:\n%s", msg)
	}
}
