# W7.1 — Cadastro centralizado, CSV, e-mail e primeiro acesso

## Política de contas

- Não existe mais autocadastro público de Aluno, Mentor ou Administrador.
- Somente Administradores podem cadastrar novas contas.
- O cadastro pode ser individual ou em lote por CSV.
- Mentores não podem cadastrar/ativar Alunos nem emitir credenciais de Mentor.
- Códigos antigos de convite/credenciamento permanecem no arquivo de dados apenas por compatibilidade histórica, mas os endpoints de ativação pública foram bloqueados.

## CSV

Cabeçalhos aceitos:

`nome,email,papel,instituicao,id_institucional`

`papel` aceita: `aluno`, `mentor`, `admin` (ou `administrador`). Vírgula e ponto e vírgula são aceitos pela interface Web.

## Senha temporária

Todas as contas criadas pelo painel usam inicialmente:

`acbd1234`

O servidor marca a conta com `must_change_password=true`. Após o login, somente `/api/v1/me`, `/api/v1/password` e logout permanecem utilizáveis até a senha ser substituída. A Web redireciona automaticamente para `/alterar-senha`.

## E-mail

Após criar a conta, o JED Servidor tenta enviar um aviso para o e-mail cadastrado com perfil, e-mail de acesso, senha temporária e instrução de troca obrigatória.

O status de entrega (`sent`, `failed`, `not_configured`) é persistido no usuário e aparece no painel administrativo.

Em Docker, quando `JED_SERVER_DATA=/data`, a configuração SMTP padrão fica em `/data/servidor_email.json`, portanto sobrevive à recriação do container.

Variáveis SMTP continuam suportadas: `JED_SMTP_HOST`, `JED_SMTP_PORT`, `JED_SMTP_USER`, `JED_SMTP_PASSWORD`, `JED_SMTP_FROM`, `JED_SMTP_FROM_NAME`, `JED_SMTP_SECURITY`. `JED_PUBLIC_URL` define o endereço incluído no e-mail de cadastro.
