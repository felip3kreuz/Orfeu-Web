package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestInitialAdminAndPrimaryTransfer(t *testing.T) {
	st := newServerState(t.TempDir())
	first, err := st.createInitialAdmin("Admin Um", "admin1@example.com", "SenhaAdmin123")
	if err != nil {
		t.Fatal(err)
	}
	if first.Role != "admin" || !first.IsPrimaryAdmin || !first.CanInviteMentors {
		t.Fatalf("admin inicial inválido: %#v", first)
	}
	if _, err := st.createInitialAdmin("Admin Dois", "admin2@example.com", "SenhaAdmin123"); err == nil {
		t.Fatal("segundo admin inicial deveria ser rejeitado")
	}
	second, err := st.createAdmin(first, "Admin Dois", "admin2@example.com", "SenhaAdmin456")
	if err != nil {
		t.Fatal(err)
	}
	if second.IsPrimaryAdmin {
		t.Fatal("segundo admin não deveria nascer principal")
	}
	target, err := st.transferPrimaryAdmin(first, second.ID)
	if err != nil {
		t.Fatal(err)
	}
	if !target.IsPrimaryAdmin {
		t.Fatal("transferência não marcou o novo principal")
	}
	if st.Users[first.ID].IsPrimaryAdmin {
		t.Fatal("admin anterior permaneceu principal")
	}
}

func TestMentorDelegationRequiresAdminPermission(t *testing.T) {
	st := newServerState(t.TempDir())
	admin, err := st.createInitialAdmin("Admin", "admin@example.com", "SenhaAdmin123")
	if err != nil {
		t.Fatal(err)
	}
	cred, err := st.createMentorInvitation(admin, "Mentor", "mentor@example.com", "Escola", "")
	if err != nil {
		t.Fatal(err)
	}
	login, err := st.redeemMentorInvitation(cred.Code, "Mentor", "mentor@example.com", "SenhaMentor123", "Escola", "")
	if err != nil {
		t.Fatal(err)
	}
	mentor := login.User
	if mentor.CanInviteMentors {
		t.Fatal("mentor novo não deveria poder credenciar outros mentores")
	}
	if _, err := st.createMentorInvitation(mentor, "Outro", "outro@example.com", "", ""); err == nil {
		t.Fatal("mentor sem permissão conseguiu emitir credencial")
	}
	mentor, err = st.setMentorPermission(admin, mentor.ID, true)
	if err != nil {
		t.Fatal(err)
	}
	if !mentor.CanInviteMentors {
		t.Fatal("permissão não foi concedida")
	}
	if _, err := st.createMentorInvitation(mentor, "Outro", "outro@example.com", "", ""); err != nil {
		t.Fatalf("mentor autorizado deveria emitir credencial: %v", err)
	}
}

func TestDisabledUserCannotAuthenticate(t *testing.T) {
	st := newServerState(t.TempDir())
	admin, err := st.createInitialAdmin("Admin", "admin@example.com", "SenhaAdmin123")
	if err != nil {
		t.Fatal(err)
	}
	student, err := st.createUser("Aluno", "aluno@example.com", "SenhaAluno123", "aluno")
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := st.authenticate(student.Email, "SenhaAluno123"); !ok {
		t.Fatal("aluno ativo deveria autenticar")
	}
	if _, err := st.setUserStatus(admin, student.ID, "disabled"); err != nil {
		t.Fatal(err)
	}
	if _, ok := st.authenticate(student.Email, "SenhaAluno123"); ok {
		t.Fatal("aluno desativado não deveria autenticar")
	}
}

func TestAdminAPI(t *testing.T) {
	st := newServerState(t.TempDir())
	admin, err := st.createInitialAdmin("Admin", "admin@example.com", "SenhaAdmin123")
	if err != nil {
		t.Fatal(err)
	}
	token := st.newSession(admin.ID)
	ts := httptest.NewServer(st.handler())
	defer ts.Close()

	call := func(method, path string, body any, out any) int {
		t.Helper()
		var buf bytes.Buffer
		if body != nil {
			if err := json.NewEncoder(&buf).Encode(body); err != nil {
				t.Fatal(err)
			}
		}
		req, err := http.NewRequest(method, ts.URL+path, &buf)
		if err != nil {
			t.Fatal(err)
		}
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

	var created OnlineUser
	if code := call("POST", "/api/v1/admin/create-admin", map[string]string{
		"name": "Admin Dois", "email": "admin2@example.com", "password": "SenhaAdmin456",
	}, &created); code != http.StatusCreated {
		t.Fatalf("create-admin HTTP %d", code)
	}
	if created.Role != "admin" {
		t.Fatalf("papel inesperado: %q", created.Role)
	}

	var users []OnlineUser
	if code := call("GET", "/api/v1/admin/users", nil, &users); code != http.StatusOK {
		t.Fatalf("users HTTP %d", code)
	}
	if len(users) != 2 {
		t.Fatalf("esperava 2 usuários, recebeu %d", len(users))
	}
}

func TestPrimaryAdminDeletionAndClassDetach(t *testing.T) {
	st := newServerState(t.TempDir())
	primary, err := st.createInitialAdmin("Admin Principal", "principal@example.com", "SenhaAdmin123")
	if err != nil {
		t.Fatal(err)
	}
	secondary, err := st.createAdmin(primary, "Admin Dois", "admin2@example.com", "SenhaAdmin456")
	if err != nil {
		t.Fatal(err)
	}
	mentor, err := st.createUser("Mentor", "mentor-delete@example.com", "SenhaMentor123", "mentor")
	if err != nil {
		t.Fatal(err)
	}
	student, err := st.createUser("Aluno", "aluno-delete@example.com", "SenhaAluno123", "aluno")
	if err != nil {
		t.Fatal(err)
	}
	cl := OnlineClass{ID: "tur-delete", Name: "Turma Exclusão", TutorID: mentor.ID, JoinCode: "DEL123", StudentIDs: []string{student.ID}, Scenario: cenariosBase[0]}
	st.Classes[cl.ID] = cl
	st.Companies[student.ID+":empresa"] = RemoteCompany{
		ID: student.ID + ":empresa", OwnerID: student.ID, ClassID: cl.ID,
		Company: Empresa{LocalID: "empresa", Nome: "Empresa Teste", TurmaID: cl.ID},
	}

	if _, err := st.deleteUserAsPrimary(secondary, student.ID); err == nil {
		t.Fatal("administrador secundário não deveria excluir usuários")
	}
	if _, err := st.deleteUserAsPrimary(primary, mentor.ID); err == nil {
		t.Fatal("Mentor com turma deveria exigir exclusão da turma antes")
	}

	deletedClass, detached, err := st.deleteClassAsPrimary(primary, cl.ID)
	if err != nil {
		t.Fatal(err)
	}
	if deletedClass.ID != cl.ID || detached != 1 {
		t.Fatalf("exclusão de turma inesperada: class=%#v detached=%d", deletedClass, detached)
	}
	remote := st.Companies[student.ID+":empresa"]
	if remote.ClassID != "" || remote.Company.TurmaID != "" {
		t.Fatalf("empresa deveria ser preservada e desvinculada: %#v", remote)
	}
	if _, err := st.deleteUserAsPrimary(primary, mentor.ID); err != nil {
		t.Fatalf("Mentor sem turmas deveria poder ser excluído: %v", err)
	}
	if _, ok := st.Users[mentor.ID]; ok {
		t.Fatal("Mentor excluído permaneceu na base")
	}

	// Recria um vínculo para confirmar a limpeza integral ao excluir Aluno.
	cl2 := OnlineClass{ID: "tur-student", Name: "Turma Aluno", TutorID: primary.ID, JoinCode: "STU123", StudentIDs: []string{student.ID}, Scenario: cenariosBase[0]}
	st.Classes[cl2.ID] = cl2
	remote.ClassID = cl2.ID
	remote.Company.TurmaID = cl2.ID
	st.Companies[remote.ID] = remote
	if _, err := st.deleteUserAsPrimary(primary, student.ID); err != nil {
		t.Fatal(err)
	}
	if _, ok := st.Users[student.ID]; ok {
		t.Fatal("Aluno excluído permaneceu na base")
	}
	if _, ok := st.Companies[remote.ID]; ok {
		t.Fatal("empresa do Aluno excluído permaneceu na base")
	}
	for _, id := range st.Classes[cl2.ID].StudentIDs {
		if id == student.ID {
			t.Fatal("Aluno excluído permaneceu vinculado à turma")
		}
	}
	if _, err := st.deleteUserAsPrimary(primary, primary.ID); err == nil {
		t.Fatal("Administrador Principal não deveria excluir a própria conta")
	}
}

func TestAdminCanListAllClasses(t *testing.T) {
	st := newServerState(t.TempDir())
	admin, err := st.createInitialAdmin("Admin", "admin-classes@example.com", "SenhaAdmin123")
	if err != nil {
		t.Fatal(err)
	}
	mentor, err := st.createUser("Mentor", "mentor-classes@example.com", "SenhaMentor123", "mentor")
	if err != nil {
		t.Fatal(err)
	}
	st.Classes["tur-admin-list"] = OnlineClass{ID: "tur-admin-list", Name: "Turma Visível", TutorID: mentor.ID, JoinCode: "ABC999", Scenario: cenariosBase[0]}
	token := st.newSession(admin.ID)
	ts := httptest.NewServer(st.handler())
	defer ts.Close()

	req, err := http.NewRequest(http.MethodGet, ts.URL+"/api/v1/classes", nil)
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Authorization", "Bearer "+token)
	resp, err := ts.Client().Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("classes do admin HTTP %d", resp.StatusCode)
	}
	var classes []OnlineClass
	if err := json.NewDecoder(resp.Body).Decode(&classes); err != nil {
		t.Fatal(err)
	}
	if len(classes) != 1 || classes[0].ID != "tur-admin-list" {
		t.Fatalf("Administrador deveria listar todas as turmas: %#v", classes)
	}
}

func TestPrimaryAdminDeleteAPI(t *testing.T) {
	st := newServerState(t.TempDir())
	admin, err := st.createInitialAdmin("Admin", "admin-delete-api@example.com", "SenhaAdmin123")
	if err != nil {
		t.Fatal(err)
	}
	student, err := st.createUser("Aluno API", "aluno-api@example.com", "SenhaAluno123", "aluno")
	if err != nil {
		t.Fatal(err)
	}
	mentor, err := st.createUser("Mentor API", "mentor-api@example.com", "SenhaMentor123", "mentor")
	if err != nil {
		t.Fatal(err)
	}
	cl := OnlineClass{ID: "tur-api-delete", Name: "Turma API", TutorID: mentor.ID, JoinCode: "API123", StudentIDs: []string{student.ID}, Scenario: cenariosBase[0]}
	st.Classes[cl.ID] = cl
	token := st.newSession(admin.ID)
	ts := httptest.NewServer(st.handler())
	defer ts.Close()

	post := func(path string, body any) int {
		t.Helper()
		var buf bytes.Buffer
		if err := json.NewEncoder(&buf).Encode(body); err != nil {
			t.Fatal(err)
		}
		req, err := http.NewRequest(http.MethodPost, ts.URL+path, &buf)
		if err != nil {
			t.Fatal(err)
		}
		req.Header.Set("Authorization", "Bearer "+token)
		req.Header.Set("Content-Type", "application/json")
		resp, err := ts.Client().Do(req)
		if err != nil {
			t.Fatal(err)
		}
		defer resp.Body.Close()
		return resp.StatusCode
	}

	if code := post("/api/v1/admin/classes/delete", map[string]string{"class_id": cl.ID}); code != http.StatusOK {
		t.Fatalf("delete class HTTP %d", code)
	}
	if code := post("/api/v1/admin/users/delete", map[string]string{"user_id": student.ID}); code != http.StatusOK {
		t.Fatalf("delete user HTTP %d", code)
	}
	if _, ok := st.Users[student.ID]; ok {
		t.Fatal("API não removeu o Aluno")
	}
}
