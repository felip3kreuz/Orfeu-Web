# W3 — Next.js + primeira implantação na Vercel

## Objetivo

Criar a primeira aplicação web navegável do JED Simulador e provar, dentro da própria interface,
que o JED Core em Go é carregado e executado no navegador por WebAssembly.

A W3 ainda não autentica usuários nem se conecta ao JED Servidor. Essas responsabilidades entram
na W4. Os painéis completos de Aluno, Mentor e Administrador serão construídos nas etapas seguintes.

## Estrutura

- `web/app/`: aplicação Next.js.
- `web/lib/jed-core.js`: adaptador do navegador para o módulo Go/WebAssembly.
- `scripts/build-web.sh`: compila o WASM, copia os artefatos para `web/public/wasm` e executa o build Next.js.
- `vercel.json`: configura a Vercel para construir o projeto a partir da raiz do repositório.
- `.github/workflows/build-web.yml`: valida Go + WASM + Next.js em Pull Requests e pushes para `main`.

## Fluxo de build

1. `scripts/build-wasm.sh` gera `dist/wasm/jed-core.wasm` e o runtime `wasm_exec.js`.
2. `scripts/build-web.sh` copia os artefatos para `web/public/wasm/`.
3. `next build` gera `web/.next`.
4. A Vercel publica a aplicação Next.js.

`web/public/wasm/` é gerado durante o build e não deve ser versionado.

## Primeira página

A página inicial mostra:

- identidade da versão Web RC1.8;
- estado de inicialização do JED Core;
- versão e protocolo da ponte WASM;
- contagem dos catálogos de canais e ferramentas digitais;
- os três papéis previstos: Aluno, Mentor e Administrador;
- indicação explícita de que a conexão autenticada com o servidor entra na W4.

## Vercel

Importe o repositório `JED-Simulador` na Vercel sem definir `web/` como Root Directory. A raiz do
projeto deve continuar sendo a raiz do repositório, porque a compilação WebAssembly precisa acessar
`go.mod`, `cmd/jed-wasm` e `internal/`.

O `vercel.json` fornece:

- framework: Next.js;
- instalação: `npm --prefix web install`;
- build: `npm run build`;
- saída: `web/.next`.

## Critério de conclusão da W3

A W3 está concluída quando:

1. `Build Windows`, `Build WebAssembly` e `Build Web` passam no Pull Request;
2. o Pull Request é integrado à `main`;
3. o repositório é importado na Vercel;
4. a URL da Vercel abre a página inicial;
5. o cartão de estado informa `JED Core ativo no navegador`.
