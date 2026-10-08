# W5.0 — Empresa remota do Aluno

## Objetivo

W5.0 inicia a área funcional do Aluno sem alterar o protocolo RC1.8 do JED Servidor.
A aplicação web passa a buscar as empresas pertencentes ao Aluno autenticado, selecionar uma
empresa e apresentar seu estado real no navegador.

## Fluxo

1. O Aluno entra pela sessão HttpOnly implementada no W4.
2. `GET /api/student/companies` é atendido pelo backend Next.js.
3. O backend recupera o token HttpOnly e chama `GET /api/v1/companies` no JED Servidor.
4. O navegador recebe apenas as empresas permitidas para a conta, nunca o token do servidor.
5. A empresa selecionada é enviada localmente ao JED Core em WebAssembly.
6. Indicadores e Score são calculados pelo mesmo motor Go usado como base no Windows.

## Escopo desta subversão

Incluído:

- listagem das empresas do Aluno;
- seleção quando houver mais de uma empresa;
- estado atual de caixa, preço, clientes, reputação, estoque e insumos;
- progresso da simulação;
- dados da última rodada;
- indicadores acumulados calculados no JED Core;
- Score JED calculado no JED Core;
- revisão e data da última sincronização;
- tratamento de conta sem empresa sincronizada.

Não incluído ainda:

- criação de empresa no navegador;
- alteração de decisões;
- processamento de uma nova semana;
- persistência IndexedDB;
- sincronização PUT feita pelo navegador;
- resolução de conflitos de revisão.

Esses itens permanecem para as próximas etapas W5.x.

## Segurança

A nova rota BFF verifica sessão e papel `aluno`. O navegador nunca recebe o token do
JED Servidor. Respostas são marcadas `private, no-store`.
