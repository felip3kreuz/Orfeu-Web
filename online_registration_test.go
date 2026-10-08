package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestMentorCredentialFlow(t *testing.T) {
	st := newServerState(t.TempDir())
	first, err := st.createInitialAdmin("Admin Inicial", "admin1@example.com", "SenhaSegura123")
	if err != nil {
		t.Fatal(err)
	}
	inv, err := st.createMentorInvitation(first, "Nova Mentora", "mentor2@example.com", "Escola JED", "M-002")
	if err != nil {
		t.Fatal(err)
	}
	if inv.Code == "" || inv.Code[:4] != "MTR-" {
		t.Fatalf("código inesperado: %q", inv.Code)
	}
	out, err := st.redeemMentorInvitation(inv.Code, "Nova Mentora", "mentor2@example.com", "OutraSenha123", "Outra Escola", "")
	if err != nil {
		t.Fatal(err)
	}
	if out.User.Role != "mentor" {
		t.Fatalf("papel esperado mentor; recebido %q", out.User.Role)
	}
	if out.User.Institution != "Escola JED" {
		t.Fatalf("instituição da credencial não preservada: %q", out.User.Institution)
	}
	if _, err := st.redeemMentorInvitation(inv.Code, "Outra", "mentor2@example.com", "OutraSenha123", "", ""); err == nil {
		t.Fatal("credencial reutilizada deveria falhar")
	}
}

func TestMentorCredentialRejectsWrongEmail(t *testing.T) {
	st := newServerState(t.TempDir())
	first, err := st.createInitialAdmin("Admin Inicial", "admin1@example.com", "SenhaSegura123")
	if err != nil {
		t.Fatal(err)
	}
	inv, err := st.createMentorInvitation(first, "Nova Mentora", "mentor2@example.com", "", "")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := st.redeemMentorInvitation(inv.Code, "Intruso", "outro@example.com", "OutraSenha123", "", ""); err == nil {
		t.Fatal("e-mail diferente deveria ser rejeitado")
	}
}

func TestStudentSelfRegistrationProfile(t *testing.T) {
	st := newServerState(t.TempDir())
	u, err := st.createUserProfile("Aluno Teste", "aluno@example.com", "SenhaSegura123", "aluno", "", "A-123")
	if err != nil {
		t.Fatal(err)
	}
	if u.Role != "aluno" || u.InstitutionalID != "A-123" {
		t.Fatalf("perfil inesperado: %#v", u)
	}
}

func postJSON(t *testing.T, client *http.Client, url string, body any, out any) int {
	t.Helper()
	b, err := json.Marshal(body)
	if err != nil {
		t.Fatal(err)
	}
	resp, err := client.Post(url, "application/json", bytes.NewReader(b))
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

func TestPublicRegistrationIsDisabledAndAdminProvisioningRequiresPasswordChange(t *testing.T) {
	st := newServerState(t.TempDir())
	admin, err := st.createInitialAdmin("Admin Inicial", "admin1@example.com", "SenhaSegura123")
	if err != nil {
		t.Fatal(err)
	}
	ts := httptest.NewServer(st.handler())
	defer ts.Close()

	var apiErrBody APIError
	status := postJSON(t, ts.Client(), ts.URL+"/api/v1/register/student", map[string]string{
		"name": "Aluno Um", "email": "aluno1@example.com", "password": "SenhaAluno123",
	}, &apiErrBody)
	if status != http.StatusForbidden {
		t.Fatalf("autocadastro de aluno deveria retornar 403; recebeu %d", status)
	}

	token := st.newSession(admin.ID)
	body, _ := json.Marshal(map[string]string{
		"name": "Aluno Um", "email": "aluno1@example.com", "role": "aluno", "institutional_id": "A-001",
	})
	req, _ := http.NewRequest(http.MethodPost, ts.URL+"/api/v1/admin/users/create", bytes.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	resp, err := ts.Client().Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("cadastro administrativo retornou HTTP %d", resp.StatusCode)
	}
	var created OnlineUser
	if err := json.NewDecoder(resp.Body).Decode(&created); err != nil {
		t.Fatal(err)
	}
	if created.Role != "aluno" || !created.MustChangePassword || created.InstitutionalID != "A-001" {
		t.Fatalf("usuário provisionado inválido: %#v", created)
	}
	if _, ok := st.authenticate(created.Email, defaultProvisionedPassword); !ok {
		t.Fatal("senha temporária padrão deveria autenticar")
	}
}

func TestTemporaryPasswordMustBeChangedBeforeProtectedAPI(t *testing.T) {
	st := newServerState(t.TempDir())
	admin, err := st.createInitialAdmin("Admin", "admin@example.com", "SenhaAdmin123")
	if err != nil {
		t.Fatal(err)
	}
	student, err := st.createProvisionedUser(admin, "Aluno", "aluno@example.com", "aluno", "Escola", "A-1")
	if err != nil {
		t.Fatal(err)
	}
	token := st.newSession(student.ID)
	ts := httptest.NewServer(st.handler())
	defer ts.Close()

	req, _ := http.NewRequest(http.MethodGet, ts.URL+"/api/v1/classes", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	resp, err := ts.Client().Do(req)
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusPreconditionRequired {
		t.Fatalf("API protegida deveria exigir troca de senha (428); recebeu %d", resp.StatusCode)
	}

	body, _ := json.Marshal(map[string]string{"current": defaultProvisionedPassword, "new": "NovaSenha123"})
	req, _ = http.NewRequest(http.MethodPost, ts.URL+"/api/v1/password", bytes.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	resp, err = ts.Client().Do(req)
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("troca de senha HTTP %d", resp.StatusCode)
	}
	if st.Users[student.ID].MustChangePassword {
		t.Fatal("flag de primeiro acesso não foi limpa")
	}
}

func TestStudentInvitationCreatesPasswordAndPreservesInstitutionalID(t *testing.T) {
	st := newServerState(t.TempDir())
	mentor, err := st.createUser("Mentor Teste", "mentor@example.com", "SenhaMentor123", "mentor")
	if err != nil {
		t.Fatal(err)
	}
	class := OnlineClass{ID: "tur-test", Name: "Turma Teste", TutorID: mentor.ID, JoinCode: "ABC123", StudentIDs: []string{}, Scenario: cenariosBase[0]}
	st.Classes[class.ID] = class
	inv, err := st.createInvitation(mentor, class.ID, "Aluno Convidado", "convite@example.com", "MAT-009")
	if err != nil {
		t.Fatal(err)
	}
	out, err := st.redeemInvitation(inv.Code, "SenhaAluno123")
	if err != nil {
		t.Fatal(err)
	}
	if out.User.Role != "aluno" {
		t.Fatalf("papel esperado aluno; recebido %q", out.User.Role)
	}
	if out.Token == "" {
		t.Fatal("ativação do convite deve abrir uma sessão")
	}
	if out.User.InstitutionalID != "MAT-009" {
		t.Fatalf("ID institucional do convite não preservado: %q", out.User.InstitutionalID)
	}
	joined := st.Classes[class.ID]
	found := false
	for _, id := range joined.StudentIDs {
		if id == out.User.ID {
			found = true
			break
		}
	}
	if !found {
		t.Fatal("Aluno ativado pelo convite deveria estar vinculado à turma")
	}
}

func TestMentorScenarioAPI(t *testing.T) {
	st := newServerState(t.TempDir())
	mentor, err := st.createUser("Mentor Cenários", "mentor-cenarios@example.com", "SenhaMentor123", "mentor")
	if err != nil {
		t.Fatal(err)
	}
	token := st.newSession(mentor.ID)
	ts := httptest.NewServer(st.handler())
	defer ts.Close()

	body, _ := json.Marshal(map[string]any{
		"nome":                "Piloto Web",
		"alcance":             1.05,
		"conversao":           1.02,
		"oscilacao":           0.09,
		"duracao":             10,
		"concorrencia_nivel":  "media",
		"concorrencia_indice": 1.0,
		"dificuldade":         "intermediario",
	})
	req, err := http.NewRequest(http.MethodPost, ts.URL+"/api/v1/scenarios", bytes.NewReader(body))
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
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("criação de cenário retornou HTTP %d", resp.StatusCode)
	}
	var created OnlineScenario
	if err := json.NewDecoder(resp.Body).Decode(&created); err != nil {
		t.Fatal(err)
	}
	if created.ID == "" || created.Scenario.Nome != "Piloto Web" || created.Scenario.Duracao != 10 {
		t.Fatalf("cenário criado inválido: %#v", created)
	}

	req, _ = http.NewRequest(http.MethodGet, ts.URL+"/api/v1/scenarios", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	resp, err = ts.Client().Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	var list []OnlineScenario
	if err := json.NewDecoder(resp.Body).Decode(&list); err != nil {
		t.Fatal(err)
	}
	if len(list) < 5 { // 4 cenários-base + o personalizado
		t.Fatalf("lista de cenários incompleta: %d", len(list))
	}
}

func TestMentorEvaluatesOwnCompanyAndStudentSyncPreservesEvaluation(t *testing.T) {
	st := newServerState(t.TempDir())
	mentor, err := st.createUser("Mentor Avaliador", "mentor-avalia@example.com", "SenhaMentor123", "mentor")
	if err != nil {
		t.Fatal(err)
	}
	student, err := st.createUser("Aluno Avaliado", "aluno-avalia@example.com", "SenhaAluno123", "aluno")
	if err != nil {
		t.Fatal(err)
	}
	class := OnlineClass{ID: "tur-avalia", Name: "Turma Avaliação", TutorID: mentor.ID, JoinCode: "AV1234", StudentIDs: []string{student.ID}, Scenario: cenariosBase[0]}
	st.Classes[class.ID] = class
	companyKey := student.ID + ":empresa-1"
	st.Companies[companyKey] = RemoteCompany{ID: companyKey, OwnerID: student.ID, ClassID: class.ID, Revision: 1, Company: Empresa{LocalID: "empresa-1", Nome: "Negócio Teste"}}

	ts := httptest.NewServer(st.handler())
	defer ts.Close()
	mentorToken := st.newSession(mentor.ID)

	body, _ := json.Marshal(map[string]string{
		"company_id": companyKey,
		"status":     "aprovado",
		"comment":    "Aprovado, mas com ressalvas sobre a validação do público-alvo.",
	})
	req, _ := http.NewRequest(http.MethodPost, ts.URL+"/api/v1/mentor/companies/evaluate", bytes.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+mentorToken)
	req.Header.Set("Content-Type", "application/json")
	resp, err := ts.Client().Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("avaliação retornou HTTP %d", resp.StatusCode)
	}
	var evaluated RemoteCompany
	if err := json.NewDecoder(resp.Body).Decode(&evaluated); err != nil {
		t.Fatal(err)
	}
	if evaluated.ApprovalStatus != "aprovado" || evaluated.MentorComment == "" || evaluated.EvaluatedBy != mentor.ID || evaluated.EvaluatedAt == "" {
		t.Fatalf("avaliação inválida: %#v", evaluated)
	}

	studentToken := st.newSession(student.ID)
	studentBody, _ := json.Marshal(map[string]any{
		"class_id": class.ID,
		"company":  Empresa{LocalID: "empresa-1", Nome: "Negócio Teste Atualizado"},
	})
	req, _ = http.NewRequest(http.MethodPut, ts.URL+"/api/v1/companies/empresa-1", bytes.NewReader(studentBody))
	req.Header.Set("Authorization", "Bearer "+studentToken)
	req.Header.Set("Content-Type", "application/json")
	resp2, err := ts.Client().Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp2.Body.Close()
	if resp2.StatusCode != http.StatusOK {
		t.Fatalf("sincronização do aluno retornou HTTP %d", resp2.StatusCode)
	}
	var synced RemoteCompany
	if err := json.NewDecoder(resp2.Body).Decode(&synced); err != nil {
		t.Fatal(err)
	}
	if synced.ApprovalStatus != "aprovado" || synced.MentorComment != evaluated.MentorComment || synced.EvaluatedBy != mentor.ID {
		t.Fatalf("sincronização apagou avaliação: %#v", synced)
	}
}

func TestMentorCannotEvaluateCompanyFromAnotherMentor(t *testing.T) {
	st := newServerState(t.TempDir())
	mentorA, _ := st.createUser("Mentor A", "mentor-a@example.com", "SenhaMentor123", "mentor")
	mentorB, _ := st.createUser("Mentor B", "mentor-b@example.com", "SenhaMentor123", "mentor")
	student, _ := st.createUser("Aluno", "aluno-outro@example.com", "SenhaAluno123", "aluno")
	class := OnlineClass{ID: "tur-a", Name: "Turma A", TutorID: mentorA.ID, JoinCode: "A12345", StudentIDs: []string{student.ID}, Scenario: cenariosBase[0]}
	st.Classes[class.ID] = class
	companyKey := student.ID + ":empresa-2"
	st.Companies[companyKey] = RemoteCompany{ID: companyKey, OwnerID: student.ID, ClassID: class.ID, Revision: 1, Company: Empresa{LocalID: "empresa-2", Nome: "Negócio A"}}

	ts := httptest.NewServer(st.handler())
	defer ts.Close()
	body, _ := json.Marshal(map[string]string{"company_id": companyKey, "status": "reprovado", "comment": "Sem autorização."})
	req, _ := http.NewRequest(http.MethodPost, ts.URL+"/api/v1/mentor/companies/evaluate", bytes.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+st.newSession(mentorB.ID))
	req.Header.Set("Content-Type", "application/json")
	resp, err := ts.Client().Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusForbidden {
		t.Fatalf("Mentor de outra turma deveria receber 403; recebeu %d", resp.StatusCode)
	}
	if got := st.Companies[companyKey].ApprovalStatus; got != "" {
		t.Fatalf("empresa não autorizada foi alterada: %q", got)
	}
}
