# JED Simulador

Simulador educacional de empreendedorismo para apoiar atividades do programa **Jovens Empreendedores Digitais (JED)**.

O software permite que estudantes criem e administrem empreendimentos simulados, trabalhem **Persona** e **Lean Canvas**, escolham canais e ferramentas digitais e acompanhem indicadores comerciais, operacionais e financeiros.

Tutores podem organizar turmas, utilizar cenários pedagógicos, convidar alunos e acompanhar resultados sincronizados quando o modo online é utilizado.

## Estado do projeto

Versão-base deste repositório: **v2.0.0-rc1.8**.

O projeto é distribuído como software livre sob a licença **GNU General Public License v3.0 (GPLv3)**.

## Arquitetura de contas

O modo online utiliza três papéis:

- **Administrador** — administra contas, permissões e segurança da plataforma;
- **Mentor** — administra suas turmas e acompanha resultados;
- **Aluno** — administra suas próprias simulações.

Não existe cadastro público de Administrador. O primeiro Administrador é definido
localmente no servidor com `JED_Servidor.exe --create-admin`.

Um Mentor novo não pode credenciar outros Mentores por padrão. Essa permissão pode
ser delegada individualmente por um Administrador.

## Principais recursos

- simulação semanal de um empreendimento;
- Persona;
- Lean Canvas;
- canais digitais;
- ferramentas digitais;
- marketing, preço e promoções;
- estoque, insumos e fornecedores;
- caixa, custos, contas a receber e contas a pagar;
- indicadores de desempenho;
- modo Tutor;
- turmas e convites no modo online;
- autocadastro de alunos;
- autocadastro de mentores mediante credencial `MTR` de uso único;
- sincronização com servidor JED;
- execução portátil no Windows;
- suporte a Windows x64 e x86.

## Compilação

O projeto usa **Go 1.23**.

Teste local:

```bash
go test ./...
go run . --self-test
```

Build básico para Windows x64:

```bash
GOOS=windows GOARCH=amd64 CGO_ENABLED=0 go build -trimpath -ldflags="-H windowsgui -s -w" -o JED_Simulador.exe .
```

O workflow do GitHub Actions em `.github/workflows/build-windows.yml` compila automaticamente os executáveis Windows.

## Distribuição

As versões oficiais devem ser publicadas na seção **Releases** do repositório.

Antes da aprovação do SignPath Foundation, as releases serão não assinadas. Depois da aprovação, o workflow preparado em `.github/workflows/signpath-release.yml` poderá solicitar a assinatura dos binários.

## Code signing policy

Leia a política completa em [CODE_SIGNING_POLICY.md](CODE_SIGNING_POLICY.md).

**Free code signing provided by SignPath.io, certificate by SignPath Foundation.**

## Privacidade

Leia [PRIVACY.md](PRIVACY.md).

## Segurança

Para comunicar uma vulnerabilidade, consulte [SECURITY.md](SECURITY.md).

## Como contribuir

Consulte [CONTRIBUTING.md](CONTRIBUTING.md).

## SignPath Foundation

O material necessário para preparar a solicitação está em:

- [docs/SIGNPATH_SETUP.md](docs/SIGNPATH_SETUP.md)
- [docs/SIGNPATH_APPLICATION_TEMPLATE.md](docs/SIGNPATH_APPLICATION_TEMPLATE.md)
- [docs/RELEASE_CHECKLIST.md](docs/RELEASE_CHECKLIST.md)

> Antes de solicitar a assinatura gratuita, substitua no repositório os marcadores `SEU_USUARIO_GITHUB` pelo usuário ou organização que realmente manterá o projeto.


## Primeiro uso do servidor

A partir da v2.0 RC1.8, basta abrir `JED_Servidor.exe`.
Se não existir um Administrador, o próprio servidor inicia o assistente de
configuração do Administrador Principal e, ao concluir, inicia o serviço.
O comando `JED_Servidor.exe --create-admin` continua disponível como alternativa.


## E-mail automático de credenciais

O servidor pode enviar credenciais de Mentor por SMTP. O recurso é opcional e não
depende de um provedor específico. Sem SMTP configurado, o código MTR continua
disponível para envio manual.

A configuração local pode ser criada com:

```text
JED_Servidor.exe --configure-email
```

O arquivo `servidor_email.json`, que pode conter uma senha SMTP, é local e está
excluído do repositório por `.gitignore`.

## Web W6.1 — cadastro e interface Órbita Clean

A W6.1 adiciona cadastro/ativação Web completos para Aluno e Mentor, preserva ativação de convite individual e reformula Login, Administrador, Mentor e Aluno para seguir a identidade visual da interface Windows. Consulte `docs/W6_1_ONBOARDING_UI.md`.


## W6.2 — Fluxo do Aluno sem empresa

Corrige navegação e feedback de vínculo de turma para contas de Aluno que ainda não possuem empresa sincronizada. Consulte `docs/W6_2_STUDENT_FLOW.md`.

## Web W6.3 — navegação e favicon

- consolida as correções do fluxo do Aluno introduzidas na W6.2;
- corrige a navegação lateral de Administrador, Mentor e Aluno com rolagem explícita e item ativo;
- corrige VISÃO GERAL, TURMAS e ALUNOS no painel Mentor;
- usa no site o mesmo `jed_icon.ico` do executável Windows como favicon.

## Web W7.0 — paridade funcional

A W7.0 substitui o portal parcial W6.x por uma versão Web funcional do simulador: criação de empresa pelo catálogo Setor → Tipo → Especialidade, Persona, Lean Canvas, canais/ferramentas, decisões e processamento via Go/WASM, insumos, financeiro, indicadores, Jornada JED, turmas e sincronização. O Mentor também passa a criar cenários persistentes, associá-los a turmas, importar alunos por CSV e acompanhar resultados. Consulte `docs/W7_0_WEB_PARITY.md`.

## Web W7.1 — Contas gerenciadas pelo Administrador

A W7.1 centraliza a criação de contas no painel de Administração. Alunos, Mentores e novos Administradores podem ser cadastrados individualmente ou importados por CSV. Todas as novas contas recebem a senha temporária `acbd1234`, com troca obrigatória no primeiro acesso. Quando o SMTP está configurado, o servidor envia automaticamente um aviso de cadastro ao e-mail informado.

Modelo: `MODELO_IMPORTACAO_USUARIOS.csv`.

### W7.2 — avaliação dos empreendimentos

Mentores podem aprovar ou reprovar os empreendimentos vinculados às suas turmas e registrar comentários pedagógicos. A avaliação fica visível ao Aluno e persiste no JED Servidor. A barra lateral do Administrador também foi simplificada para eliminar navegação redundante.

### W7.3 — turmas e matrículas centralizadas

A criação e o nome das turmas passam a ser responsabilidade do Administrador, que também designa o Mentor responsável. O Aluno pode ser vinculado à turma já no cadastro individual ou na importação CSV e recebe uma matrícula automática no padrão `T{turma}A{sequência}` (por exemplo, `T25A4`). O ingresso por código de turma foi desativado. Transferências preservam o histórico e nunca reutilizam números de matrícula. O Mentor continua administrando os cenários pedagógicos das turmas sob sua responsabilidade.
