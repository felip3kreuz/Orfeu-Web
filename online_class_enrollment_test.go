package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestCentralizedClassEnrollmentIDsAreMonotonic(t *testing.T) {
	st := newServerState(t.TempDir())
	admin, err := st.createInitialAdmin("Admin", "admin@example.com", "SenhaAdmin123")
	if err != nil {
		t.Fatal(err)
	}
	mentor, err := st.createUser("Mentor", "mentor@example.com", "SenhaMentor123", "mentor")
	if err != nil {
		t.Fatal(err)
	}

	st.NextClassNumber = 25
	cl25, err := st.createAdminClass(admin, "Empreendedorismo 2026", mentor.ID)
	if err != nil {
		t.Fatal(err)
	}
	if cl25.Number != 25 {
		t.Fatalf("número esperado 25; recebido %d", cl25.Number)
	}

	var fourth OnlineUser
	for i := 1; i <= 4; i++ {
		u, err := st.createProvisionedUserForClass(admin, "Aluno", "aluno"+string(rune('0'+i))+"@example.com", "aluno", "Escola", "", cl25.ID)
		if err != nil {
			t.Fatal(err)
		}
		expected := "T25A" + string(rune('0'+i))
		if u.EnrollmentID != expected {
			t.Fatalf("matrícula esperada %s; recebida %s", expected, u.EnrollmentID)
		}
		fourth = u
	}
	if fourth.EnrollmentID != "T25A4" {
		t.Fatalf("quarto aluno deveria ser T25A4; recebeu %s", fourth.EnrollmentID)
	}

	if _, err := st.assignStudentClass(admin, fourth.ID, ""); err != nil {
		t.Fatal(err)
	}
	fifth, err := st.createProvisionedUserForClass(admin, "Aluno Cinco", "aluno5@example.com", "aluno", "Escola", "", cl25.ID)
	if err != nil {
		t.Fatal(err)
	}
	if fifth.EnrollmentID != "T25A5" {
		t.Fatalf("número removido não pode ser reutilizado; esperado T25A5, recebeu %s", fifth.EnrollmentID)
	}

	cl26, err := st.createAdminClass(admin, "Empreendedorismo 2026 B", mentor.ID)
	if err != nil {
		t.Fatal(err)
	}
	transferred, err := st.assignStudentClass(admin, fourth.ID, cl26.ID)
	if err != nil {
		t.Fatal(err)
	}
	if transferred.EnrollmentID != "T26A1" {
		t.Fatalf("transferência deveria gerar T26A1; recebeu %s", transferred.EnrollmentID)
	}
	if len(transferred.Enrollments) != 2 || transferred.Enrollments[0].EndedAt == "" {
		t.Fatalf("histórico de matrícula não foi preservado: %#v", transferred.Enrollments)
	}
}

func TestAdminCreatesClassAndStudentAlreadyEnrolled(t *testing.T) {
	st := newServerState(t.TempDir())
	admin, err := st.createInitialAdmin("Admin", "admin@example.com", "SenhaAdmin123")
	if err != nil {
		t.Fatal(err)
	}
	mentor, err := st.createUser("Mentor", "mentor@example.com", "SenhaMentor123", "mentor")
	if err != nil {
		t.Fatal(err)
	}
	ts := httptest.NewServer(st.handler())
	defer ts.Close()
	token := st.newSession(admin.ID)

	post := func(path string, body any, out any) int {
		b, _ := json.Marshal(body)
		req, _ := http.NewRequest(http.MethodPost, ts.URL+path, bytes.NewReader(b))
		req.Header.Set("Authorization", "Bearer "+token)
		req.Header.Set("Content-Type", "application/json")
		resp, err := ts.Client().Do(req)
		if err != nil {
			t.Fatal(err)
		}
		defer resp.Body.Close()
		if out != nil {
			if err := json.NewDecoder(resp.Body).Decode(out); err != nil {
				t.Fatal(err)
			}
		}
		return resp.StatusCode
	}

	var cl OnlineClass
	if code := post("/api/v1/admin/classes", map[string]string{"name": "Turma Central", "mentor_id": mentor.ID}, &cl); code != http.StatusCreated {
		t.Fatalf("criação da turma retornou HTTP %d", code)
	}
	var student OnlineUser
	if code := post("/api/v1/admin/users/create", map[string]string{"name": "Aluno", "email": "aluno@example.com", "role": "aluno", "class_id": cl.ID}, &student); code != http.StatusCreated {
		t.Fatalf("cadastro do aluno retornou HTTP %d", code)
	}
	if student.CurrentClassID != cl.ID || student.EnrollmentID != "T1A1" {
		t.Fatalf("aluno deveria sair do cadastro já matriculado: %#v", student)
	}
}

func TestLegacyJoinAndMentorClassCreationAreDisabled(t *testing.T) {
	st := newServerState(t.TempDir())
	mentor, err := st.createUser("Mentor", "mentor@example.com", "SenhaMentor123", "mentor")
	if err != nil {
		t.Fatal(err)
	}
	student, err := st.createUser("Aluno", "aluno@example.com", "SenhaAluno123", "aluno")
	if err != nil {
		t.Fatal(err)
	}
	ts := httptest.NewServer(st.handler())
	defer ts.Close()

	call := func(token, path string, body any) int {
		b, _ := json.Marshal(body)
		req, _ := http.NewRequest(http.MethodPost, ts.URL+path, bytes.NewReader(b))
		req.Header.Set("Authorization", "Bearer "+token)
		req.Header.Set("Content-Type", "application/json")
		resp, err := ts.Client().Do(req)
		if err != nil {
			t.Fatal(err)
		}
		defer resp.Body.Close()
		return resp.StatusCode
	}
	if code := call(st.newSession(mentor.ID), "/api/v1/classes", map[string]any{"name": "Não pode", "scenario": cenariosBase[0]}); code != http.StatusForbidden {
		t.Fatalf("Mentor não deve criar turma; HTTP %d", code)
	}
	if code := call(st.newSession(student.ID), "/api/v1/classes/join", map[string]string{"code": "ABC123"}); code != http.StatusForbidden {
		t.Fatalf("Aluno não deve entrar por código; HTTP %d", code)
	}
}

func TestDeleteCentralizedClassMovesStudentToWaitingAndClosesEnrollment(t *testing.T) {
	st := newServerState(t.TempDir())
	admin, err := st.createInitialAdmin("Admin", "admin-delete-class@example.com", "SenhaAdmin123")
	if err != nil {
		t.Fatal(err)
	}
	mentor, err := st.createUser("Mentor", "mentor-delete-class@example.com", "SenhaMentor123", "mentor")
	if err != nil {
		t.Fatal(err)
	}
	cl, err := st.createAdminClass(admin, "Turma a excluir", mentor.ID)
	if err != nil {
		t.Fatal(err)
	}
	student, err := st.createProvisionedUserForClass(admin, "Aluno", "aluno-delete-class@example.com", "aluno", "Escola", "", cl.ID)
	if err != nil {
		t.Fatal(err)
	}
	if student.EnrollmentID != "T1A1" {
		t.Fatalf("matrícula inesperada antes da exclusão: %s", student.EnrollmentID)
	}

	if _, _, err := st.deleteClassAsPrimary(admin, cl.ID); err != nil {
		t.Fatal(err)
	}
	rec := st.Users[student.ID]
	if rec.CurrentClassID != "" || rec.EnrollmentID != "" || rec.EnrollmentStatus != "waiting" {
		t.Fatalf("Aluno deveria ficar aguardando turma: %#v", rec.OnlineUser)
	}
	if len(rec.Enrollments) != 1 || rec.Enrollments[0].EndedAt == "" {
		t.Fatalf("histórico da matrícula deveria ser encerrado: %#v", rec.Enrollments)
	}
}
