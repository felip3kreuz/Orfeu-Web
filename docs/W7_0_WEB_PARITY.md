# JED Simulador — W7.0 / Web Parity

A W7.0 transforma a versão Web de um portal de autenticação e sincronização em uma interface funcional do JED Simulador, preservando o JED Core em Go/WebAssembly e o JED Servidor RC1.8 como persistência canônica.

## Aluno

- criação de empreendimento inteiramente no navegador;
- seleção hierárquica Setor → Tipo → Especialidade usando `catalogo_negocios.json`;
- preço de referência, custo e capacidade exibidos antes da criação;
- capital inicial, dificuldade, cenário e vínculo opcional com turma;
- múltiplas empresas por Aluno;
- edição completa de Persona;
- edição de Lean Canvas;
- canais e ferramentas digitais;
- decisões semanais de preço, marketing, promoção, equipe, operação, localização e delivery;
- processamento pelo mesmo JED Core compilado para WebAssembly;
- sincronização automática da rodada com o JED Servidor;
- insumos, fornecedores, pedidos e estoque;
- financeiro, contas a pagar/receber e caixa;
- indicadores, score JED e histórico de semanas;
- Jornada JED;
- entrada em turma por código e vínculo da empresa atual a uma turma.

## Mentor

- navegação modular real em vez de âncoras de rolagem;
- criação e persistência de cenários pedagógicos;
- quatro cenários-base do JED sempre disponíveis;
- criação de turma com cenário selecionado;
- convites individuais de Aluno;
- importação simples de CSV de alunos;
- acompanhamento de turmas, alunos, empresas, semanas, caixa, resultado e score.

## Servidor

A persistência ganha `scenarios` de forma retrocompatível. Bases RC1.8 existentes continuam válidas; se o campo não existir, ele é inicializado vazio.

Novo endpoint autenticado para Mentor:

- `GET /api/v1/scenarios`
- `POST /api/v1/scenarios`

## Navegação

Os itens laterais de Aluno e Mentor passam a trocar módulos reais por evento de navegação no frontend. O Administrador mantém fallback para suas seções existentes.

## JED Core / WASM

Além de processamento, indicadores, score e revisão, a ponte Web passa a expor:

- pedido de insumos;
- compra de estoque simples;
- passo atual da Jornada JED;
- explicações ativas do Lean Canvas.

## Compatibilidade

- formato de `Empresa` preservado;
- endpoints existentes preservados;
- login/sessão HttpOnly preservados;
- favicon Windows preservado;
- Docker/Oracle continuam sendo o servidor de produção;
- Vercel continua hospedando Next.js/BFF e o WASM.
