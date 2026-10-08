# JED Simulador v2.0 RC1 — Custo Zero

Objetivo: permitir piloto real sem contratar infraestrutura.

- Persistência autocontida em JSON.
- Gravação atômica.
- Backups automáticos locais por 30 dias.
- Limite de tentativas de login.
- Troca de senha.
- Convites individuais, de uso único, expiram em 30 dias.
- Tutor pode revogar convite pendente.
- ALUNO/TUTOR continuam validados no servidor.
- Funciona no mesmo PC ou em rede local sem mensalidade.


## Ajuste visual RC1.1

- aumento do espaçamento vertical dos itens do menu lateral;
- subtítulos do menu lateral encurtados para evitar compressão visual;
- botões de ação com melhor respiro interno e quebra de linha do texto secundário;
- área de status/tendência do painel reorganizada para reduzir sobreposição visual.

## RC1.2 — correção de navegação

- `← VOLTAR` na Visão Geral retorna à tela inicial.
- `← VOLTAR` em Persona, Lean Canvas, Canais, Ferramentas, Insumos,
  Financeiro, Indicadores, Jornada JED e Modo Tutor retorna à Visão Geral.
- `← VOLTAR` no catálogo, lista de arquivos e JED Online retorna à tela inicial.


## RC1.3 — ícone da janela

- inclusão de `jed_icon.ico` no pacote;
- carregamento do ícone da aplicação em tempo de execução para a janela principal e diálogos;
- melhora a exibição do ícone na barra de tarefas e no título da janela enquanto o programa estiver aberto.

Observação: para o ícone do próprio arquivo `.exe` no Explorer, o ideal futuro é embutir o recurso de ícone no binário. Nesta RC1.3, o foco é o ícone do programa aberto.


## RC1.4 — círculos completos na tela inicial

- substituição dos dois semicírculos do painel inicial por dois círculos completos;
- manutenção do restante da identidade visual da tela de abertura;
- preservação do ícone de janela introduzido na RC1.3.


## RC1.5 — autocadastro de Aluno e Mentor

- Alunos podem criar a própria conta diretamente no JED Online.
- Após o cadastro, o aluno entra na turma pelo código fornecido pelo Mentor.
- Mentores também podem iniciar o próprio cadastro, mas a conta só é criada com uma credencial `MTR-XXXX-XXXX` válida.
- Credenciais de Mentor são emitidas por um Mentor já autorizado, têm uso único e validade de 30 dias.
- O e-mail informado no cadastro de Mentor deve coincidir com o e-mail da credencial.
- Uma conta existente não pode ser promovida silenciosamente de Aluno para Mentor.
- Mentores podem listar e revogar credenciais ainda não utilizadas.
- `tutor` continua aceito internamente para compatibilidade com bases antigas; novas contas privilegiadas usam o papel `mentor`.
- O primeiro Mentor pode ser criado no servidor com:
  `JED_Servidor.exe --create-mentor "Nome" email@exemplo.com "SenhaSegura123"`


## RC1.6 — Administrador e cadeia de confiança

### Papéis
- `admin`: administra a plataforma.
- `mentor`: administra suas turmas e alunos.
- `aluno`: administra a própria simulação.
- `tutor` continua aceito como papel legado equivalente a Mentor.

### Primeiro Administrador
O servidor **não inicia** enquanto não existir uma conta `admin`.
A primeira conta é criada localmente na máquina do servidor:

`JED_Servidor.exe --create-admin`

O programa solicita nome, e-mail, senha e confirmação. No Windows, a senha é digitada sem eco no terminal.
Essa conta recebe `is_primary_admin=true`.

### Recuperação local
Se a senha administrativa for perdida, execute **na máquina do servidor**:

`JED_Servidor.exe --reset-admin-password`

A recuperação exige o e-mail de uma conta Administrador e redefine sua senha, invalidando as sessões anteriores.

### Administração
O painel ADMINISTRADOR permite:
- listar usuários;
- criar outros Administradores;
- ativar/desativar contas;
- credenciar Mentores;
- revogar credenciais MTR;
- conceder ou revogar de um Mentor a permissão de credenciar outros Mentores;
- transferir a função de Administrador Principal para outro Administrador ativo.

### Segurança
- Administrador não possui cadastro público.
- Mentor comum nasce com `can_invite_mentors=false`.
- Administrador sempre pode emitir credenciais MTR.
- Um Mentor só pode emitir credenciais MTR se um Administrador conceder essa permissão.
- Uma conta desativada não autentica e suas sessões ativas são invalidadas.
- O Administrador Principal não pode ser desativado antes de transferir a função principal.


## RC1.7 — configuração inicial automática do servidor

A RC1.6 exigia que o responsável soubesse executar `JED_Servidor.exe --create-admin`.
Ao abrir o servidor pela primeira vez com duplo clique, a janela podia encerrar
rapidamente, dando a impressão de que o programa não funcionava.

Na RC1.7:

- ao iniciar `JED_Servidor.exe` sem Administrador, o servidor detecta automaticamente o primeiro uso;
- exibe um assistente textual de configuração;
- oferece `1. Criar Administrador Principal agora` ou `2. Sair sem alterar nada`;
- erros de validação não fecham imediatamente o programa: o usuário pode tentar novamente;
- concluída a criação do Administrador Principal, o servidor inicia automaticamente;
- `--create-admin` continua disponível como alternativa manual;
- servidores já configurados não mudam de comportamento.


## RC1.8 — e-mail automático de credenciais de Mentor

- Credenciais `MTR` podem ser enviadas automaticamente ao e-mail do Mentor.
- O envio utiliza SMTP configurado pelo operador do servidor.
- Não há dependência obrigatória de serviço pago ou provedor específico.
- Sem SMTP, a credencial continua sendo criada para envio manual.
- Falha no SMTP nunca invalida nem duplica o código já persistido.
- Status do envio é exibido ao Administrador/Mentor: enviado, manual ou falhou.
- Novo comando local: `JED_Servidor.exe --configure-email`.
- Novo comando de teste: `JED_Servidor.exe --test-email`.
- Configuração local padrão: `servidor_email.json`.
- O arquivo de configuração SMTP é um segredo local e deve permanecer fora do GitHub.

## W6.1 — Onboarding + Interface Órbita Clean

A camada Web agora expõe os fluxos de cadastro já suportados pelo servidor RC1.8: Aluno, Mentor por credencial MTR e ativação de convite individual. Login/cadastro mantêm o token do servidor exclusivamente em cookie Secure + HttpOnly. A interface foi redesenhada com a mesma paleta, tipografia, grade técnica e hierarquia visual da GUI Win32. Detalhes em `docs/W6_1_ONBOARDING_UI.md`.

## W7.0 — Web Parity

- interface do Aluno deixa de depender da versão Windows para criar a empresa;
- catálogo completo Setor → Tipo → Especialidade fica disponível na Web;
- Persona, Lean Canvas, canais, ferramentas, decisões, insumos, financeiro, indicadores e Jornada entram no painel Web;
- processamento semanal continua no JED Core Go compilado para WebAssembly;
- ponte WASM ganha operações de compras/insumos e utilitários de Jornada/Canvas;
- Mentor ganha cenários persistidos no JED Servidor, turmas com cenário, CSV de alunos e resultados;
- navegação lateral passa a trocar módulos reais em vez de depender de âncoras.

## W7.1 — Cadastro centralizado

- cadastro de Aluno, Mentor e Administrador exclusivo do Administrador;
- cadastro individual ou por CSV (`nome,email,papel,instituicao,id_institucional`);
- senha temporária padrão `acbd1234` para novas contas;
- troca obrigatória da senha no primeiro acesso;
- aviso de cadastro enviado por SMTP ao endereço cadastrado;
- status de entrega exibido no painel administrativo;
- autocadastro público, ativação por convite e credenciamento MTR desativados para novos fluxos;
- Mentores continuam administrando cenários, turmas e resultados, mas não criam contas.


## W7.1.1 — acessos da tela inicial

- corrige os comandos PRIMEIRO ACESSO e CADASTRO CENTRALIZADO, que eram blocos visuais sem navegação;
- adiciona rota dedicada `/primeiro-acesso`;
- adiciona atalhos equivalentes na tela de login;
- não altera o servidor.

## W7.2 — avaliação pedagógica e simplificação do Administrador

- Mentor classifica empreendimentos de suas turmas como `APROVADO` ou `REPROVADO`.
- Parecer textual opcional de até 4.000 caracteres, com autoria e data da avaliação.
- Aluno visualiza a avaliação e o parecer na área `MINHA EMPRESA`.
- A avaliação fica persistida no servidor e não é apagada por sincronizações posteriores do Aluno.
- O Administrador deixa de exibir na barra lateral os atalhos redundantes `VISÃO GERAL`, `USUÁRIOS` e `IMPORTAR CSV`.

## W7.3 — Gestão centralizada de turmas e matrículas

- Administrador cria e nomeia turmas e escolhe o Mentor responsável.
- Numeração sequencial permanente de turmas (`T1`, `T2`, ...).
- Cadastro de Aluno pode definir a turma imediatamente.
- Matrícula automática por turma no padrão `T{turma}A{sequência}`.
- Sequências de matrícula não são reutilizadas.
- Transferências preservam o histórico de matrículas.
- Aluno sem turma fica em `AGUARDANDO TURMA`.
- CSV aceita a coluna `turma` por número, identificador `Tn` ou nome exato.
- Ingresso do Aluno por código foi desativado.
- Mentor deixa de criar turmas, mas pode alterar o cenário de suas turmas.
- Novos empreendimentos herdam o vínculo administrativo atual; empreendimentos já vinculados preservam a turma histórica em transferências.

## W7.3.1 — correção de integração

- corrige regressão entre exclusão administrativa e gestão centralizada de turmas;
- restaura `deleteUserAsPrimary` e `deleteClassAsPrimary`;
- restaura endpoints Web de exclusão do Administrador Principal;
- exclusão de turma encerra a matrícula ativa e move o Aluno para `AGUARDANDO TURMA`;
- mantém IDs automáticos `T{turma}A{sequência}`;
- preserva filtros de avaliação do Mentor e ações destrutivas com confirmação `EXCLUIR`.
