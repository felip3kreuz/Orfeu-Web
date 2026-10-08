package main

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
)

type serverUserRecord struct {
	OnlineUser
	Salt string `json:"salt"`
	Hash string `json:"hash"`
}

type serverState struct {
	mu                sync.RWMutex
	root              string
	Users             map[string]serverUserRecord       `json:"users"`
	Classes           map[string]OnlineClass            `json:"classes"`
	Companies         map[string]RemoteCompany          `json:"companies"`
	Scenarios         map[string]OnlineScenario         `json:"scenarios"`
	Invitations       map[string]serverInvitation       `json:"invitations"`
	MentorInvitations map[string]serverMentorInvitation `json:"mentor_invitations"`
	NextClassNumber   int                               `json:"next_class_number"`
	Sessions          map[string]serverSession          `json:"-"`
}

type serverSession struct {
	UserID  string
	Expires time.Time
}

type loginAttempt struct {
	When time.Time
}

const defaultProvisionedPassword = "acbd1234"

var loginGuard = struct {
	sync.Mutex
	Attempts map[string][]loginAttempt
}{Attempts: map[string][]loginAttempt{}}

func loginAllowed(key string) bool {
	loginGuard.Lock()
	defer loginGuard.Unlock()
	cut := time.Now().Add(-10 * time.Minute)
	list := loginGuard.Attempts[key]
	kept := list[:0]
	for _, a := range list {
		if a.When.After(cut) {
			kept = append(kept, a)
		}
	}
	loginGuard.Attempts[key] = kept
	return len(kept) < 8
}

func recordLoginFailure(key string) {
	loginGuard.Lock()
	loginGuard.Attempts[key] = append(loginGuard.Attempts[key], loginAttempt{When: time.Now()})
	loginGuard.Unlock()
}

func clearLoginFailures(key string) {
	loginGuard.Lock()
	delete(loginGuard.Attempts, key)
	loginGuard.Unlock()
}

type serverInvitation struct {
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

type serverMentorInvitation struct {
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

func newServerState(root string) *serverState {
	return &serverState{
		root:              root,
		Users:             map[string]serverUserRecord{},
		Classes:           map[string]OnlineClass{},
		Companies:         map[string]RemoteCompany{},
		Scenarios:         map[string]OnlineScenario{},
		Invitations:       map[string]serverInvitation{},
		MentorInvitations: map[string]serverMentorInvitation{},
		NextClassNumber:   1,
		Sessions:          map[string]serverSession{},
	}
}

func serverDataPath(root string) string { return filepath.Join(root, "jed_server_data.json") }

func (st *serverState) load() error {
	_ = os.MkdirAll(st.root, 0755)
	b, err := os.ReadFile(serverDataPath(st.root))
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return err
	}
	var disk struct {
		Users             map[string]serverUserRecord       `json:"users"`
		Classes           map[string]OnlineClass            `json:"classes"`
		Companies         map[string]RemoteCompany          `json:"companies"`
		Scenarios         map[string]OnlineScenario         `json:"scenarios"`
		Invitations       map[string]serverInvitation       `json:"invitations"`
		MentorInvitations map[string]serverMentorInvitation `json:"mentor_invitations"`
		NextClassNumber   int                               `json:"next_class_number"`
	}
	if err := json.Unmarshal(b, &disk); err != nil {
		return err
	}
	if disk.Users != nil {
		st.Users = disk.Users
	}
	if disk.Classes != nil {
		st.Classes = disk.Classes
	}
	if disk.Companies != nil {
		st.Companies = disk.Companies
	}
	if disk.Scenarios != nil {
		st.Scenarios = disk.Scenarios
	}
	if disk.Invitations != nil {
		st.Invitations = disk.Invitations
	}
	if disk.MentorInvitations != nil {
		st.MentorInvitations = disk.MentorInvitations
	}
	if disk.NextClassNumber > 0 {
		st.NextClassNumber = disk.NextClassNumber
	}

	// RC1.6 migration: normalize status and administrative metadata.
	adminIDs := []string{}
	primaryFound := false
	for id, rec := range st.Users {
		if rec.Status == "" {
			rec.Status = "active"
		}
		if rec.Role == "admin" {
			rec.CanInviteMentors = true
			adminIDs = append(adminIDs, id)
			if rec.IsPrimaryAdmin {
				primaryFound = true
			}
		}
		st.Users[id] = rec
	}
	if len(adminIDs) > 0 && !primaryFound {
		// Deterministic fallback for data created before is_primary_admin existed.
		sort.Strings(adminIDs)
		rec := st.Users[adminIDs[0]]
		rec.IsPrimaryAdmin = true
		st.Users[adminIDs[0]] = rec
	}

	// W7.3 migration: every class receives a stable numeric identifier and
	// existing class memberships receive permanent enrollment IDs.
	st.migrateCentralizedClasses()
	return nil
}

func (st *serverState) migrateCentralizedClasses() {
	type classRef struct {
		ID      string
		Created string
	}
	refs := make([]classRef, 0, len(st.Classes))
	usedNumbers := map[int]bool{}
	maxNumber := 0
	for id, cl := range st.Classes {
		refs = append(refs, classRef{ID: id, Created: cl.CreatedAt})
		if cl.Number > 0 {
			usedNumbers[cl.Number] = true
			if cl.Number > maxNumber {
				maxNumber = cl.Number
			}
		}
	}
	sort.Slice(refs, func(i, j int) bool {
		if refs[i].Created != refs[j].Created {
			return refs[i].Created < refs[j].Created
		}
		return refs[i].ID < refs[j].ID
	})
	next := 1
	for _, ref := range refs {
		cl := st.Classes[ref.ID]
		if cl.Number <= 0 {
			for usedNumbers[next] {
				next++
			}
			cl.Number = next
			usedNumbers[next] = true
			if next > maxNumber {
				maxNumber = next
			}
			next++
		}
		st.Classes[ref.ID] = cl
	}
	if st.NextClassNumber <= maxNumber {
		st.NextClassNumber = maxNumber + 1
	}
	if st.NextClassNumber <= 0 {
		st.NextClassNumber = 1
	}

	// Respect any enrollment history already created by W7.3.
	for id, rec := range st.Users {
		if rec.Role != "aluno" {
			continue
		}
		for _, enrollment := range rec.Enrollments {
			if enrollment.StudentNumber <= 0 {
				continue
			}
			if cl, ok := st.Classes[enrollment.ClassID]; ok && enrollment.StudentNumber > cl.NextStudentNumber {
				cl.NextStudentNumber = enrollment.StudentNumber
				st.Classes[enrollment.ClassID] = cl
			}
		}
		if rec.EnrollmentStatus == "" {
			rec.EnrollmentStatus = "waiting"
			st.Users[id] = rec
		}
	}

	now := time.Now().UTC().Format(time.RFC3339)
	for _, ref := range refs {
		cl := st.Classes[ref.ID]
		for _, studentID := range cl.StudentIDs {
			rec, ok := st.Users[studentID]
			if !ok || rec.Role != "aluno" {
				continue
			}
			found := false
			for _, enrollment := range rec.Enrollments {
				if enrollment.ClassID == cl.ID {
					found = true
					break
				}
			}
			if !found {
				cl.NextStudentNumber++
				enrollmentID := fmt.Sprintf("T%dA%d", cl.Number, cl.NextStudentNumber)
				rec.Enrollments = append(rec.Enrollments, OnlineEnrollment{
					ClassID: cl.ID, ClassNumber: cl.Number, ClassName: cl.Name,
					StudentNumber: cl.NextStudentNumber, EnrollmentID: enrollmentID, StartedAt: now,
				})
			}
			// Legacy memberships may contain a student in several classes. The
			// class with the greatest numeric identifier becomes the current one.
			if rec.CurrentClassID == "" {
				rec.CurrentClassID = cl.ID
				for _, enrollment := range rec.Enrollments {
					if enrollment.ClassID == cl.ID {
						rec.EnrollmentID = enrollment.EnrollmentID
					}
				}
			} else if current, ok := st.Classes[rec.CurrentClassID]; !ok || cl.Number > current.Number {
				rec.CurrentClassID = cl.ID
				for _, enrollment := range rec.Enrollments {
					if enrollment.ClassID == cl.ID {
						rec.EnrollmentID = enrollment.EnrollmentID
					}
				}
			}
			rec.EnrollmentStatus = "active"
			st.Users[studentID] = rec
		}
		st.Classes[ref.ID] = cl
	}

	// Close duplicate legacy memberships so exactly one enrollment is current.
	for id, rec := range st.Users {
		if rec.Role != "aluno" || rec.CurrentClassID == "" {
			continue
		}
		for i := range rec.Enrollments {
			if rec.Enrollments[i].ClassID != rec.CurrentClassID && rec.Enrollments[i].EndedAt == "" {
				rec.Enrollments[i].EndedAt = now
			}
		}
		for classID, cl := range st.Classes {
			if classID != rec.CurrentClassID && containsStudentID(cl.StudentIDs, id) {
				cl.StudentIDs = removeStudentID(cl.StudentIDs, id)
				st.Classes[classID] = cl
			}
		}
		st.Users[id] = rec
	}
}

func (st *serverState) saveLocked() error {
	disk := struct {
		Users             map[string]serverUserRecord       `json:"users"`
		Classes           map[string]OnlineClass            `json:"classes"`
		Companies         map[string]RemoteCompany          `json:"companies"`
		Scenarios         map[string]OnlineScenario         `json:"scenarios"`
		Invitations       map[string]serverInvitation       `json:"invitations"`
		MentorInvitations map[string]serverMentorInvitation `json:"mentor_invitations"`
		NextClassNumber   int                               `json:"next_class_number"`
	}{st.Users, st.Classes, st.Companies, st.Scenarios, st.Invitations, st.MentorInvitations, st.NextClassNumber}

	b, err := json.MarshalIndent(disk, "", "  ")
	if err != nil {
		return err
	}

	target := serverDataPath(st.root)
	tmp := target + ".tmp"

	if old, err := os.ReadFile(target); err == nil && len(old) > 0 {
		backupDir := filepath.Join(st.root, "backups")
		_ = os.MkdirAll(backupDir, 0700)
		name := "jed_server_" + time.Now().UTC().Format("20060102_150405") + ".json"
		_ = os.WriteFile(filepath.Join(backupDir, name), old, 0600)

		if entries, err := os.ReadDir(backupDir); err == nil {
			cut := time.Now().Add(-30 * 24 * time.Hour)
			for _, e := range entries {
				if info, err := e.Info(); err == nil && info.ModTime().Before(cut) {
					_ = os.Remove(filepath.Join(backupDir, e.Name()))
				}
			}
		}
	}

	if err := os.WriteFile(tmp, b, 0600); err != nil {
		return err
	}
	return os.Rename(tmp, target)
}

func randomHex(n int) string {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		return fmt.Sprintf("%d", time.Now().UnixNano())
	}
	return hex.EncodeToString(b)
}

func passwordHash(password, salt string) string {
	// PBKDF2-HMAC-SHA256, implemented here to keep the server self-contained.
	// 180k iterations is intentionally expensive enough for this educational server.
	const iterations = 180000
	const keyLen = 32
	saltBytes, _ := hex.DecodeString(salt)
	if len(saltBytes) == 0 {
		saltBytes = []byte(salt)
	}
	hLen := sha256.Size
	numBlocks := (keyLen + hLen - 1) / hLen
	out := make([]byte, 0, numBlocks*hLen)
	for block := 1; block <= numBlocks; block++ {
		mac := hmac.New(sha256.New, []byte(password))
		mac.Write(saltBytes)
		mac.Write([]byte{byte(block >> 24), byte(block >> 16), byte(block >> 8), byte(block)})
		u := mac.Sum(nil)
		t := append([]byte(nil), u...)
		for i := 1; i < iterations; i++ {
			mac = hmac.New(sha256.New, []byte(password))
			mac.Write(u)
			u = mac.Sum(nil)
			for j := range t {
				t[j] ^= u[j]
			}
		}
		out = append(out, t...)
	}
	return hex.EncodeToString(out[:keyLen])
}

func validPassword(p string) bool { return len(p) >= 8 }

func isMentorRole(role string) bool {
	return role == "mentor" || role == "tutor"
}

func isAdminRole(role string) bool {
	return role == "admin"
}

func activeStatus(status string) bool {
	return status == "" || status == "active"
}

func (st *serverState) createUser(name, email, password, role string) (OnlineUser, error) {
	return st.createUserProfile(name, email, password, role, "", "")
}

func (st *serverState) createUserProfile(name, email, password, role, institution, institutionalID string) (OnlineUser, error) {
	email = strings.ToLower(strings.TrimSpace(email))
	name = strings.TrimSpace(name)
	institution = strings.TrimSpace(institution)
	institutionalID = strings.TrimSpace(institutionalID)
	if name == "" || !strings.Contains(email, "@") {
		return OnlineUser{}, errors.New("nome ou e-mail inválido")
	}
	if !validPassword(password) {
		return OnlineUser{}, errors.New("a senha deve ter pelo menos 8 caracteres")
	}
	if role != "aluno" && role != "tutor" && role != "mentor" && role != "admin" {
		return OnlineUser{}, errors.New("papel inválido")
	}
	st.mu.Lock()
	defer st.mu.Unlock()
	for _, u := range st.Users {
		if strings.EqualFold(u.Email, email) {
			return OnlineUser{}, errors.New("e-mail já cadastrado")
		}
	}
	id := "usr-" + randomHex(8)
	salt := randomHex(16)
	u := OnlineUser{
		ID: id, Name: name, Email: email, Role: role, Status: "active",
		Institution: institution, InstitutionalID: institutionalID,
		CanInviteMentors: role == "admin",
		CreatedAt:        time.Now().UTC().Format(time.RFC3339),
	}
	st.Users[id] = serverUserRecord{OnlineUser: u, Salt: salt, Hash: passwordHash(password, salt)}
	return u, st.saveLocked()
}

func (st *serverState) createProvisionedUser(actor OnlineUser, name, email, role, institution, institutionalID string) (OnlineUser, error) {
	return st.createProvisionedUserForClass(actor, name, email, role, institution, institutionalID, "")
}

func (st *serverState) createProvisionedUserForClass(actor OnlineUser, name, email, role, institution, institutionalID, classID string) (OnlineUser, error) {
	if actor.Role != "admin" || !activeStatus(actor.Status) {
		return OnlineUser{}, errors.New("somente administradores podem cadastrar usuários")
	}
	role = strings.ToLower(strings.TrimSpace(role))
	if role == "tutor" {
		role = "mentor"
	}
	if role != "aluno" && role != "mentor" && role != "admin" {
		return OnlineUser{}, errors.New("papel inválido")
	}
	classID = strings.TrimSpace(classID)
	if classID != "" && role != "aluno" {
		return OnlineUser{}, errors.New("turma só pode ser atribuída a contas de Aluno")
	}
	if classID != "" {
		st.mu.RLock()
		_, exists := st.Classes[classID]
		st.mu.RUnlock()
		if !exists {
			return OnlineUser{}, errors.New("turma não encontrada")
		}
	}
	u, err := st.createUserProfile(name, email, defaultProvisionedPassword, role, institution, institutionalID)
	if err != nil {
		return OnlineUser{}, err
	}
	st.mu.Lock()
	rec := st.Users[u.ID]
	rec.MustChangePassword = true
	if role == "mentor" {
		rec.CanInviteMentors = false
	}
	if role == "aluno" {
		rec.EnrollmentStatus = "waiting"
	}
	st.Users[u.ID] = rec
	if role == "aluno" && classID != "" {
		if _, err := st.enrollStudentLocked(u.ID, classID, time.Now().UTC().Format(time.RFC3339)); err != nil {
			delete(st.Users, u.ID)
			st.mu.Unlock()
			return OnlineUser{}, err
		}
	}
	err = st.saveLocked()
	if current, ok := st.Users[u.ID]; ok {
		u = current.OnlineUser
	}
	st.mu.Unlock()
	if err != nil {
		return OnlineUser{}, err
	}
	return st.deliverAccountCreatedEmail(u, defaultProvisionedPassword), nil
}

func removeStudentID(ids []string, target string) []string {
	out := ids[:0]
	for _, id := range ids {
		if id != target {
			out = append(out, id)
		}
	}
	return out
}

func containsStudentID(ids []string, target string) bool {
	for _, id := range ids {
		if id == target {
			return true
		}
	}
	return false
}

// enrollStudentLocked assigns or transfers a student while st.mu is held.
// Enrollment numbers are monotonic per class and are never reused.
func (st *serverState) enrollStudentLocked(userID, classID, when string) (OnlineUser, error) {
	rec, exists := st.Users[userID]
	if !exists {
		return OnlineUser{}, errors.New("aluno não encontrado")
	}
	if rec.Role != "aluno" {
		return OnlineUser{}, errors.New("somente contas de Aluno podem ser vinculadas a turmas")
	}
	classID = strings.TrimSpace(classID)
	if rec.CurrentClassID == classID && classID != "" {
		return rec.OnlineUser, nil
	}

	if rec.CurrentClassID != "" {
		if oldClass, ok := st.Classes[rec.CurrentClassID]; ok {
			oldClass.StudentIDs = removeStudentID(oldClass.StudentIDs, userID)
			st.Classes[oldClass.ID] = oldClass
		}
		for i := range rec.Enrollments {
			if rec.Enrollments[i].ClassID == rec.CurrentClassID && rec.Enrollments[i].EndedAt == "" {
				rec.Enrollments[i].EndedAt = when
			}
		}
	}

	if classID == "" {
		rec.CurrentClassID = ""
		rec.EnrollmentID = ""
		rec.EnrollmentStatus = "waiting"
		st.Users[userID] = rec
		return rec.OnlineUser, nil
	}

	cl, ok := st.Classes[classID]
	if !ok {
		return OnlineUser{}, errors.New("turma não encontrada")
	}
	if cl.Number <= 0 {
		return OnlineUser{}, errors.New("turma sem numeração válida")
	}
	cl.NextStudentNumber++
	studentNumber := cl.NextStudentNumber
	enrollmentID := fmt.Sprintf("T%dA%d", cl.Number, studentNumber)
	if !containsStudentID(cl.StudentIDs, userID) {
		cl.StudentIDs = append(cl.StudentIDs, userID)
	}
	rec.CurrentClassID = cl.ID
	rec.EnrollmentID = enrollmentID
	rec.EnrollmentStatus = "active"
	rec.Enrollments = append(rec.Enrollments, OnlineEnrollment{
		ClassID: cl.ID, ClassNumber: cl.Number, ClassName: cl.Name,
		StudentNumber: studentNumber, EnrollmentID: enrollmentID, StartedAt: when,
	})
	st.Classes[cl.ID] = cl
	st.Users[userID] = rec
	return rec.OnlineUser, nil
}

func (st *serverState) assignStudentClass(actor OnlineUser, userID, classID string) (OnlineUser, error) {
	if actor.Role != "admin" || !activeStatus(actor.Status) {
		return OnlineUser{}, errors.New("somente administradores podem alterar matrículas")
	}
	st.mu.Lock()
	defer st.mu.Unlock()
	u, err := st.enrollStudentLocked(strings.TrimSpace(userID), strings.TrimSpace(classID), time.Now().UTC().Format(time.RFC3339))
	if err != nil {
		return OnlineUser{}, err
	}
	if err := st.saveLocked(); err != nil {
		return OnlineUser{}, err
	}
	return u, nil
}

func (st *serverState) resolveClassRef(ref string) (string, error) {
	ref = strings.TrimSpace(ref)
	if ref == "" {
		return "", nil
	}
	st.mu.RLock()
	defer st.mu.RUnlock()
	if _, ok := st.Classes[ref]; ok {
		return ref, nil
	}
	normalized := strings.TrimPrefix(strings.ToUpper(ref), "T")
	if n, err := strconv.Atoi(normalized); err == nil {
		for _, cl := range st.Classes {
			if cl.Number == n {
				return cl.ID, nil
			}
		}
	}
	for _, cl := range st.Classes {
		if strings.EqualFold(strings.TrimSpace(cl.Name), ref) {
			return cl.ID, nil
		}
	}
	return "", errors.New("turma não encontrada: " + ref)
}

func (st *serverState) createAdminClass(actor OnlineUser, name, mentorID string) (OnlineClass, error) {
	if actor.Role != "admin" || !activeStatus(actor.Status) {
		return OnlineClass{}, errors.New("somente administradores podem criar turmas")
	}
	name = strings.TrimSpace(name)
	mentorID = strings.TrimSpace(mentorID)
	if name == "" {
		return OnlineClass{}, errors.New("informe o nome da turma")
	}
	st.mu.Lock()
	defer st.mu.Unlock()
	mentor, ok := st.Users[mentorID]
	if !ok || !isMentorRole(mentor.Role) || !activeStatus(mentor.Status) {
		return OnlineClass{}, errors.New("Mentor responsável inválido ou inativo")
	}
	for _, existing := range st.Classes {
		if strings.EqualFold(strings.TrimSpace(existing.Name), name) {
			return OnlineClass{}, errors.New("já existe uma turma com esse nome")
		}
	}
	number := st.NextClassNumber
	if number <= 0 {
		number = 1
	}
	st.NextClassNumber = number + 1
	cl := OnlineClass{
		ID: "tur-" + randomHex(8), Number: number, Name: name, TutorID: mentorID,
		StudentIDs: []string{}, NextStudentNumber: 0, Scenario: cenariosBase[0],
		CreatedAt: time.Now().UTC().Format(time.RFC3339),
	}
	st.Classes[cl.ID] = cl
	if err := st.saveLocked(); err != nil {
		return OnlineClass{}, err
	}
	return cl, nil
}

func (st *serverState) authenticate(email, password string) (OnlineUser, bool) {
	email = strings.ToLower(strings.TrimSpace(email))
	st.mu.RLock()
	defer st.mu.RUnlock()
	for _, u := range st.Users {
		if strings.EqualFold(u.Email, email) {
			if !activeStatus(u.Status) {
				return OnlineUser{}, false
			}
			got := passwordHash(password, u.Salt)
			return u.OnlineUser, hmac.Equal([]byte(got), []byte(u.Hash))
		}
	}
	return OnlineUser{}, false
}

func (st *serverState) newSession(userID string) string {
	tok := randomHex(32)
	st.mu.Lock()
	st.Sessions[tok] = serverSession{UserID: userID, Expires: time.Now().Add(12 * time.Hour)}
	st.mu.Unlock()
	return tok
}

func (st *serverState) sessionUser(r *http.Request) (OnlineUser, bool) {
	auth := strings.TrimSpace(r.Header.Get("Authorization"))
	if !strings.HasPrefix(auth, "Bearer ") {
		return OnlineUser{}, false
	}
	tok := strings.TrimSpace(strings.TrimPrefix(auth, "Bearer "))
	st.mu.RLock()
	s, ok := st.Sessions[tok]
	u, uok := st.Users[s.UserID]
	st.mu.RUnlock()
	if !ok || !uok || time.Now().After(s.Expires) || !activeStatus(u.Status) {
		return OnlineUser{}, false
	}
	return u.OnlineUser, true
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func readJSON(r *http.Request, v any) error {
	defer r.Body.Close()
	return json.NewDecoder(io.LimitReader(r.Body, 4<<20)).Decode(v)
}
func apiErr(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, APIError{Error: msg})
}

func (st *serverState) requireSession(w http.ResponseWriter, r *http.Request) (OnlineUser, bool) {
	u, ok := st.sessionUser(r)
	if !ok {
		apiErr(w, 401, "sessão inválida ou expirada")
		return OnlineUser{}, false
	}
	return u, true
}

func (st *serverState) require(w http.ResponseWriter, r *http.Request, roles ...string) (OnlineUser, bool) {
	u, ok := st.requireSession(w, r)
	if !ok {
		return OnlineUser{}, false
	}
	if u.MustChangePassword {
		apiErr(w, 428, "troque a senha temporária antes de continuar")
		return OnlineUser{}, false
	}
	if len(roles) > 0 {
		allow := false
		for _, role := range roles {
			if u.Role == role {
				allow = true
				break
			}
		}
		if !allow {
			apiErr(w, 403, "permissão insuficiente")
			return OnlineUser{}, false
		}
	}
	return u, true
}

func (st *serverState) createInvitation(tutor OnlineUser, classID, name, email, institutionalID string) (serverInvitation, error) {
	name = strings.TrimSpace(name)
	email = strings.ToLower(strings.TrimSpace(email))
	institutionalID = strings.TrimSpace(institutionalID)
	if name == "" || !strings.Contains(email, "@") {
		return serverInvitation{}, errors.New("nome ou e-mail inválido")
	}
	st.mu.Lock()
	defer st.mu.Unlock()
	cl, ok := st.Classes[classID]
	if !ok || cl.TutorID != tutor.ID {
		return serverInvitation{}, errors.New("turma não encontrada")
	}
	for _, inv := range st.Invitations {
		if inv.ClassID == classID && strings.EqualFold(inv.Email, email) && inv.RedeemedAt == "" {
			return inv, nil
		}
	}
	for _, ur := range st.Users {
		if strings.EqualFold(ur.Email, email) {
			found := false
			for _, sid := range cl.StudentIDs {
				if sid == ur.ID {
					found = true
					break
				}
			}
			if !found {
				cl.StudentIDs = append(cl.StudentIDs, ur.ID)
				st.Classes[classID] = cl
			}
			inv := serverInvitation{
				Code:            strings.ToUpper(randomHex(4)),
				Name:            name,
				Email:           email,
				InstitutionalID: institutionalID,
				ClassID:         classID,
				TutorID:         tutor.ID,
				CreatedAt:       time.Now().UTC().Format(time.RFC3339),
				ExpiresAt:       time.Now().UTC().Add(30 * 24 * time.Hour).Format(time.RFC3339),
				RedeemedAt:      time.Now().UTC().Format(time.RFC3339),
				UserID:          ur.ID,
			}
			st.Invitations[inv.Code] = inv
			return inv, st.saveLocked()
		}
	}
	inv := serverInvitation{
		Code:            strings.ToUpper(randomHex(4)),
		Name:            name,
		Email:           email,
		InstitutionalID: institutionalID,
		ClassID:         classID,
		TutorID:         tutor.ID,
		CreatedAt:       time.Now().UTC().Format(time.RFC3339),
		ExpiresAt:       time.Now().UTC().Add(30 * 24 * time.Hour).Format(time.RFC3339),
	}
	st.Invitations[inv.Code] = inv
	return inv, st.saveLocked()
}

func (st *serverState) redeemInvitation(code, password string) (OnlineLoginResponse, error) {
	code = strings.ToUpper(strings.TrimSpace(code))
	if !validPassword(password) {
		return OnlineLoginResponse{}, errors.New("a senha deve ter pelo menos 8 caracteres")
	}
	st.mu.Lock()
	inv, ok := st.Invitations[code]
	if !ok || inv.RedeemedAt != "" || inv.RevokedAt != "" {
		st.mu.Unlock()
		return OnlineLoginResponse{}, errors.New("convite inválido, revogado ou já utilizado")
	}
	if inv.ExpiresAt != "" {
		if exp, err := time.Parse(time.RFC3339, inv.ExpiresAt); err == nil && time.Now().After(exp) {
			st.mu.Unlock()
			return OnlineLoginResponse{}, errors.New("convite expirado")
		}
	}
	for _, ur := range st.Users {
		if strings.EqualFold(ur.Email, inv.Email) {
			st.mu.Unlock()
			return OnlineLoginResponse{}, errors.New("já existe uma conta com este e-mail; entre normalmente")
		}
	}
	id := "usr-" + randomHex(8)
	salt := randomHex(16)
	u := OnlineUser{ID: id, Name: inv.Name, Email: inv.Email, Role: "aluno", Status: "active", InstitutionalID: inv.InstitutionalID, CreatedAt: time.Now().UTC().Format(time.RFC3339)}
	st.Users[id] = serverUserRecord{OnlineUser: u, Salt: salt, Hash: passwordHash(password, salt)}
	cl := st.Classes[inv.ClassID]
	cl.StudentIDs = append(cl.StudentIDs, id)
	st.Classes[inv.ClassID] = cl
	inv.RedeemedAt = time.Now().UTC().Format(time.RFC3339)
	inv.UserID = id
	st.Invitations[code] = inv
	if err := st.saveLocked(); err != nil {
		st.mu.Unlock()
		return OnlineLoginResponse{}, err
	}
	st.mu.Unlock()
	return OnlineLoginResponse{Token: st.newSession(id), User: u}, nil
}

func (st *serverState) hasAdmin() bool {
	st.mu.RLock()
	defer st.mu.RUnlock()
	for _, rec := range st.Users {
		if rec.Role == "admin" {
			return true
		}
	}
	return false
}

func (st *serverState) createInitialAdmin(name, email, password string) (OnlineUser, error) {
	email = strings.ToLower(strings.TrimSpace(email))
	name = strings.TrimSpace(name)
	if name == "" || !strings.Contains(email, "@") {
		return OnlineUser{}, errors.New("nome ou e-mail inválido")
	}
	if !validPassword(password) {
		return OnlineUser{}, errors.New("a senha deve ter pelo menos 8 caracteres")
	}
	st.mu.Lock()
	defer st.mu.Unlock()
	for _, rec := range st.Users {
		if rec.Role == "admin" {
			return OnlineUser{}, errors.New("já existe um administrador; novos administradores devem ser criados pelo painel administrativo")
		}
		if strings.EqualFold(rec.Email, email) {
			return OnlineUser{}, errors.New("já existe uma conta com este e-mail")
		}
	}
	id := "usr-" + randomHex(8)
	salt := randomHex(16)
	u := OnlineUser{
		ID: id, Name: name, Email: email, Role: "admin", Status: "active",
		IsPrimaryAdmin: true, CanInviteMentors: true,
		CreatedAt: time.Now().UTC().Format(time.RFC3339),
	}
	st.Users[id] = serverUserRecord{OnlineUser: u, Salt: salt, Hash: passwordHash(password, salt)}
	return u, st.saveLocked()
}

func (st *serverState) createAdmin(actor OnlineUser, name, email, password string) (OnlineUser, error) {
	if actor.Role != "admin" || !activeStatus(actor.Status) {
		return OnlineUser{}, errors.New("somente administradores podem criar outro administrador")
	}
	return st.createUserProfile(name, email, password, "admin", "", "")
}

func (st *serverState) setUserStatus(actor OnlineUser, userID, status string) (OnlineUser, error) {
	if actor.Role != "admin" {
		return OnlineUser{}, errors.New("somente administradores podem alterar contas")
	}
	status = strings.ToLower(strings.TrimSpace(status))
	if status != "active" && status != "disabled" {
		return OnlineUser{}, errors.New("status inválido")
	}
	st.mu.Lock()
	defer st.mu.Unlock()
	rec, ok := st.Users[userID]
	if !ok {
		return OnlineUser{}, errors.New("usuário não encontrado")
	}
	if rec.ID == actor.ID && status == "disabled" {
		return OnlineUser{}, errors.New("um administrador não pode desativar a própria conta")
	}
	if rec.IsPrimaryAdmin && status == "disabled" {
		return OnlineUser{}, errors.New("transfira a administração principal antes de desativar esta conta")
	}
	rec.Status = status
	st.Users[userID] = rec
	if status == "disabled" {
		for tok, sess := range st.Sessions {
			if sess.UserID == userID {
				delete(st.Sessions, tok)
			}
		}
	}
	if err := st.saveLocked(); err != nil {
		return OnlineUser{}, err
	}
	return rec.OnlineUser, nil
}

func (st *serverState) deleteUserAsPrimary(actor OnlineUser, userID string) (OnlineUser, error) {
	if actor.Role != "admin" || !actor.IsPrimaryAdmin {
		return OnlineUser{}, errors.New("somente o administrador principal pode excluir contas")
	}
	userID = strings.TrimSpace(userID)
	if userID == "" {
		return OnlineUser{}, errors.New("usuário não informado")
	}

	st.mu.Lock()
	defer st.mu.Unlock()
	rec, ok := st.Users[userID]
	if !ok {
		return OnlineUser{}, errors.New("usuário não encontrado")
	}
	if rec.ID == actor.ID || rec.IsPrimaryAdmin {
		return OnlineUser{}, errors.New("o administrador principal não pode excluir a própria conta")
	}
	if isMentorRole(rec.Role) {
		owned := 0
		for _, cl := range st.Classes {
			if cl.TutorID == rec.ID {
				owned++
			}
		}
		if owned > 0 {
			return OnlineUser{}, fmt.Errorf("este Mentor possui %d turma(s); exclua as turmas antes de excluir a conta", owned)
		}
	}

	for id, cl := range st.Classes {
		if containsStudentID(cl.StudentIDs, rec.ID) {
			cl.StudentIDs = removeStudentID(cl.StudentIDs, rec.ID)
			st.Classes[id] = cl
		}
	}
	for id, company := range st.Companies {
		if company.OwnerID == rec.ID {
			delete(st.Companies, id)
		}
	}
	for id, scenario := range st.Scenarios {
		if scenario.TutorID == rec.ID {
			delete(st.Scenarios, id)
		}
	}
	for code, inv := range st.Invitations {
		if inv.UserID == rec.ID || inv.TutorID == rec.ID || strings.EqualFold(inv.Email, rec.Email) {
			delete(st.Invitations, code)
		}
	}
	for code, inv := range st.MentorInvitations {
		if inv.UserID == rec.ID || inv.IssuerID == rec.ID || strings.EqualFold(inv.Email, rec.Email) {
			delete(st.MentorInvitations, code)
		}
	}
	for token, session := range st.Sessions {
		if session.UserID == rec.ID {
			delete(st.Sessions, token)
		}
	}
	delete(st.Users, rec.ID)
	if err := st.saveLocked(); err != nil {
		return OnlineUser{}, err
	}
	return rec.OnlineUser, nil
}

func (st *serverState) deleteClassAsPrimary(actor OnlineUser, classID string) (OnlineClass, int, error) {
	if actor.Role != "admin" || !actor.IsPrimaryAdmin {
		return OnlineClass{}, 0, errors.New("somente o administrador principal pode excluir turmas")
	}
	classID = strings.TrimSpace(classID)
	if classID == "" {
		return OnlineClass{}, 0, errors.New("turma não informada")
	}

	st.mu.Lock()
	defer st.mu.Unlock()
	cl, ok := st.Classes[classID]
	if !ok {
		return OnlineClass{}, 0, errors.New("turma não encontrada")
	}
	delete(st.Classes, classID)
	for code, inv := range st.Invitations {
		if inv.ClassID == classID {
			delete(st.Invitations, code)
		}
	}

	now := time.Now().UTC().Format(time.RFC3339)
	for id, rec := range st.Users {
		if rec.Role != "aluno" {
			continue
		}
		changed := false
		for i := range rec.Enrollments {
			if rec.Enrollments[i].ClassID == classID && rec.Enrollments[i].EndedAt == "" {
				rec.Enrollments[i].EndedAt = now
				changed = true
			}
		}
		if rec.CurrentClassID == classID {
			rec.CurrentClassID = ""
			rec.EnrollmentID = ""
			rec.EnrollmentStatus = "waiting"
			changed = true
		}
		if changed {
			st.Users[id] = rec
		}
	}

	detached := 0
	for id, company := range st.Companies {
		if company.ClassID != classID {
			continue
		}
		company.ClassID = ""
		company.Company.TurmaID = ""
		company.Revision++
		company.UpdatedAt = now
		st.Companies[id] = company
		detached++
	}
	if err := st.saveLocked(); err != nil {
		return OnlineClass{}, 0, err
	}
	return cl, detached, nil
}

func (st *serverState) setMentorPermission(actor OnlineUser, userID string, allowed bool) (OnlineUser, error) {
	if actor.Role != "admin" {
		return OnlineUser{}, errors.New("somente administradores podem alterar permissões de mentor")
	}
	st.mu.Lock()
	defer st.mu.Unlock()
	rec, ok := st.Users[userID]
	if !ok {
		return OnlineUser{}, errors.New("usuário não encontrado")
	}
	if !isMentorRole(rec.Role) {
		return OnlineUser{}, errors.New("a permissão só pode ser aplicada a um mentor")
	}
	rec.CanInviteMentors = allowed
	st.Users[userID] = rec
	if err := st.saveLocked(); err != nil {
		return OnlineUser{}, err
	}
	return rec.OnlineUser, nil
}

func (st *serverState) transferPrimaryAdmin(actor OnlineUser, targetID string) (OnlineUser, error) {
	if actor.Role != "admin" || !actor.IsPrimaryAdmin {
		return OnlineUser{}, errors.New("somente o administrador principal pode transferir a administração principal")
	}
	st.mu.Lock()
	defer st.mu.Unlock()
	target, ok := st.Users[targetID]
	if !ok || target.Role != "admin" || !activeStatus(target.Status) {
		return OnlineUser{}, errors.New("o destino deve ser um administrador ativo")
	}
	current := st.Users[actor.ID]
	current.IsPrimaryAdmin = false
	st.Users[actor.ID] = current
	target.IsPrimaryAdmin = true
	target.CanInviteMentors = true
	st.Users[targetID] = target
	if err := st.saveLocked(); err != nil {
		return OnlineUser{}, err
	}
	return target.OnlineUser, nil
}

func (st *serverState) resetAdminPasswordLocal(email, password string) error {
	email = strings.ToLower(strings.TrimSpace(email))
	if !validPassword(password) {
		return errors.New("a senha deve ter pelo menos 8 caracteres")
	}
	st.mu.Lock()
	defer st.mu.Unlock()
	for id, rec := range st.Users {
		if rec.Role == "admin" && strings.EqualFold(rec.Email, email) {
			rec.Salt = randomHex(16)
			rec.Hash = passwordHash(password, rec.Salt)
			rec.Status = "active"
			st.Users[id] = rec
			for tok, sess := range st.Sessions {
				if sess.UserID == id {
					delete(st.Sessions, tok)
				}
			}
			return st.saveLocked()
		}
	}
	return errors.New("administrador não encontrado")
}

func mentorCredentialCode() string {
	return "MTR-" + strings.ToUpper(randomHex(2)) + "-" + strings.ToUpper(randomHex(2))
}

func (st *serverState) createMentorInvitation(issuer OnlineUser, name, email, institution, institutionalID string) (serverMentorInvitation, error) {
	if !(isAdminRole(issuer.Role) || (isMentorRole(issuer.Role) && issuer.CanInviteMentors)) {
		return serverMentorInvitation{}, errors.New("esta conta não possui permissão para credenciar mentores")
	}
	name = strings.TrimSpace(name)
	email = strings.ToLower(strings.TrimSpace(email))
	institution = strings.TrimSpace(institution)
	institutionalID = strings.TrimSpace(institutionalID)
	if name == "" || !strings.Contains(email, "@") {
		return serverMentorInvitation{}, errors.New("nome ou e-mail inválido")
	}
	st.mu.Lock()
	defer st.mu.Unlock()
	for _, ur := range st.Users {
		if strings.EqualFold(ur.Email, email) {
			return serverMentorInvitation{}, errors.New("já existe uma conta com este e-mail")
		}
	}
	for _, inv := range st.MentorInvitations {
		if inv.IssuerID == issuer.ID && strings.EqualFold(inv.Email, email) &&
			inv.RedeemedAt == "" && inv.RevokedAt == "" {
			if inv.ExpiresAt == "" {
				return inv, nil
			}
			if exp, err := time.Parse(time.RFC3339, inv.ExpiresAt); err != nil || time.Now().Before(exp) {
				return inv, nil
			}
		}
	}
	code := mentorCredentialCode()
	for {
		if _, exists := st.MentorInvitations[code]; !exists {
			break
		}
		code = mentorCredentialCode()
	}
	inv := serverMentorInvitation{
		Code: code, Name: name, Email: email, Institution: institution,
		InstitutionalID: institutionalID, IssuerID: issuer.ID,
		CreatedAt: time.Now().UTC().Format(time.RFC3339),
		ExpiresAt: time.Now().UTC().Add(30 * 24 * time.Hour).Format(time.RFC3339),
	}
	st.MentorInvitations[code] = inv
	return inv, st.saveLocked()
}

func (st *serverState) redeemMentorInvitation(code, name, email, password, institution, institutionalID string) (OnlineLoginResponse, error) {
	code = strings.ToUpper(strings.TrimSpace(code))
	email = strings.ToLower(strings.TrimSpace(email))
	name = strings.TrimSpace(name)
	institution = strings.TrimSpace(institution)
	institutionalID = strings.TrimSpace(institutionalID)
	if name == "" || !strings.Contains(email, "@") {
		return OnlineLoginResponse{}, errors.New("nome ou e-mail inválido")
	}
	if !validPassword(password) {
		return OnlineLoginResponse{}, errors.New("a senha deve ter pelo menos 8 caracteres")
	}

	st.mu.Lock()
	inv, ok := st.MentorInvitations[code]
	if !ok || inv.RedeemedAt != "" || inv.RevokedAt != "" {
		st.mu.Unlock()
		return OnlineLoginResponse{}, errors.New("credencial de mentor inválida, revogada ou já utilizada")
	}
	if inv.ExpiresAt != "" {
		if exp, err := time.Parse(time.RFC3339, inv.ExpiresAt); err == nil && time.Now().After(exp) {
			st.mu.Unlock()
			return OnlineLoginResponse{}, errors.New("credencial de mentor expirada")
		}
	}
	if !strings.EqualFold(inv.Email, email) {
		st.mu.Unlock()
		return OnlineLoginResponse{}, errors.New("esta credencial foi emitida para outro e-mail")
	}
	for _, ur := range st.Users {
		if strings.EqualFold(ur.Email, email) {
			st.mu.Unlock()
			return OnlineLoginResponse{}, errors.New("já existe uma conta com este e-mail; não é possível promover uma conta existente")
		}
	}
	if inv.Institution != "" {
		institution = inv.Institution
	}
	if inv.InstitutionalID != "" {
		institutionalID = inv.InstitutionalID
	}
	id := "usr-" + randomHex(8)
	salt := randomHex(16)
	u := OnlineUser{
		ID: id, Name: name, Email: email, Role: "mentor", Status: "active",
		Institution: institution, InstitutionalID: institutionalID,
		CanInviteMentors: false,
		CreatedAt:        time.Now().UTC().Format(time.RFC3339),
	}
	st.Users[id] = serverUserRecord{OnlineUser: u, Salt: salt, Hash: passwordHash(password, salt)}
	inv.RedeemedAt = time.Now().UTC().Format(time.RFC3339)
	inv.UserID = id
	st.MentorInvitations[code] = inv
	if err := st.saveLocked(); err != nil {
		st.mu.Unlock()
		return OnlineLoginResponse{}, err
	}
	st.mu.Unlock()
	return OnlineLoginResponse{Token: st.newSession(id), User: u}, nil
}

func (st *serverState) handler() http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, 200, map[string]any{"ok": true, "service": "JED Servidor", "version": version})
	})
	mux.HandleFunc("/api/v1/login", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			apiErr(w, 405, "método não permitido")
			return
		}
		var in struct{ Email, Password string }
		if readJSON(r, &in) != nil {
			apiErr(w, 400, "JSON inválido")
			return
		}
		host, _, _ := net.SplitHostPort(r.RemoteAddr)
		key := strings.ToLower(strings.TrimSpace(in.Email)) + "|" + host
		if !loginAllowed(key) {
			apiErr(w, 429, "muitas tentativas de login; aguarde alguns minutos")
			return
		}
		u, ok := st.authenticate(in.Email, in.Password)
		if !ok {
			recordLoginFailure(key)
			apiErr(w, 401, "e-mail ou senha inválidos")
			return
		}
		clearLoginFailures(key)
		writeJSON(w, 200, OnlineLoginResponse{Token: st.newSession(u.ID), User: u})
	})
	mux.HandleFunc("/api/v1/logout", func(w http.ResponseWriter, r *http.Request) {
		auth := strings.TrimSpace(strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer "))
		st.mu.Lock()
		delete(st.Sessions, auth)
		st.mu.Unlock()
		writeJSON(w, 200, map[string]bool{"ok": true})
	})
	mux.HandleFunc("/api/v1/me", func(w http.ResponseWriter, r *http.Request) {
		u, ok := st.requireSession(w, r)
		if !ok {
			return
		}
		writeJSON(w, 200, u)
	})

	mux.HandleFunc("/api/v1/register/student", func(w http.ResponseWriter, r *http.Request) {
		apiErr(w, 403, "cadastro de Aluno é realizado exclusivamente por Administradores")
	})
	mux.HandleFunc("/api/v1/register/mentor", func(w http.ResponseWriter, r *http.Request) {
		apiErr(w, 403, "cadastro de Mentor é realizado por Administradores")
	})
	mux.HandleFunc("/api/v1/invitations/redeem", func(w http.ResponseWriter, r *http.Request) {
		apiErr(w, 403, "ativação por convite foi desativada; contas de Aluno são criadas por Administradores")
	})
	mux.HandleFunc("/api/v1/invitations", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "POST" {
			apiErr(w, 403, "Mentores não podem cadastrar Alunos; use o painel do Administrador")
			return
		}
		u, ok := st.require(w, r, "tutor", "mentor")
		if !ok {
			return
		}
		switch r.Method {
		case "POST":
			var in struct {
				ClassID         string `json:"class_id"`
				Name            string `json:"name"`
				Email           string `json:"email"`
				InstitutionalID string `json:"institutional_id"`
			}
			if readJSON(r, &in) != nil {
				apiErr(w, 400, "JSON inválido")
				return
			}
			inv, err := st.createInvitation(u, in.ClassID, in.Name, in.Email, in.InstitutionalID)
			if err != nil {
				apiErr(w, 400, err.Error())
				return
			}
			writeJSON(w, 201, inv)
		case "GET":
			classID := strings.TrimSpace(r.URL.Query().Get("class_id"))
			st.mu.RLock()
			out := []serverInvitation{}
			for _, inv := range st.Invitations {
				if inv.TutorID == u.ID && (classID == "" || inv.ClassID == classID) {
					out = append(out, inv)
				}
			}
			st.mu.RUnlock()
			writeJSON(w, 200, out)
		default:
			apiErr(w, 405, "método não permitido")
		}
	})

	mux.HandleFunc("/api/v1/mentor-invitations", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "POST" {
			apiErr(w, 403, "cadastro de Mentor é realizado pelo painel do Administrador")
			return
		}
		u, ok := st.require(w, r, "tutor", "mentor", "admin")
		if !ok {
			return
		}
		switch r.Method {
		case "POST":
			var in struct {
				Name            string `json:"name"`
				Email           string `json:"email"`
				Institution     string `json:"institution"`
				InstitutionalID string `json:"institutional_id"`
			}
			if readJSON(r, &in) != nil {
				apiErr(w, 400, "JSON inválido")
				return
			}
			inv, err := st.createMentorInvitation(u, in.Name, in.Email, in.Institution, in.InstitutionalID)
			if err != nil {
				apiErr(w, 400, err.Error())
				return
			}
			// A credencial é persistida antes do envio. Falha de e-mail nunca invalida o código.
			inv = st.deliverMentorInvitationEmail(inv)
			writeJSON(w, 201, inv)
		case "GET":
			st.mu.RLock()
			out := []serverMentorInvitation{}
			for _, inv := range st.MentorInvitations {
				if u.Role == "admin" || inv.IssuerID == u.ID {
					out = append(out, inv)
				}
			}
			st.mu.RUnlock()
			writeJSON(w, 200, out)
		default:
			apiErr(w, 405, "método não permitido")
		}
	})
	mux.HandleFunc("/api/v1/mentor-invitations/revoke", func(w http.ResponseWriter, r *http.Request) {
		u, ok := st.require(w, r, "tutor", "mentor", "admin")
		if !ok {
			return
		}
		if r.Method != "POST" {
			apiErr(w, 405, "método não permitido")
			return
		}
		var in struct {
			Code string `json:"code"`
		}
		if readJSON(r, &in) != nil {
			apiErr(w, 400, "JSON inválido")
			return
		}
		code := strings.ToUpper(strings.TrimSpace(in.Code))
		st.mu.Lock()
		inv, exists := st.MentorInvitations[code]
		if !exists || (u.Role != "admin" && inv.IssuerID != u.ID) {
			st.mu.Unlock()
			apiErr(w, 404, "credencial de mentor não encontrada")
			return
		}
		if inv.RedeemedAt != "" {
			st.mu.Unlock()
			apiErr(w, 400, "credencial já utilizada")
			return
		}
		inv.RevokedAt = time.Now().UTC().Format(time.RFC3339)
		st.MentorInvitations[code] = inv
		err := st.saveLocked()
		st.mu.Unlock()
		if err != nil {
			apiErr(w, 500, "erro ao revogar credencial")
			return
		}
		writeJSON(w, 200, inv)
	})

	mux.HandleFunc("/api/v1/admin/classes", func(w http.ResponseWriter, r *http.Request) {
		actor, ok := st.require(w, r, "admin")
		if !ok {
			return
		}
		switch r.Method {
		case "GET":
			st.mu.RLock()
			out := make([]OnlineClass, 0, len(st.Classes))
			for _, cl := range st.Classes {
				out = append(out, cl)
			}
			st.mu.RUnlock()
			sort.Slice(out, func(i, j int) bool { return out[i].Number < out[j].Number })
			writeJSON(w, 200, out)
		case "POST":
			var in struct {
				Name     string `json:"name"`
				MentorID string `json:"mentor_id"`
			}
			if readJSON(r, &in) != nil {
				apiErr(w, 400, "JSON inválido")
				return
			}
			cl, err := st.createAdminClass(actor, in.Name, in.MentorID)
			if err != nil {
				apiErr(w, 400, err.Error())
				return
			}
			writeJSON(w, 201, cl)
		default:
			apiErr(w, 405, "método não permitido")
		}
	})

	mux.HandleFunc("/api/v1/admin/students/assign-class", func(w http.ResponseWriter, r *http.Request) {
		actor, ok := st.require(w, r, "admin")
		if !ok {
			return
		}
		if r.Method != "POST" {
			apiErr(w, 405, "método não permitido")
			return
		}
		var in struct {
			UserID  string `json:"user_id"`
			ClassID string `json:"class_id"`
		}
		if readJSON(r, &in) != nil {
			apiErr(w, 400, "JSON inválido")
			return
		}
		u, err := st.assignStudentClass(actor, in.UserID, in.ClassID)
		if err != nil {
			apiErr(w, 400, err.Error())
			return
		}
		writeJSON(w, 200, u)
	})

	mux.HandleFunc("/api/v1/admin/users/delete", func(w http.ResponseWriter, r *http.Request) {
		actor, ok := st.require(w, r, "admin")
		if !ok {
			return
		}
		if r.Method != "POST" {
			apiErr(w, 405, "método não permitido")
			return
		}
		var in struct {
			UserID string `json:"user_id"`
		}
		if readJSON(r, &in) != nil {
			apiErr(w, 400, "JSON inválido")
			return
		}
		deleted, err := st.deleteUserAsPrimary(actor, in.UserID)
		if err != nil {
			apiErr(w, 400, err.Error())
			return
		}
		writeJSON(w, 200, map[string]any{"deleted": deleted})
	})

	mux.HandleFunc("/api/v1/admin/classes/delete", func(w http.ResponseWriter, r *http.Request) {
		actor, ok := st.require(w, r, "admin")
		if !ok {
			return
		}
		if r.Method != "POST" {
			apiErr(w, 405, "método não permitido")
			return
		}
		var in struct {
			ClassID string `json:"class_id"`
		}
		if readJSON(r, &in) != nil {
			apiErr(w, 400, "JSON inválido")
			return
		}
		deleted, detached, err := st.deleteClassAsPrimary(actor, in.ClassID)
		if err != nil {
			apiErr(w, 400, err.Error())
			return
		}
		writeJSON(w, 200, map[string]any{"deleted": deleted, "detached_companies": detached})
	})

	mux.HandleFunc("/api/v1/admin/users", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" {
			apiErr(w, 405, "método não permitido")
			return
		}
		if _, ok := st.require(w, r, "admin"); !ok {
			return
		}
		st.mu.RLock()
		out := make([]OnlineUser, 0, len(st.Users))
		for _, rec := range st.Users {
			out = append(out, rec.OnlineUser)
		}
		st.mu.RUnlock()
		sort.Slice(out, func(i, j int) bool {
			if out[i].Role != out[j].Role {
				return out[i].Role < out[j].Role
			}
			return strings.ToLower(out[i].Name) < strings.ToLower(out[j].Name)
		})
		writeJSON(w, 200, out)
	})

	mux.HandleFunc("/api/v1/admin/create-admin", func(w http.ResponseWriter, r *http.Request) {
		actor, ok := st.require(w, r, "admin")
		if !ok {
			return
		}
		if r.Method != "POST" {
			apiErr(w, 405, "método não permitido")
			return
		}
		var in struct {
			Name     string `json:"name"`
			Email    string `json:"email"`
			Password string `json:"password"`
		}
		if readJSON(r, &in) != nil {
			apiErr(w, 400, "JSON inválido")
			return
		}
		u, err := st.createProvisionedUser(actor, in.Name, in.Email, "admin", "", "")
		if err != nil {
			apiErr(w, 400, err.Error())
			return
		}
		writeJSON(w, 201, u)
	})

	mux.HandleFunc("/api/v1/admin/users/create", func(w http.ResponseWriter, r *http.Request) {
		actor, ok := st.require(w, r, "admin")
		if !ok {
			return
		}
		if r.Method != "POST" {
			apiErr(w, 405, "método não permitido")
			return
		}
		var in struct {
			Name            string `json:"name"`
			Email           string `json:"email"`
			Role            string `json:"role"`
			Institution     string `json:"institution"`
			InstitutionalID string `json:"institutional_id"`
			ClassID         string `json:"class_id"`
		}
		if readJSON(r, &in) != nil {
			apiErr(w, 400, "JSON inválido")
			return
		}
		u, err := st.createProvisionedUserForClass(actor, in.Name, in.Email, in.Role, in.Institution, in.InstitutionalID, in.ClassID)
		if err != nil {
			apiErr(w, 400, err.Error())
			return
		}
		writeJSON(w, 201, u)
	})

	mux.HandleFunc("/api/v1/admin/users/import", func(w http.ResponseWriter, r *http.Request) {
		actor, ok := st.require(w, r, "admin")
		if !ok {
			return
		}
		if r.Method != "POST" {
			apiErr(w, 405, "método não permitido")
			return
		}
		var in struct {
			Users []struct {
				Name            string `json:"name"`
				Email           string `json:"email"`
				Role            string `json:"role"`
				Institution     string `json:"institution"`
				InstitutionalID string `json:"institutional_id"`
				ClassID         string `json:"class_id"`
				ClassRef        string `json:"class_ref"`
			} `json:"users"`
		}
		if readJSON(r, &in) != nil {
			apiErr(w, 400, "JSON inválido")
			return
		}
		if len(in.Users) == 0 {
			apiErr(w, 400, "a lista está vazia")
			return
		}
		if len(in.Users) > 1000 {
			apiErr(w, 400, "a importação aceita no máximo 1000 usuários por arquivo")
			return
		}
		created := []OnlineUser{}
		errorsOut := []map[string]any{}
		for i, row := range in.Users {
			classID := strings.TrimSpace(row.ClassID)
			if classID == "" && strings.TrimSpace(row.ClassRef) != "" {
				var err error
				classID, err = st.resolveClassRef(row.ClassRef)
				if err != nil {
					errorsOut = append(errorsOut, map[string]any{"row": i + 2, "email": row.Email, "error": err.Error()})
					continue
				}
			}
			u, err := st.createProvisionedUserForClass(actor, row.Name, row.Email, row.Role, row.Institution, row.InstitutionalID, classID)
			if err != nil {
				errorsOut = append(errorsOut, map[string]any{"row": i + 2, "email": row.Email, "error": err.Error()})
				continue
			}
			created = append(created, u)
		}
		writeJSON(w, 200, map[string]any{"created": created, "errors": errorsOut, "temporary_password": defaultProvisionedPassword})
	})

	mux.HandleFunc("/api/v1/admin/users/resend-email", func(w http.ResponseWriter, r *http.Request) {
		_, ok := st.require(w, r, "admin")
		if !ok {
			return
		}
		if r.Method != "POST" {
			apiErr(w, 405, "método não permitido")
			return
		}
		var in struct {
			UserID string `json:"user_id"`
		}
		if readJSON(r, &in) != nil {
			apiErr(w, 400, "JSON inválido")
			return
		}
		st.mu.RLock()
		rec, exists := st.Users[in.UserID]
		st.mu.RUnlock()
		if !exists {
			apiErr(w, 404, "usuário não encontrado")
			return
		}
		if !rec.MustChangePassword {
			apiErr(w, 400, "a senha temporária já foi substituída; não é possível reenviar essa credencial")
			return
		}
		writeJSON(w, 200, st.deliverAccountCreatedEmail(rec.OnlineUser, defaultProvisionedPassword))
	})

	mux.HandleFunc("/api/v1/admin/email-status", func(w http.ResponseWriter, r *http.Request) {
		if _, ok := st.require(w, r, "admin"); !ok {
			return
		}
		if r.Method != "GET" {
			apiErr(w, 405, "método não permitido")
			return
		}
		writeJSON(w, 200, emailConfigurationStatus())
	})

	mux.HandleFunc("/api/v1/admin/user-status", func(w http.ResponseWriter, r *http.Request) {
		actor, ok := st.require(w, r, "admin")
		if !ok {
			return
		}
		if r.Method != "POST" {
			apiErr(w, 405, "método não permitido")
			return
		}
		var in struct {
			UserID string `json:"user_id"`
			Status string `json:"status"`
		}
		if readJSON(r, &in) != nil {
			apiErr(w, 400, "JSON inválido")
			return
		}
		u, err := st.setUserStatus(actor, in.UserID, in.Status)
		if err != nil {
			apiErr(w, 400, err.Error())
			return
		}
		writeJSON(w, 200, u)
	})

	mux.HandleFunc("/api/v1/admin/mentor-permission", func(w http.ResponseWriter, r *http.Request) {
		actor, ok := st.require(w, r, "admin")
		if !ok {
			return
		}
		if r.Method != "POST" {
			apiErr(w, 405, "método não permitido")
			return
		}
		var in struct {
			UserID  string `json:"user_id"`
			Allowed bool   `json:"allowed"`
		}
		if readJSON(r, &in) != nil {
			apiErr(w, 400, "JSON inválido")
			return
		}
		u, err := st.setMentorPermission(actor, in.UserID, in.Allowed)
		if err != nil {
			apiErr(w, 400, err.Error())
			return
		}
		writeJSON(w, 200, u)
	})

	mux.HandleFunc("/api/v1/admin/transfer-primary", func(w http.ResponseWriter, r *http.Request) {
		actor, ok := st.require(w, r, "admin")
		if !ok {
			return
		}
		if r.Method != "POST" {
			apiErr(w, 405, "método não permitido")
			return
		}
		var in struct {
			UserID string `json:"user_id"`
		}
		if readJSON(r, &in) != nil {
			apiErr(w, 400, "JSON inválido")
			return
		}
		u, err := st.transferPrimaryAdmin(actor, in.UserID)
		if err != nil {
			apiErr(w, 400, err.Error())
			return
		}
		writeJSON(w, 200, u)
	})

	mux.HandleFunc("/api/v1/users/student", func(w http.ResponseWriter, r *http.Request) {
		apiErr(w, 403, "cadastro de Aluno é realizado exclusivamente por Administradores")
	})
	mux.HandleFunc("/api/v1/password", func(w http.ResponseWriter, r *http.Request) {
		u, ok := st.requireSession(w, r)
		if !ok {
			return
		}
		if r.Method != "POST" {
			apiErr(w, 405, "método não permitido")
			return
		}
		var in struct {
			Current string `json:"current"`
			New     string `json:"new"`
		}
		if readJSON(r, &in) != nil {
			apiErr(w, 400, "JSON inválido")
			return
		}
		if !validPassword(in.New) {
			apiErr(w, 400, "a nova senha deve ter pelo menos 8 caracteres")
			return
		}
		st.mu.Lock()
		rec := st.Users[u.ID]
		if !hmac.Equal([]byte(passwordHash(in.Current, rec.Salt)), []byte(rec.Hash)) {
			st.mu.Unlock()
			apiErr(w, 401, "senha atual incorreta")
			return
		}
		rec.Salt = randomHex(16)
		rec.Hash = passwordHash(in.New, rec.Salt)
		rec.MustChangePassword = false
		st.Users[u.ID] = rec
		err := st.saveLocked()
		st.mu.Unlock()
		if err != nil {
			apiErr(w, 500, "erro ao salvar senha")
			return
		}
		writeJSON(w, 200, map[string]bool{"ok": true})
	})

	mux.HandleFunc("/api/v1/invitations/revoke", func(w http.ResponseWriter, r *http.Request) {
		u, ok := st.require(w, r, "tutor", "mentor")
		if !ok {
			return
		}
		if r.Method != "POST" {
			apiErr(w, 405, "método não permitido")
			return
		}
		var in struct {
			Code string `json:"code"`
		}
		if readJSON(r, &in) != nil {
			apiErr(w, 400, "JSON inválido")
			return
		}
		code := strings.ToUpper(strings.TrimSpace(in.Code))
		st.mu.Lock()
		inv, exists := st.Invitations[code]
		if !exists || inv.TutorID != u.ID {
			st.mu.Unlock()
			apiErr(w, 404, "convite não encontrado")
			return
		}
		if inv.RedeemedAt != "" {
			st.mu.Unlock()
			apiErr(w, 400, "convite já utilizado")
			return
		}
		inv.RevokedAt = time.Now().UTC().Format(time.RFC3339)
		st.Invitations[code] = inv
		err := st.saveLocked()
		st.mu.Unlock()
		if err != nil {
			apiErr(w, 500, "erro ao revogar convite")
			return
		}
		writeJSON(w, 200, inv)
	})

	mux.HandleFunc("/api/v1/mentor/students", func(w http.ResponseWriter, r *http.Request) {
		u, ok := st.require(w, r, "tutor", "mentor")
		if !ok {
			return
		}
		if r.Method != "GET" {
			apiErr(w, 405, "método não permitido")
			return
		}
		st.mu.RLock()
		ids := map[string]bool{}
		for _, cl := range st.Classes {
			if cl.TutorID == u.ID {
				for _, id := range cl.StudentIDs {
					ids[id] = true
				}
			}
		}
		out := []OnlineUser{}
		for id := range ids {
			if rec, exists := st.Users[id]; exists {
				out = append(out, rec.OnlineUser)
			}
		}
		st.mu.RUnlock()
		sort.Slice(out, func(i, j int) bool { return strings.ToLower(out[i].Name) < strings.ToLower(out[j].Name) })
		writeJSON(w, 200, out)
	})

	mux.HandleFunc("/api/v1/scenarios", func(w http.ResponseWriter, r *http.Request) {
		u, ok := st.require(w, r, "tutor", "mentor")
		if !ok {
			return
		}
		switch r.Method {
		case "GET":
			out := []OnlineScenario{}
			baseIDs := []string{"base-estavel", "base-aquecido", "base-desaceleracao", "base-concorrencia"}
			for i, scenario := range cenariosBase {
				if scenario.Duracao == 0 {
					scenario.Duracao = 12
				}
				if scenario.Dificuldade == "" {
					scenario.Dificuldade = "intermediario"
				}
				if scenario.ConcorrenciaNivel == "" {
					scenario.ConcorrenciaNivel = "media"
				}
				if scenario.ConcorrenciaIndice == 0 {
					scenario.ConcorrenciaIndice = 1
				}
				out = append(out, OnlineScenario{ID: baseIDs[i], Builtin: true, Scenario: scenario})
			}
			st.mu.RLock()
			for _, scenario := range st.Scenarios {
				if scenario.TutorID == u.ID {
					out = append(out, scenario)
				}
			}
			st.mu.RUnlock()
			sort.SliceStable(out, func(i, j int) bool {
				if out[i].Builtin != out[j].Builtin {
					return out[i].Builtin
				}
				return strings.ToLower(out[i].Scenario.Nome) < strings.ToLower(out[j].Scenario.Nome)
			})
			writeJSON(w, 200, out)
		case "POST":
			var in Cenario
			if readJSON(r, &in) != nil || strings.TrimSpace(in.Nome) == "" {
				apiErr(w, 400, "dados inválidos")
				return
			}
			in.Nome = strings.TrimSpace(in.Nome)
			if in.Alcance <= 0 {
				in.Alcance = 1
			}
			if in.Conversao <= 0 {
				in.Conversao = 1
			}
			if in.Oscilacao < 0 {
				in.Oscilacao = 0
			}
			if in.Duracao <= 0 {
				in.Duracao = 12
			}
			if in.Dificuldade == "" {
				in.Dificuldade = "intermediario"
			}
			if in.ConcorrenciaNivel == "" {
				in.ConcorrenciaNivel = "media"
			}
			if in.ConcorrenciaIndice <= 0 {
				in.ConcorrenciaIndice = 1
			}
			now := time.Now().UTC().Format(time.RFC3339)
			in.LocalID = "scn-" + randomHex(8)
			in.CreatedAt = now
			in.UpdatedAt = now
			in.Revision = 1
			entry := OnlineScenario{ID: in.LocalID, TutorID: u.ID, Scenario: in, CreatedAt: now}
			st.mu.Lock()
			st.Scenarios[entry.ID] = entry
			err := st.saveLocked()
			st.mu.Unlock()
			if err != nil {
				apiErr(w, 500, "erro ao salvar cenário")
				return
			}
			writeJSON(w, 201, entry)
		default:
			apiErr(w, 405, "método não permitido")
		}
	})

	mux.HandleFunc("/api/v1/classes", func(w http.ResponseWriter, r *http.Request) {
		u, ok := st.require(w, r)
		if !ok {
			return
		}
		switch r.Method {
		case "GET":
			st.mu.RLock()
			out := []OnlineClass{}
			for _, c := range st.Classes {
				if u.Role == "admin" {
					out = append(out, c)
					continue
				}
				if isMentorRole(u.Role) && c.TutorID == u.ID {
					out = append(out, c)
				}
				if u.Role == "aluno" {
					for _, sid := range c.StudentIDs {
						if sid == u.ID {
							out = append(out, c)
							break
						}
					}
				}
			}
			st.mu.RUnlock()
			sort.Slice(out, func(i, j int) bool { return out[i].Number < out[j].Number })
			writeJSON(w, 200, out)
		case "POST":
			apiErr(w, 403, "turmas são criadas e nomeadas exclusivamente por Administradores")
		default:
			apiErr(w, 405, "método não permitido")
		}
	})
	mux.HandleFunc("/api/v1/classes/join", func(w http.ResponseWriter, r *http.Request) {
		if _, ok := st.require(w, r, "aluno"); !ok {
			return
		}
		apiErr(w, 403, "o vínculo com turmas é administrado centralmente; procure o Administrador")
	})
	mux.HandleFunc("/api/v1/companies", func(w http.ResponseWriter, r *http.Request) {
		u, ok := st.require(w, r)
		if !ok {
			return
		}
		if r.Method != "GET" {
			apiErr(w, 405, "método não permitido")
			return
		}
		st.mu.RLock()
		out := []RemoteCompany{}
		for _, c := range st.Companies {
			if u.Role == "aluno" && c.OwnerID == u.ID {
				out = append(out, c)
			}
			if isMentorRole(u.Role) {
				if cl, ok := st.Classes[c.ClassID]; ok && cl.TutorID == u.ID {
					out = append(out, c)
				}
			}
		}
		st.mu.RUnlock()
		writeJSON(w, 200, out)
	})
	mux.HandleFunc("/api/v1/companies/", func(w http.ResponseWriter, r *http.Request) {
		u, ok := st.require(w, r, "aluno")
		if !ok {
			return
		}
		if r.Method != "PUT" {
			apiErr(w, 405, "método não permitido")
			return
		}
		localID := strings.TrimSpace(strings.TrimPrefix(r.URL.Path, "/api/v1/companies/"))
		if localID == "" || strings.Contains(localID, "/") {
			apiErr(w, 400, "ID inválido")
			return
		}
		var in struct {
			ClassID string  `json:"class_id"`
			Company Empresa `json:"company"`
		}
		if readJSON(r, &in) != nil {
			apiErr(w, 400, "JSON inválido")
			return
		}
		in.Company.LocalID = localID
		in.Company.Responsavel = u.Name
		in.Company.SyncState = "synced"
		now := time.Now().UTC().Format(time.RFC3339)
		key := u.ID + ":" + localID
		st.mu.Lock()
		prev, exists := st.Companies[key]
		rev := 1
		if exists {
			rev = prev.Revision + 1
		}
		// W7.3: class membership is centrally managed. A company already tied
		// to a class keeps that historical link after a student transfer; a new
		// or previously unlinked company inherits the student's current class.
		classID := ""
		if exists && prev.ClassID != "" {
			classID = prev.ClassID
		} else {
			classID = u.CurrentClassID
		}
		in.Company.TurmaID = classID
		in.Company.Revision = rev
		in.Company.UpdatedAt = now
		rc := RemoteCompany{ID: key, OwnerID: u.ID, ClassID: classID, Revision: rev, UpdatedAt: now, Company: in.Company}
		if exists {
			rc.ApprovalStatus = prev.ApprovalStatus
			rc.MentorComment = prev.MentorComment
			rc.EvaluatedAt = prev.EvaluatedAt
			rc.EvaluatedBy = prev.EvaluatedBy
			rc.EvaluatedByName = prev.EvaluatedByName
		}
		st.Companies[key] = rc
		err := st.saveLocked()
		st.mu.Unlock()
		if err != nil {
			apiErr(w, 500, "erro ao persistir empresa")
			return
		}
		writeJSON(w, 200, rc)
	})
	mux.HandleFunc("/api/v1/mentor/companies/evaluate", func(w http.ResponseWriter, r *http.Request) {
		u, ok := st.require(w, r, "tutor", "mentor")
		if !ok {
			return
		}
		if r.Method != "POST" {
			apiErr(w, 405, "método não permitido")
			return
		}
		var in struct {
			CompanyID string `json:"company_id"`
			Status    string `json:"status"`
			Comment   string `json:"comment"`
		}
		if readJSON(r, &in) != nil {
			apiErr(w, 400, "JSON inválido")
			return
		}
		in.CompanyID = strings.TrimSpace(in.CompanyID)
		in.Status = strings.ToLower(strings.TrimSpace(in.Status))
		in.Comment = strings.TrimSpace(in.Comment)
		if in.CompanyID == "" {
			apiErr(w, 400, "empresa não informada")
			return
		}
		if in.Status != "aprovado" && in.Status != "reprovado" {
			apiErr(w, 400, "classificação deve ser APROVADO ou REPROVADO")
			return
		}
		if len([]rune(in.Comment)) > 4000 {
			apiErr(w, 400, "comentário deve ter no máximo 4000 caracteres")
			return
		}
		st.mu.Lock()
		company, exists := st.Companies[in.CompanyID]
		if !exists {
			st.mu.Unlock()
			apiErr(w, 404, "empresa não encontrada")
			return
		}
		cl, classExists := st.Classes[company.ClassID]
		if !classExists || cl.TutorID != u.ID {
			st.mu.Unlock()
			apiErr(w, 403, "esta empresa não pertence a uma turma deste Mentor")
			return
		}
		now := time.Now().UTC().Format(time.RFC3339)
		company.ApprovalStatus = in.Status
		company.MentorComment = in.Comment
		company.EvaluatedAt = now
		company.EvaluatedBy = u.ID
		company.EvaluatedByName = u.Name
		company.Revision++
		company.UpdatedAt = now
		st.Companies[in.CompanyID] = company
		err := st.saveLocked()
		st.mu.Unlock()
		if err != nil {
			apiErr(w, 500, "erro ao salvar avaliação")
			return
		}
		writeJSON(w, 200, company)
	})
	mux.HandleFunc("/api/v1/classes/", func(w http.ResponseWriter, r *http.Request) {
		u, ok := st.require(w, r, "tutor", "mentor")
		if !ok {
			return
		}
		path := strings.TrimPrefix(r.URL.Path, "/api/v1/classes/")
		if strings.HasSuffix(path, "/companies") {
			classID := strings.TrimSuffix(path, "/companies")
			st.mu.RLock()
			cl, exists := st.Classes[classID]
			if !exists || cl.TutorID != u.ID {
				st.mu.RUnlock()
				apiErr(w, 404, "turma não encontrada")
				return
			}
			out := []RemoteCompany{}
			for _, c := range st.Companies {
				if c.ClassID == classID {
					out = append(out, c)
				}
			}
			st.mu.RUnlock()
			writeJSON(w, 200, out)
			return
		}
		if strings.HasSuffix(path, "/scenario") {
			if r.Method != "POST" {
				apiErr(w, 405, "método não permitido")
				return
			}
			classID := strings.TrimSuffix(path, "/scenario")
			var scenario Cenario
			if readJSON(r, &scenario) != nil || strings.TrimSpace(scenario.Nome) == "" {
				apiErr(w, 400, "cenário inválido")
				return
			}
			st.mu.Lock()
			cl, exists := st.Classes[classID]
			if !exists || cl.TutorID != u.ID {
				st.mu.Unlock()
				apiErr(w, 404, "turma não encontrada")
				return
			}
			cl.Scenario = scenario
			st.Classes[classID] = cl
			err := st.saveLocked()
			st.mu.Unlock()
			if err != nil {
				apiErr(w, 500, "erro ao atualizar cenário da turma")
				return
			}
			writeJSON(w, 200, cl)
			return
		}
		apiErr(w, 404, "rota não encontrada")
	})
	return securityMiddleware(mux)
}

func securityMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Frame-Options", "DENY")
		w.Header().Set("Referrer-Policy", "no-referrer")
		w.Header().Set("Permissions-Policy", "camera=(), microphone=(), geolocation=()")
		next.ServeHTTP(w, r)
	})
}

func readLocalLine(prompt string) (string, error) {
	fmt.Print(prompt)
	buf := make([]byte, 0, 128)
	one := make([]byte, 1)
	for {
		n, err := os.Stdin.Read(one)
		if n > 0 {
			if one[0] == '\n' {
				break
			}
			if one[0] != '\r' {
				buf = append(buf, one[0])
			}
		}
		if err != nil {
			if len(buf) > 0 {
				break
			}
			return "", err
		}
	}
	return strings.TrimSpace(string(buf)), nil
}

func createInitialAdminInteractive(st *serverState) error {
	if st.hasAdmin() {
		return errors.New("já existe um administrador; novos administradores devem ser criados pelo painel administrativo")
	}
	fmt.Println("CONFIGURAÇÃO INICIAL DO JED SERVIDOR")
	fmt.Println("O primeiro administrador será o Administrador Principal.")
	fmt.Println()
	name, err := readLocalLine("Nome do administrador: ")
	if err != nil {
		return err
	}
	email, err := readLocalLine("E-mail do administrador: ")
	if err != nil {
		return err
	}
	password, err := readSecretConsole("Senha: ")
	if err != nil {
		return err
	}
	confirm, err := readSecretConsole("Repita a senha: ")
	if err != nil {
		return err
	}
	if password != confirm {
		return errors.New("as senhas não coincidem")
	}
	u, err := st.createInitialAdmin(name, email, password)
	if err != nil {
		return err
	}
	fmt.Println()
	fmt.Printf("Administrador Principal criado: %s <%s> [%s]\n", u.Name, u.Email, u.ID)
	return nil
}

func needsFirstRunSetup(st *serverState) bool {
	return !st.hasAdmin()
}

func bootstrapInitialAdminFromEnv(st *serverState) (bool, error) {
	name := strings.TrimSpace(os.Getenv("JED_BOOTSTRAP_ADMIN_NAME"))
	email := strings.TrimSpace(os.Getenv("JED_BOOTSTRAP_ADMIN_EMAIL"))
	password := os.Getenv("JED_BOOTSTRAP_ADMIN_PASSWORD")

	if name == "" && email == "" && password == "" {
		return false, nil
	}
	if name == "" || email == "" || password == "" {
		return false, errors.New("JED_BOOTSTRAP_ADMIN_NAME, JED_BOOTSTRAP_ADMIN_EMAIL e JED_BOOTSTRAP_ADMIN_PASSWORD devem ser informados juntos")
	}
	u, err := st.createInitialAdmin(name, email, password)
	if err != nil {
		return false, fmt.Errorf("falha ao criar administrador inicial via ambiente: %w", err)
	}
	log.Printf("Administrador Principal inicial criado via ambiente: %s <%s> [%s]", u.Name, u.Email, u.ID)
	return true, nil
}

func firstRunSetupInteractive(st *serverState) (bool, error) {
	if !needsFirstRunSetup(st) {
		return true, nil
	}

	fmt.Println("============================================================")
	fmt.Println(" JED SERVIDOR — PRIMEIRA CONFIGURAÇÃO")
	fmt.Println("============================================================")
	fmt.Println()
	fmt.Println("Nenhum Administrador foi configurado neste servidor.")
	fmt.Println("Antes do primeiro uso, é necessário criar o")
	fmt.Println("Administrador Principal.")
	fmt.Println()
	fmt.Println("1. Criar Administrador Principal agora")
	fmt.Println("2. Sair sem alterar nada")
	fmt.Println()

	for {
		choice, err := readLocalLine("Escolha [1/2]: ")
		if err != nil {
			return false, err
		}
		switch strings.TrimSpace(choice) {
		case "1":
			for {
				fmt.Println()
				if err := createInitialAdminInteractive(st); err == nil {
					fmt.Println()
					fmt.Println("Configuração inicial concluída.")
					fmt.Println("O JED Servidor será iniciado agora.")
					fmt.Println()
					return true, nil
				} else {
					fmt.Println()
					fmt.Println("Não foi possível criar o Administrador:")
					fmt.Println(err)
					fmt.Println()
					retry, readErr := readLocalLine("Tentar novamente? [S/n]: ")
					if readErr != nil {
						return false, readErr
					}
					retry = strings.ToLower(strings.TrimSpace(retry))
					if retry == "n" || retry == "nao" || retry == "não" {
						fmt.Println("Configuração cancelada. Nenhuma conta administrativa foi criada.")
						return false, nil
					}
				}
			}
		case "2":
			fmt.Println("Configuração cancelada. O servidor não foi iniciado.")
			return false, nil
		default:
			fmt.Println("Opção inválida. Digite 1 para configurar ou 2 para sair.")
		}
	}
}

func resetAdminPasswordInteractive(st *serverState) error {
	if !st.hasAdmin() {
		return errors.New("não existe administrador para recuperar")
	}
	fmt.Println("RECUPERAÇÃO LOCAL DE ADMINISTRADOR")
	email, err := readLocalLine("E-mail do administrador: ")
	if err != nil {
		return err
	}
	password, err := readSecretConsole("Nova senha: ")
	if err != nil {
		return err
	}
	confirm, err := readSecretConsole("Repita a nova senha: ")
	if err != nil {
		return err
	}
	if password != confirm {
		return errors.New("as senhas não coincidem")
	}
	if err := st.resetAdminPasswordLocal(email, password); err != nil {
		return err
	}
	fmt.Println("Senha administrativa redefinida. Sessões anteriores dessa conta foram encerradas.")
	return nil
}

func runServerCLI(args []string) error {
	root := os.Getenv("JED_SERVER_DATA")
	if root == "" {
		root = filepath.Join(".", "servidor_dados")
	}
	st := newServerState(root)
	if err := st.load(); err != nil {
		return err
	}

	if len(args) > 0 && args[0] == "--configure-email" {
		if len(args) != 1 {
			return errors.New("uso: JED_Servidor.exe --configure-email")
		}
		return configureEmailInteractive()
	}
	if len(args) > 0 && args[0] == "--test-email" {
		if len(args) != 1 {
			return errors.New("uso: JED_Servidor.exe --test-email")
		}
		return testEmailInteractive()
	}

	if len(args) > 0 && args[0] == "--create-admin" {
		if len(args) != 1 {
			return errors.New("uso: JED_Servidor.exe --create-admin")
		}
		return createInitialAdminInteractive(st)
	}
	if len(args) > 0 && args[0] == "--reset-admin-password" {
		if len(args) != 1 {
			return errors.New("uso: JED_Servidor.exe --reset-admin-password")
		}
		return resetAdminPasswordInteractive(st)
	}

	if len(args) > 0 && (args[0] == "--create-mentor" || args[0] == "--create-tutor") {
		return errors.New("criação direta de mentor foi desativada; crie o primeiro administrador com --create-admin e credencie mentores pelo painel")
	}
	if len(args) > 0 && args[0] == "--create-student" {
		if len(args) < 4 {
			return errors.New("uso: JED_Servidor.exe --create-student NOME EMAIL SENHA")
		}
		u, err := st.createUser(args[1], args[2], args[3], "aluno")
		if err != nil {
			return err
		}
		fmt.Printf("Aluno criado: %s <%s> [%s]\n", u.Name, u.Email, u.ID)
		return nil
	}
	if !st.hasAdmin() {
		bootstrapped, err := bootstrapInitialAdminFromEnv(st)
		if err != nil {
			return err
		}
		if !bootstrapped {
			start, err := firstRunSetupInteractive(st)
			if err != nil {
				return err
			}
			if !start {
				return nil
			}
		}
	}
	if cfg, configured, cfgErr := loadServerEmailConfig(); cfgErr != nil {
		log.Printf("AVISO: configuração de e-mail inválida: %v", cfgErr)
	} else if configured {
		log.Printf("E-mail automático: configurado via SMTP %s:%d (%s)", cfg.Host, cfg.Port, cfg.Security)
	} else {
		log.Printf("E-mail automático: não configurado. Contas poderão ser criadas, mas os avisos de cadastro não serão enviados até configurar o SMTP.")
		log.Printf("Para configurar: JED_Servidor.exe --configure-email")
	}

	addr := strings.TrimSpace(os.Getenv("JED_SERVER_ADDR"))
	if addr == "" {
		if port := strings.TrimSpace(os.Getenv("PORT")); port != "" {
			addr = ":" + port
		} else {
			addr = ":8787"
		}
	}
	srv := &http.Server{
		Addr:              addr,
		Handler:           st.handler(),
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       60 * time.Second,
		MaxHeaderBytes:    1 << 20,
	}

	cert, key := os.Getenv("JED_TLS_CERT"), os.Getenv("JED_TLS_KEY")
	host, port, _ := net.SplitHostPort(addr)
	if host == "" {
		host = "0.0.0.0"
	}
	log.Printf("JED Servidor %s iniciado em %s:%s", version, host, port)
	if cert != "" && key != "" {
		log.Printf("HTTPS ativo")
		return srv.ListenAndServeTLS(cert, key)
	}
	log.Printf("HTTP ativo. Para Internet pública, use HTTPS (JED_TLS_CERT/JED_TLS_KEY) ou um proxy reverso TLS.")
	err := srv.ListenAndServe()
	if err == http.ErrServerClosed {
		return nil
	}
	return err
}

func serverCountUsers(st *serverState) int {
	st.mu.RLock()
	defer st.mu.RUnlock()
	return len(st.Users)
}

func serverPortFromEnv() int {
	p, _ := strconv.Atoi(os.Getenv("JED_PORT"))
	if p <= 0 {
		p = 8787
	}
	return p
}
