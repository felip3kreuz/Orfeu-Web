package main

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"mime"
	"mime/quotedprintable"
	"net"
	"net/mail"
	"net/smtp"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

type serverEmailConfig struct {
	Host      string `json:"host"`
	Port      int    `json:"port"`
	Username  string `json:"username,omitempty"`
	Password  string `json:"password,omitempty"`
	FromEmail string `json:"from_email"`
	FromName  string `json:"from_name,omitempty"`
	Security  string `json:"security,omitempty"` // starttls | tls | plain
}

func serverEmailConfigPath() string {
	if p := strings.TrimSpace(os.Getenv("JED_SMTP_CONFIG")); p != "" {
		return p
	}
	if root := strings.TrimSpace(os.Getenv("JED_SERVER_DATA")); root != "" {
		return filepath.Join(root, "servidor_email.json")
	}
	exe, err := os.Executable()
	if err == nil {
		return filepath.Join(filepath.Dir(exe), "servidor_email.json")
	}
	return "servidor_email.json"
}

func loadServerEmailConfig() (serverEmailConfig, bool, error) {
	var cfg serverEmailConfig
	path := serverEmailConfigPath()
	if b, err := os.ReadFile(path); err == nil {
		if err := json.Unmarshal(b, &cfg); err != nil {
			return serverEmailConfig{}, false, fmt.Errorf("configuração SMTP inválida em %s: %w", path, err)
		}
	} else if !os.IsNotExist(err) {
		return serverEmailConfig{}, false, fmt.Errorf("não foi possível ler a configuração SMTP: %w", err)
	}

	// Variáveis de ambiente sobrescrevem o arquivo local.
	if v := strings.TrimSpace(os.Getenv("JED_SMTP_HOST")); v != "" {
		cfg.Host = v
	}
	if v := strings.TrimSpace(os.Getenv("JED_SMTP_PORT")); v != "" {
		n, err := strconv.Atoi(v)
		if err != nil {
			return serverEmailConfig{}, false, errors.New("JED_SMTP_PORT deve ser um número")
		}
		cfg.Port = n
	}
	if v := os.Getenv("JED_SMTP_USER"); v != "" {
		cfg.Username = strings.TrimSpace(v)
	}
	if v := os.Getenv("JED_SMTP_PASSWORD"); v != "" {
		cfg.Password = v
	}
	if v := strings.TrimSpace(os.Getenv("JED_SMTP_FROM")); v != "" {
		cfg.FromEmail = v
	}
	if v := strings.TrimSpace(os.Getenv("JED_SMTP_FROM_NAME")); v != "" {
		cfg.FromName = v
	}
	if v := strings.TrimSpace(os.Getenv("JED_SMTP_SECURITY")); v != "" {
		cfg.Security = v
	}

	cfg.Host = strings.TrimSpace(cfg.Host)
	cfg.Username = strings.TrimSpace(cfg.Username)
	cfg.FromEmail = strings.TrimSpace(cfg.FromEmail)
	cfg.FromName = strings.TrimSpace(cfg.FromName)
	cfg.Security = strings.ToLower(strings.TrimSpace(cfg.Security))

	if cfg.Host == "" {
		return serverEmailConfig{}, false, nil
	}
	if cfg.Port == 0 {
		cfg.Port = 587
	}
	if cfg.Security == "" {
		cfg.Security = "starttls"
	}
	if cfg.FromName == "" {
		cfg.FromName = "JED Simulador"
	}
	if cfg.FromEmail == "" && strings.Contains(cfg.Username, "@") {
		cfg.FromEmail = cfg.Username
	}

	if cfg.Port < 1 || cfg.Port > 65535 {
		return serverEmailConfig{}, false, errors.New("porta SMTP inválida")
	}
	if cfg.Security != "starttls" && cfg.Security != "tls" && cfg.Security != "plain" {
		return serverEmailConfig{}, false, errors.New("segurança SMTP deve ser starttls, tls ou plain")
	}
	if _, err := mail.ParseAddress(cfg.FromEmail); err != nil {
		return serverEmailConfig{}, false, errors.New("endereço de remetente SMTP inválido")
	}
	if (cfg.Username == "") != (cfg.Password == "") {
		return serverEmailConfig{}, false, errors.New("usuário e senha SMTP devem ser informados juntos")
	}
	if cfg.Security == "plain" && cfg.Username != "" {
		return serverEmailConfig{}, false, errors.New("autenticação SMTP não é permitida em modo plain; use starttls ou tls")
	}

	return cfg, true, nil
}

func saveServerEmailConfig(cfg serverEmailConfig) error {
	if cfg.Port == 0 {
		cfg.Port = 587
	}
	if cfg.FromName == "" {
		cfg.FromName = "JED Simulador"
	}
	if cfg.Security == "" {
		cfg.Security = "starttls"
	}
	b, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	b = append(b, '\n')
	path := serverEmailConfigPath()
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, b, 0600); err != nil {
		return err
	}
	if err := os.Rename(tmp, path); err != nil {
		_ = os.Remove(tmp)
		return err
	}
	return nil
}

func encodeSMTPMessage(cfg serverEmailConfig, to, subject, body string) ([]byte, error) {
	toAddr, err := mail.ParseAddress(strings.TrimSpace(to))
	if err != nil {
		return nil, errors.New("e-mail do destinatário inválido")
	}
	fromAddr, err := mail.ParseAddress(cfg.FromEmail)
	if err != nil {
		return nil, errors.New("e-mail do remetente inválido")
	}
	fromAddr.Name = cfg.FromName

	cleanSubject := strings.ReplaceAll(strings.ReplaceAll(subject, "\r", " "), "\n", " ")
	var msg bytes.Buffer
	fmt.Fprintf(&msg, "From: %s\r\n", fromAddr.String())
	fmt.Fprintf(&msg, "To: %s\r\n", toAddr.String())
	fmt.Fprintf(&msg, "Subject: %s\r\n", mime.QEncoding.Encode("UTF-8", cleanSubject))
	fmt.Fprint(&msg, "MIME-Version: 1.0\r\n")
	fmt.Fprint(&msg, "Content-Type: text/plain; charset=UTF-8\r\n")
	fmt.Fprint(&msg, "Content-Transfer-Encoding: quoted-printable\r\n")
	fmt.Fprint(&msg, "\r\n")

	qp := quotedprintable.NewWriter(&msg)
	if _, err := qp.Write([]byte(body)); err != nil {
		return nil, err
	}
	if err := qp.Close(); err != nil {
		return nil, err
	}
	return msg.Bytes(), nil
}

func sendSMTPMessage(cfg serverEmailConfig, to, subject, body string) error {
	msg, err := encodeSMTPMessage(cfg, to, subject, body)
	if err != nil {
		return err
	}
	toAddr, _ := mail.ParseAddress(strings.TrimSpace(to))
	fromAddr, _ := mail.ParseAddress(cfg.FromEmail)

	addr := net.JoinHostPort(cfg.Host, strconv.Itoa(cfg.Port))
	dialer := &net.Dialer{Timeout: 12 * time.Second}
	tlsCfg := &tls.Config{
		ServerName: cfg.Host,
		MinVersion: tls.VersionTLS12,
	}

	var conn net.Conn
	if cfg.Security == "tls" {
		conn, err = tls.DialWithDialer(dialer, "tcp", addr, tlsCfg)
	} else {
		conn, err = dialer.Dial("tcp", addr)
	}
	if err != nil {
		return fmt.Errorf("não foi possível conectar ao servidor SMTP: %w", err)
	}
	defer conn.Close()

	client, err := smtp.NewClient(conn, cfg.Host)
	if err != nil {
		return fmt.Errorf("falha ao iniciar sessão SMTP: %w", err)
	}
	defer client.Close()

	if cfg.Security == "starttls" {
		ok, _ := client.Extension("STARTTLS")
		if !ok {
			return errors.New("o servidor SMTP não oferece STARTTLS")
		}
		if err := client.StartTLS(tlsCfg); err != nil {
			return fmt.Errorf("falha ao ativar STARTTLS: %w", err)
		}
	}

	if cfg.Username != "" {
		auth := smtp.PlainAuth("", cfg.Username, cfg.Password, cfg.Host)
		if err := client.Auth(auth); err != nil {
			return fmt.Errorf("falha de autenticação SMTP: %w", err)
		}
	}

	if err := client.Mail(fromAddr.Address); err != nil {
		return fmt.Errorf("remetente SMTP rejeitado: %w", err)
	}
	if err := client.Rcpt(toAddr.Address); err != nil {
		return fmt.Errorf("destinatário SMTP rejeitado: %w", err)
	}
	w, err := client.Data()
	if err != nil {
		return fmt.Errorf("servidor SMTP recusou o conteúdo: %w", err)
	}
	if _, err := w.Write(msg); err != nil {
		_ = w.Close()
		return fmt.Errorf("falha ao enviar conteúdo do e-mail: %w", err)
	}
	if err := w.Close(); err != nil {
		return fmt.Errorf("falha ao concluir envio do e-mail: %w", err)
	}
	if err := client.Quit(); err != nil {
		return fmt.Errorf("falha ao encerrar sessão SMTP: %w", err)
	}
	return nil
}

func accountRoleLabel(role string) string {
	switch strings.ToLower(strings.TrimSpace(role)) {
	case "admin":
		return "Administrador"
	case "mentor", "tutor":
		return "Mentor"
	default:
		return "Aluno"
	}
}

func jedPublicURL() string {
	if v := strings.TrimSpace(os.Getenv("JED_PUBLIC_URL")); v != "" {
		return strings.TrimRight(v, "/")
	}
	return "https://jed-simulador-onpq.vercel.app"
}

func accountCreatedEmail(u OnlineUser, temporaryPassword string) (string, string) {
	name := strings.TrimSpace(u.Name)
	if name == "" {
		name = "usuário"
	}
	role := accountRoleLabel(u.Role)
	enrollment := ""
	if u.Role == "aluno" {
		if u.EnrollmentID != "" {
			className := "Turma"
			classNumber := 0
			for _, item := range u.Enrollments {
				if item.ClassID == u.CurrentClassID && item.EndedAt == "" {
					className = item.ClassName
					classNumber = item.ClassNumber
					break
				}
			}
			if classNumber > 0 {
				enrollment = fmt.Sprintf("Turma: %s (T%d)\nID da matrícula: %s\n", className, classNumber, u.EnrollmentID)
			} else {
				enrollment = fmt.Sprintf("Turma: %s\nID da matrícula: %s\n", className, u.EnrollmentID)
			}
		} else {
			enrollment = "Turma: aguardando vinculação pelo Administrador\n"
		}
	}
	subject := "Seu cadastro no JED Simulador"
	body := fmt.Sprintf(`Olá, %s.

Seu cadastro no JED Simulador foi realizado por um Administrador.

Perfil: %s
E-mail de acesso: %s
Senha temporária: %s
%s
Acesse: %s

No primeiro acesso, a plataforma exigirá a substituição da senha temporária por uma senha pessoal com pelo menos 8 caracteres.

Não compartilhe sua senha. Se você não esperava este cadastro, entre em contato com a administração da sua instituição.

JED Simulador
`, name, role, u.Email, temporaryPassword, enrollment, jedPublicURL())
	return subject, body
}

func (st *serverState) updateUserEmailDelivery(userID, status, sentAt, emailErr string) OnlineUser {
	st.mu.Lock()
	defer st.mu.Unlock()
	rec, ok := st.Users[userID]
	if !ok {
		return OnlineUser{}
	}
	rec.EmailStatus = status
	rec.EmailSentAt = sentAt
	rec.EmailError = emailErr
	st.Users[userID] = rec
	if err := st.saveLocked(); err != nil && rec.EmailError == "" {
		rec.EmailError = "o envio foi concluído, mas não foi possível salvar o status do e-mail"
		st.Users[userID] = rec
	}
	return rec.OnlineUser
}

func (st *serverState) deliverAccountCreatedEmail(u OnlineUser, temporaryPassword string) OnlineUser {
	cfg, configured, err := loadServerEmailConfig()
	if err != nil {
		return st.updateUserEmailDelivery(u.ID, "failed", "", err.Error())
	}
	if !configured {
		return st.updateUserEmailDelivery(u.ID, "not_configured", "", "")
	}
	subject, body := accountCreatedEmail(u, temporaryPassword)
	if err := sendSMTPMessage(cfg, u.Email, subject, body); err != nil {
		return st.updateUserEmailDelivery(u.ID, "failed", "", err.Error())
	}
	return st.updateUserEmailDelivery(u.ID, "sent", time.Now().UTC().Format(time.RFC3339), "")
}

func emailConfigurationStatus() map[string]any {
	cfg, configured, err := loadServerEmailConfig()
	out := map[string]any{"configured": configured}
	if err != nil {
		out["configured"] = false
		out["error"] = err.Error()
		return out
	}
	if configured {
		out["from_email"] = cfg.FromEmail
		out["from_name"] = cfg.FromName
		out["host"] = cfg.Host
	}
	return out
}

func mentorCredentialEmail(inv serverMentorInvitation) (string, string) {
	subject := "Convite para acessar o JED Simulador como Mentor"
	name := strings.TrimSpace(inv.Name)
	if name == "" {
		name = "Mentor"
	}
	body := fmt.Sprintf(`Olá, %s.

Você recebeu uma credencial para criar uma conta de Mentor no JED Simulador.

Código de credenciamento: %s
Validade: 30 dias
Uso: individual e único

Para ativar:
1. Abra o JED Simulador.
2. Acesse JED Online.
3. Escolha CADASTRAR MENTOR.
4. Informe o mesmo endereço de e-mail que recebeu esta mensagem.
5. Informe o código de credenciamento acima e crie sua senha.

Instituição: %s

Se você não esperava este convite, não utilize o código.

JED Simulador
`, name, inv.Code, displayInstitution(inv.Institution))
	return subject, body
}

func displayInstitution(v string) string {
	v = strings.TrimSpace(v)
	if v == "" {
		return "não informada"
	}
	return v
}

func (st *serverState) updateMentorEmailDelivery(code, status, sentAt, emailErr string) serverMentorInvitation {
	st.mu.Lock()
	defer st.mu.Unlock()

	inv, ok := st.MentorInvitations[code]
	if !ok {
		return serverMentorInvitation{}
	}
	inv.EmailStatus = status
	inv.EmailSentAt = sentAt
	inv.EmailError = emailErr
	st.MentorInvitations[code] = inv
	if err := st.saveLocked(); err != nil && inv.EmailError == "" {
		inv.EmailError = "o envio foi concluído, mas não foi possível salvar o status do e-mail"
	}
	return inv
}

func (st *serverState) deliverMentorInvitationEmail(inv serverMentorInvitation) serverMentorInvitation {
	cfg, configured, err := loadServerEmailConfig()
	if err != nil {
		return st.updateMentorEmailDelivery(inv.Code, "failed", "", err.Error())
	}
	if !configured {
		return st.updateMentorEmailDelivery(inv.Code, "not_configured", "", "")
	}

	subject, body := mentorCredentialEmail(inv)
	if err := sendSMTPMessage(cfg, inv.Email, subject, body); err != nil {
		return st.updateMentorEmailDelivery(inv.Code, "failed", "", err.Error())
	}
	return st.updateMentorEmailDelivery(
		inv.Code,
		"sent",
		time.Now().UTC().Format(time.RFC3339),
		"",
	)
}

func configureEmailInteractive() error {
	fmt.Println("============================================================")
	fmt.Println(" JED SERVIDOR — CONFIGURAÇÃO DE E-MAIL")
	fmt.Println("============================================================")
	fmt.Println()
	fmt.Println("A senha SMTP será salva somente no arquivo local")
	fmt.Println("servidor_email.json. Esse arquivo não deve ser enviado ao GitHub.")
	fmt.Println()

	old, configured, _ := loadServerEmailConfig()

	hostDefault := old.Host
	host, err := readLocalLine(promptWithDefault("Servidor SMTP", hostDefault))
	if err != nil {
		return err
	}
	if host == "" {
		host = hostDefault
	}
	if host == "" {
		return errors.New("servidor SMTP é obrigatório")
	}

	portDefault := old.Port
	if portDefault == 0 {
		portDefault = 587
	}
	portText, err := readLocalLine(promptWithDefault("Porta", strconv.Itoa(portDefault)))
	if err != nil {
		return err
	}
	port := portDefault
	if portText != "" {
		port, err = strconv.Atoi(portText)
		if err != nil || port < 1 || port > 65535 {
			return errors.New("porta inválida")
		}
	}

	securityDefault := old.Security
	if securityDefault == "" {
		securityDefault = "starttls"
	}
	security, err := readLocalLine(promptWithDefault("Segurança (starttls/tls/plain)", securityDefault))
	if err != nil {
		return err
	}
	if security == "" {
		security = securityDefault
	}
	security = strings.ToLower(strings.TrimSpace(security))
	if security != "starttls" && security != "tls" && security != "plain" {
		return errors.New("segurança deve ser starttls, tls ou plain")
	}

	user, err := readLocalLine(promptWithDefault("Usuário SMTP (opcional)", old.Username))
	if err != nil {
		return err
	}
	if user == "" {
		user = old.Username
	}

	password := old.Password
	if user != "" {
		label := "Senha SMTP"
		if configured && old.Password != "" {
			label = "Senha SMTP (Enter mantém a atual)"
		}
		p, err := readSecretConsole(label + ": ")
		if err != nil {
			return err
		}
		if p != "" {
			password = p
		}
	} else {
		password = ""
	}

	fromDefault := old.FromEmail
	if fromDefault == "" && strings.Contains(user, "@") {
		fromDefault = user
	}
	fromEmail, err := readLocalLine(promptWithDefault("E-mail remetente", fromDefault))
	if err != nil {
		return err
	}
	if fromEmail == "" {
		fromEmail = fromDefault
	}

	nameDefault := old.FromName
	if nameDefault == "" {
		nameDefault = "JED Simulador"
	}
	fromName, err := readLocalLine(promptWithDefault("Nome do remetente", nameDefault))
	if err != nil {
		return err
	}
	if fromName == "" {
		fromName = nameDefault
	}

	cfg := serverEmailConfig{
		Host: host, Port: port, Username: user, Password: password,
		FromEmail: fromEmail, FromName: fromName, Security: security,
	}
	// Valida usando a mesma lógica da carga sem depender do arquivo.
	if cfg.FromEmail == "" {
		return errors.New("e-mail remetente é obrigatório")
	}
	if _, err := mail.ParseAddress(cfg.FromEmail); err != nil {
		return errors.New("e-mail remetente inválido")
	}
	if (cfg.Username == "") != (cfg.Password == "") {
		return errors.New("usuário e senha SMTP devem ser informados juntos")
	}
	if cfg.Security == "plain" && cfg.Username != "" {
		return errors.New("não use autenticação em modo plain; escolha starttls ou tls")
	}

	if err := saveServerEmailConfig(cfg); err != nil {
		return err
	}
	fmt.Println()
	fmt.Println("Configuração de e-mail salva em:")
	fmt.Println(serverEmailConfigPath())
	fmt.Println()
	fmt.Println("Use JED_Servidor.exe --test-email para validar o envio antes de cadastrar usuários.")
	return nil
}

func testEmailInteractive() error {
	cfg, configured, err := loadServerEmailConfig()
	if err != nil {
		return err
	}
	if !configured {
		return errors.New("e-mail automático ainda não está configurado; execute --configure-email")
	}
	to, err := readLocalLine("Enviar e-mail de teste para: ")
	if err != nil {
		return err
	}
	if strings.TrimSpace(to) == "" {
		return errors.New("destinatário é obrigatório")
	}
	body := `Este é um e-mail de teste do JED Servidor.

Se você recebeu esta mensagem, o envio automático de credenciais de Mentor está configurado corretamente.

JED Simulador
`
	if err := sendSMTPMessage(cfg, to, "Teste de e-mail — JED Simulador", body); err != nil {
		return err
	}
	fmt.Println("E-mail de teste enviado com sucesso.")
	return nil
}

func promptWithDefault(label, value string) string {
	if strings.TrimSpace(value) == "" {
		return label + ": "
	}
	return fmt.Sprintf("%s [%s]: ", label, value)
}
