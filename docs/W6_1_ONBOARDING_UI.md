# JED Simulador — W6.1 / Onboarding + Interface Órbita Clean

A W6.1 corrige lacunas de cadastro da W6.0 e aproxima a interface Web da interface nativa Windows "Órbita Clean".

## Cadastro e ativação

- **Aluno**: cadastro público completo em `/cadastro/aluno`, com nome, e-mail, matrícula/ID opcional, senha e confirmação de senha. A conta é criada pela API RC1.8 e a sessão web é aberta imediatamente em cookie Secure + HttpOnly.
- **Mentor**: ativação completa em `/cadastro/mentor`, com credencial `MTR-XXXX-XXXX`, dados institucionais, senha e confirmação. A senha é definida pelo próprio Mentor; o Administrador apenas emite a credencial.
- **Convite individual de Aluno**: compatibilidade preservada em `/cadastro/convite`; o Aluno informa o código e cria a própria senha.
- **Login**: mantém o BFF Next.js e redireciona automaticamente para `/admin`, `/mentor` ou `/aluno` conforme o papel retornado pelo JED Servidor.

## Interface

A paleta, densidade e hierarquia passam a seguir diretamente `native_windows.go`:

- fundo `#fafafa` e painéis brancos;
- tinta `#10161c`, linhas `#bebebe` e grade técnica `#eeeeee`;
- ciano operacional `#3ed6dc`, magenta de alerta `#ef246d` e âmbar `#f2aa2a`;
- tipografia Segoe UI;
- barra superior escura com `JED`, `BUSINESS SIMULATION / ORBITA CLEAN` e `SYS 2.0 RC1.8`;
- botões/painéis retangulares com barra de acento superior, em vez de cartões SaaS arredondados;
- densidade maior para tabelas, métricas e comandos;
- navegação lateral compacta em desktop e responsiva em telas menores.

## Administração

- formulários de novo Administrador e credenciamento de Mentor foram movidos para janelas modais compactas;
- o painel explica explicitamente que o Administrador não cria a senha do Mentor;
- códigos MTR podem ser copiados diretamente da tabela;
- usuários, estados e permissões continuam usando a mesma API RC1.8.

## Mentor

- criação de turma e convite individual passa a ocorrer em modais;
- códigos de turma e convite podem ser copiados;
- a interface diferencia o fluxo recomendado (Aluno cria conta + entra com código da turma) do convite individual compatível com a versão Windows;
- a senha do Aluno nunca é escolhida pelo Mentor no fluxo de convite.

## Aluno

O painel mantém o JED Core em WebAssembly, processamento semanal, rascunho local, sincronização e entrada em turmas. O desenho foi refeito sem alterar o formato de dados nem o protocolo do servidor.

## Servidor

A W6.1 não exige alteração de protocolo no JED Servidor. Os endpoints de cadastro já existentes na RC1.8 são consumidos pelo BFF web:

- `POST /api/v1/register/student`
- `POST /api/v1/register/mentor`
- `POST /api/v1/invitations/redeem`

A imagem Docker e a configuração de produção introduzidas após a W6.0 foram preservadas no pacote completo.
