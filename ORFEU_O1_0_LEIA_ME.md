# Orfeu Web — O1.0

## Base verificada

Este pacote deriva de **JED-Simulador-RC1.8-W7.3.1-IntegratedFix.zip**, cuja soma SHA-256 é:

`c9a569e0c42e9c1c10844fd928ac791051509aa17941cc85844f246fabe2f216`

A alteração O1.0 é **exclusivamente da interface Web**. O servidor Go, as rotas de API, o motor de simulação WebAssembly, a estrutura de dados, o esquema de matrícula `TnAm` e a organização das telas permanecem compatíveis com a W7.3.1.

## Funcionalidades da O1.0

1. Nova identidade de apresentação **Orfeu Web** nas telas de entrada, cabeçalhos, títulos e status; `O1.0` é o identificador público da versão.
2. Paleta centralizada no início de `web/app/globals.css`: verde musgo, verde oliva, tons off-white e terracota. **As dimensões, a tipografia, o grid e a disposição dos componentes W7.3.1 foram preservados.**
3. Novo ícone vetorial do Orfeu em `web/app/icon.svg` e `web/public/orfeu-mark.svg`.
4. Tutorial interativo individualizado para **Aluno, Mentor e Administrador**, com apresentação na primeira visita, botão **TUTORIAL** na barra lateral, destaque de elementos reais, exemplos fictícios, anterior/próximo, sair, retomar e reiniciar.
5. Progresso dos tutoriais salvo apenas no navegador do usuário, via `localStorage`. Sem novas tabelas, permissões ou rotas no servidor. Pode ser apagado ao limpar os dados do navegador.

## Integração e operação

- **Instalação limpa:** extraia o ZIP, configure as variáveis de ambiente e publique conforme as instruções já existentes de Web/Railway/Vercel.
- **Atualização de W7.3.1:** preserve seus `.env`, banco de dados e arquivos JSON reais. Atualize o front-end pela nova pasta `web` e mantenha as dependências e configuração do servidor atual.
- **Não substitua arquivos de dados de produção** pelo conteúdo demonstrativo do ZIP.
- **Build original:** no diretório-raiz, `bash scripts/build-web.sh` (gera o WASM e compila a Web) ou, com WASM já criado, `npm --prefix web run build`.
- **Ambiente local:** `npm --prefix web install` e `npm --prefix web run dev`.
- **Verificações de backend:** `go test ./...` (testes preexistentes).

## Notas de segurança e compatibilidade

- `jed-core`, `jed-server`, rotas `/api/*`, arquivos Go, nomes das mensagens de eventos `jed:*` e demais identificadores técnicos **não foram renomeados**.
- A renomeação pública **não envolve ainda** Windows, Android, o servidor nem os e-mails enviados diretamente pelo servidor Go. Esses sistemas podem continuar a usar identificadores JED sem afetar o funcionamento da Web.
- A W9.0 em outra conversa **não foi utilizada como base** e não se deve instalar este pacote sobre ela sem revisão/diff.
- Os tutoriais são explicativos: não criam usuários, não fazem compras, não encerram semanas e não aprovam/reprovam empreendimentos.
- A persistência do progresso está limitada ao navegador, não a uma conta sincronizada entre dispositivos.

## Arquivos Web modificados ou adicionados

- `web/app/globals.css`, `web/app/layout.js` e arquivos de interface em `web/app/*` e `web/components/*` — novos textos e cores.
- `web/components/orfeu-tutorial.js` — tutorial contextual por perfil.
- `web/components/jed-shell.js` — inclusão do acesso ao tutorial.
- `web/components/admin-workspace.js` — `id` de seção para ancoragem do tutorial.
- `web/components/panel-navigation.js` — atributos de referência sem alterar a navegação.
- `web/app/icon.svg`, `web/public/orfeu-mark.svg` — ícones Orfeu.
- `web/package.json` — pacote web `orfeu-web` / semver `1.0.0`; versão pública O1.0.

## Roteiro mínimo de homologação

1. Verificar a landing e o login Orfeu com o favicon novo.
2. Entrar com conta de cada perfil e verificar a oferta do tutorial na primeira visita.
3. Avançar pelas etapas e verificar o destaque das seções reais, sem alteração de dados.
4. Usar SAIR e TUTORIAL novamente para testar retomada; concluir e abrir TUTORIAL para testar reinício.
5. Administrador: criar turma e matricular aluno; manter padrão `T25A4`.
6. Aluno: criar negócio, preencher Persona e Canvas, simular semana e sincronizar.
7. Mentor: abrir resultados, classificar empresa como APROVADO/REPROVADO com comentário; conferir exibição para aluno.
8. Conferir comportamento responsivo e estados de erro/pendência com a nova paleta.

Obs.: homologação integrada com servidor e produção deve ser feita em ambiente de testes antes da publicação.
