package main

import (
	"bytes"
	"encoding/csv"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type OnlineConfig struct {
	ServerURL          string `json:"server_url"`
	Token              string `json:"token"`
	UserID             string `json:"user_id"`
	Email              string `json:"email"`
	Name               string `json:"name"`
	Role               string `json:"role"`
	Status             string `json:"status,omitempty"`
	IsPrimaryAdmin     bool   `json:"is_primary_admin,omitempty"`
	CanInviteMentors   bool   `json:"can_invite_mentors,omitempty"`
	MustChangePassword bool   `json:"must_change_password,omitempty"`
	ClassID            string `json:"class_id,omitempty"`
}

type OnlineLoginResponse struct {
	Token string     `json:"token"`
	User  OnlineUser `json:"user"`
}

type OnlineEnrollment struct {
	ClassID       string `json:"class_id"`
	ClassNumber   int    `json:"class_number"`
	ClassName     string `json:"class_name"`
	StudentNumber int    `json:"student_number"`
	EnrollmentID  string `json:"enrollment_id"`
	StartedAt     string `json:"started_at"`
	EndedAt       string `json:"ended_at,omitempty"`
}

type OnlineUser struct {
	ID                 string             `json:"id"`
	Name               string             `json:"name"`
	Email              string             `json:"email"`
	Role               string             `json:"role"` // aluno | tutor (legado) | mentor | admin
	Status             string             `json:"status,omitempty"`
	Institution        string             `json:"institution,omitempty"`
	InstitutionalID    string             `json:"institutional_id,omitempty"`
	IsPrimaryAdmin     bool               `json:"is_primary_admin,omitempty"`
	CanInviteMentors   bool               `json:"can_invite_mentors,omitempty"`
	MustChangePassword bool               `json:"must_change_password,omitempty"`
	EmailStatus        string             `json:"email_status,omitempty"`
	EmailSentAt        string             `json:"email_sent_at,omitempty"`
	EmailError         string             `json:"email_error,omitempty"`
	CreatedAt          string             `json:"created_at,omitempty"`
	CurrentClassID     string             `json:"current_class_id,omitempty"`
	EnrollmentID       string             `json:"enrollment_id,omitempty"`
	EnrollmentStatus   string             `json:"enrollment_status,omitempty"` // active | waiting
	Enrollments        []OnlineEnrollment `json:"enrollments,omitempty"`
}

type OnlineClass struct {
	ID                string   `json:"id"`
	Number            int      `json:"number,omitempty"`
	Name              string   `json:"name"`
	TutorID           string   `json:"tutor_id"`
	JoinCode          string   `json:"join_code,omitempty"` // legado; novos vínculos são administrativos
	StudentIDs        []string `json:"student_ids"`
	NextStudentNumber int      `json:"next_student_number,omitempty"`
	Scenario          Cenario  `json:"scenario"`
	CreatedAt         string   `json:"created_at"`
}

type OnlineScenario struct {
	ID        string  `json:"id"`
	TutorID   string  `json:"tutor_id,omitempty"`
	Builtin   bool    `json:"builtin,omitempty"`
	Scenario  Cenario `json:"scenario"`
	CreatedAt string  `json:"created_at,omitempty"`
}

type RemoteCompany struct {
	ID              string  `json:"id"`
	OwnerID         string  `json:"owner_id"`
	ClassID         string  `json:"class_id,omitempty"`
	Revision        int     `json:"revision"`
	UpdatedAt       string  `json:"updated_at"`
	ApprovalStatus  string  `json:"approval_status,omitempty"` // aprovado | reprovado
	MentorComment   string  `json:"mentor_comment,omitempty"`
	EvaluatedAt     string  `json:"evaluated_at,omitempty"`
	EvaluatedBy     string  `json:"evaluated_by,omitempty"`
	EvaluatedByName string  `json:"evaluated_by_name,omitempty"`
	Company         Empresa `json:"company"`
}

type APIError struct {
	Error string `json:"error"`
}

type OnlineInvitation struct {
	Code            string `json:"code"`
	Name            string `json:"name"`
	Email           string `json:"email"`
	InstitutionalID string `json:"institutional_id,omitempty"`
	ClassID         string `json:"class_id"`
	TutorID         string `json:"tutor_id"`
	CreatedAt       string `json:"created_at"`
	ExpiresAt       string `json:"expires_at,omitempty"`
	RedeemedAt      string `json:"redeemed_at,omitempty"`
	RevokedAt       string `json:"revoked_at,omitempty"`
	UserID          string `json:"user_id,omitempty"`
}

type OnlineMentorInvitation struct {
	Code            string `json:"code"`
	Name            string `json:"name"`
	Email           string `json:"email"`
	Institution     string `json:"institution,omitempty"`
	InstitutionalID string `json:"institutional_id,omitempty"`
	IssuerID        string `json:"issuer_id"`
	CreatedAt       string `json:"created_at"`
	ExpiresAt       string `json:"expires_at,omitempty"`
	RedeemedAt      string `json:"redeemed_at,omitempty"`
	RevokedAt       string `json:"revoked_at,omitempty"`
	UserID          string `json:"user_id,omitempty"`
	EmailStatus     string `json:"email_status,omitempty"`
	EmailSentAt     string `json:"email_sent_at,omitempty"`
	EmailError      string `json:"email_error,omitempty"`
}

var onlineConfig OnlineConfig

func onlineConfigPath() string {
	return filepath.Join(dataDir(), "conexao_online.json")
}

func loadOnlineConfig() {
	var c OnlineConfig
	b, err := os.ReadFile(onlineConfigPath())
	if err == nil && json.Unmarshal(b, &c) == nil {
		onlineConfig = c
	}
}

func saveOnlineConfig() {
	_ = saveJSON(onlineConfigPath(), onlineConfig)
}

func normalizeServerURL(u string) string {
	u = strings.TrimSpace(u)
	u = strings.TrimRight(u, "/")
	if u == "" {
		return ""
	}
	if !strings.HasPrefix(u, "http://") && !strings.HasPrefix(u, "https://") {
		u = "https://" + u
	}
	return u
}

func apiRequest(method, path string, body any, out any) error {
	if onlineConfig.ServerURL == "" {
		return errors.New("servidor online não configurado")
	}
	var rd io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return err
		}
		rd = bytes.NewReader(b)
	}
	req, err := http.NewRequest(method, onlineConfig.ServerURL+path, rd)
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	if onlineConfig.Token != "" {
		req.Header.Set("Authorization", "Bearer "+onlineConfig.Token)
	}
	client := &http.Client{Timeout: 12 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("não foi possível conectar ao servidor: %w", err)
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(io.LimitReader(resp.Body, 8<<20))
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		var ae APIError
		if json.Unmarshal(raw, &ae) == nil && ae.Error != "" {
			return errors.New(ae.Error)
		}
		return fmt.Errorf("servidor respondeu HTTP %d", resp.StatusCode)
	}
	if out != nil && len(raw) > 0 {
		if err := json.Unmarshal(raw, out); err != nil {
			return err
		}
	}
	return nil
}

func onlineLogin(server, email, password string) error {
	onlineConfig.ServerURL = normalizeServerURL(server)
	var r OnlineLoginResponse
	if err := apiRequest("POST", "/api/v1/login", map[string]string{
		"email":    strings.TrimSpace(strings.ToLower(email)),
		"password": password,
	}, &r); err != nil {
		return err
	}
	applyOnlineLogin(server, r)
	return nil
}

func applyOnlineLogin(server string, r OnlineLoginResponse) {
	onlineConfig.ServerURL = normalizeServerURL(server)
	onlineConfig.Token = r.Token
	onlineConfig.UserID = r.User.ID
	onlineConfig.Email = r.User.Email
	onlineConfig.Name = r.User.Name
	onlineConfig.Role = r.User.Role
	onlineConfig.Status = r.User.Status
	onlineConfig.IsPrimaryAdmin = r.User.IsPrimaryAdmin
	onlineConfig.CanInviteMentors = r.User.CanInviteMentors
	onlineConfig.MustChangePassword = r.User.MustChangePassword
	saveOnlineConfig()
}

func onlineRegisterStudent(server, name, email, password, institutionalID string) error {
	onlineConfig.ServerURL = normalizeServerURL(server)
	var r OnlineLoginResponse
	if err := apiRequest("POST", "/api/v1/register/student", map[string]string{
		"name":             strings.TrimSpace(name),
		"email":            strings.TrimSpace(strings.ToLower(email)),
		"password":         password,
		"institutional_id": strings.TrimSpace(institutionalID),
	}, &r); err != nil {
		return err
	}
	applyOnlineLogin(server, r)
	return nil
}

func onlineRegisterMentor(server, name, email, password, institution, institutionalID, credentialCode string) error {
	onlineConfig.ServerURL = normalizeServerURL(server)
	var r OnlineLoginResponse
	if err := apiRequest("POST", "/api/v1/register/mentor", map[string]string{
		"name":             strings.TrimSpace(name),
		"email":            strings.TrimSpace(strings.ToLower(email)),
		"password":         password,
		"institution":      strings.TrimSpace(institution),
		"institutional_id": strings.TrimSpace(institutionalID),
		"credential_code":  strings.ToUpper(strings.TrimSpace(credentialCode)),
	}, &r); err != nil {
		return err
	}
	applyOnlineLogin(server, r)
	return nil
}

func onlineLogout() {
	if onlineConfig.Token != "" {
		_ = apiRequest("POST", "/api/v1/logout", nil, nil)
	}
	server := onlineConfig.ServerURL
	onlineConfig = OnlineConfig{ServerURL: server}
	saveOnlineConfig()
}

func onlineValidateSession() error {
	var u OnlineUser
	if err := apiRequest("GET", "/api/v1/me", nil, &u); err != nil {
		return err
	}
	onlineConfig.UserID, onlineConfig.Name, onlineConfig.Email, onlineConfig.Role = u.ID, u.Name, u.Email, u.Role
	onlineConfig.Status = u.Status
	onlineConfig.IsPrimaryAdmin = u.IsPrimaryAdmin
	onlineConfig.CanInviteMentors = u.CanInviteMentors
	onlineConfig.MustChangePassword = u.MustChangePassword
	saveOnlineConfig()
	return nil
}

func onlineChangePassword(current, next string) error {
	err := apiRequest("POST", "/api/v1/password", map[string]string{"current": current, "new": next}, nil)
	if err == nil {
		onlineConfig.MustChangePassword = false
		saveOnlineConfig()
	}
	return err
}

func onlineRevokeInvitation(code string) error {
	return apiRequest("POST", "/api/v1/invitations/revoke", map[string]string{"code": strings.TrimSpace(strings.ToUpper(code))}, nil)
}

func onlineClasses() ([]OnlineClass, error) {
	var out []OnlineClass
	err := apiRequest("GET", "/api/v1/classes", nil, &out)
	return out, err
}

func onlineCreateClass(name string, scenario Cenario) (OnlineClass, error) {
	var out OnlineClass
	err := apiRequest("POST", "/api/v1/classes", map[string]any{"name": name, "scenario": scenario}, &out)
	return out, err
}

func onlineJoinClass(code string) (OnlineClass, error) {
	var out OnlineClass
	err := apiRequest("POST", "/api/v1/classes/join", map[string]string{"code": strings.TrimSpace(strings.ToUpper(code))}, &out)
	if err == nil {
		onlineConfig.ClassID = out.ID
		saveOnlineConfig()
	}
	return out, err
}

func onlineRedeemInvitation(server, code, password string) error {
	onlineConfig.ServerURL = normalizeServerURL(server)
	var r OnlineLoginResponse
	if err := apiRequest("POST", "/api/v1/invitations/redeem", map[string]string{
		"code": strings.ToUpper(strings.TrimSpace(code)), "password": password,
	}, &r); err != nil {
		return err
	}
	applyOnlineLogin(server, r)
	return nil
}

func onlineCreateInvitation(classID, name, email, institutionalID string) (OnlineInvitation, error) {
	var out OnlineInvitation
	err := apiRequest("POST", "/api/v1/invitations", map[string]string{
		"class_id": classID, "name": name, "email": email, "institutional_id": institutionalID,
	}, &out)
	return out, err
}

func onlineCreateMentorInvitation(name, email, institution, institutionalID string) (OnlineMentorInvitation, error) {
	var out OnlineMentorInvitation
	err := apiRequest("POST", "/api/v1/mentor-invitations", map[string]string{
		"name": name, "email": email, "institution": institution, "institutional_id": institutionalID,
	}, &out)
	return out, err
}

func onlineMentorInvitations() ([]OnlineMentorInvitation, error) {
	var out []OnlineMentorInvitation
	err := apiRequest("GET", "/api/v1/mentor-invitations", nil, &out)
	return out, err
}

func onlineRevokeMentorInvitation(code string) error {
	return apiRequest("POST", "/api/v1/mentor-invitations/revoke",
		map[string]string{"code": strings.TrimSpace(strings.ToUpper(code))}, nil)
}

func onlineInvitations(classID string) ([]OnlineInvitation, error) {
	var out []OnlineInvitation
	path := "/api/v1/invitations"
	if strings.TrimSpace(classID) != "" {
		path += "?class_id=" + strings.TrimSpace(classID)
	}
	err := apiRequest("GET", path, nil, &out)
	return out, err
}

func importStudentsCSV(path string) ([][3]string, error) {
	f, err := os.Open(strings.TrimSpace(strings.Trim(path, `"`)))
	if err != nil {
		return nil, err
	}
	defer f.Close()
	r := csv.NewReader(f)
	r.FieldsPerRecord = -1
	r.TrimLeadingSpace = true
	rows, err := r.ReadAll()
	if err != nil {
		return nil, err
	}
	out := [][3]string{}
	for i, row := range rows {
		if len(row) < 2 {
			continue
		}
		name := strings.TrimSpace(row[0])
		email := strings.TrimSpace(row[1])
		inst := ""
		if len(row) > 2 {
			inst = strings.TrimSpace(row[2])
		}
		if i == 0 && (strings.Contains(strings.ToLower(name), "nome") || strings.Contains(strings.ToLower(email), "email")) {
			continue
		}
		if name != "" && strings.Contains(email, "@") {
			out = append(out, [3]string{name, email, inst})
		}
	}
	return out, nil
}

func onlineAdminUsers() ([]OnlineUser, error) {
	var out []OnlineUser
	err := apiRequest("GET", "/api/v1/admin/users", nil, &out)
	return out, err
}

func onlineAdminCreateAdmin(name, email, password string) (OnlineUser, error) {
	var out OnlineUser
	err := apiRequest("POST", "/api/v1/admin/create-admin", map[string]string{
		"name":     name,
		"email":    strings.ToLower(strings.TrimSpace(email)),
		"password": password,
	}, &out)
	return out, err
}

func onlineAdminSetUserStatus(userID, status string) (OnlineUser, error) {
	var out OnlineUser
	err := apiRequest("POST", "/api/v1/admin/user-status", map[string]string{
		"user_id": userID,
		"status":  status,
	}, &out)
	return out, err
}

func onlineAdminSetMentorPermission(userID string, allowed bool) (OnlineUser, error) {
	var out OnlineUser
	err := apiRequest("POST", "/api/v1/admin/mentor-permission", map[string]any{
		"user_id": userID,
		"allowed": allowed,
	}, &out)
	return out, err
}

func onlineAdminTransferPrimary(userID string) (OnlineUser, error) {
	var out OnlineUser
	err := apiRequest("POST", "/api/v1/admin/transfer-primary", map[string]string{
		"user_id": userID,
	}, &out)
	if err == nil {
		_ = onlineValidateSession()
	}
	return out, err
}

func onlineCreateStudent(name, email, password string) (OnlineUser, error) {
	var out OnlineUser
	err := apiRequest("POST", "/api/v1/users/student", map[string]string{
		"name": name, "email": strings.ToLower(strings.TrimSpace(email)), "password": password,
	}, &out)
	return out, err
}

func onlineSyncCompany(e *Empresa) (RemoteCompany, error) {
	var out RemoteCompany
	if e == nil {
		return out, errors.New("nenhuma empresa aberta")
	}
	if e.Responsavel == "" {
		e.Responsavel = onlineConfig.Name
	}
	if onlineConfig.ClassID != "" && e.TurmaID == "" {
		e.TurmaID = onlineConfig.ClassID
	}
	path := "/api/v1/companies/" + e.LocalID
	err := apiRequest("PUT", path, map[string]any{"class_id": e.TurmaID, "company": e}, &out)
	if err == nil {
		e.SyncState = "synced"
		e.Revision = out.Company.Revision
		e.UpdatedAt = out.Company.UpdatedAt
		_, _ = saveCompany(e)
		e.SyncState = "synced"
	}
	return out, err
}

func onlineMyCompanies() ([]RemoteCompany, error) {
	var out []RemoteCompany
	err := apiRequest("GET", "/api/v1/companies", nil, &out)
	return out, err
}

func onlineClassCompanies(classID string) ([]RemoteCompany, error) {
	var out []RemoteCompany
	err := apiRequest("GET", "/api/v1/classes/"+classID+"/companies", nil, &out)
	return out, err
}
