# Relatório de validação — Orfeu Web O1.0

**Base**: W7.3.1 IntegratedFix (SHA-256 confirmado no pacote de origem).

Verificações concluídas neste ambiente:

- Análise sintática com o parser JSX do TypeScript global: **47 arquivos JavaScript, 0 erros**.
- Testes estáticos da integração dos tutoriais: **10/10 verificações passaram**.
- Análise de CSS via tinycss2: **0 erros de parser**.
- `GOPROXY=off GOSUMDB=off go test ./internal/...`: **passou** (core e wasmbridge).
- Comparação com o ZIP original: **102 arquivos originais fora de web/ preservados sem alterações**.

**Não validado neste ambiente**:

- Build completo do Next.js (`npm --prefix web run build`) — pacotes `next`/`react` não estavam disponíveis localmente, e a instalação npm não se concluiu por falta de acesso ao registry. 
- Testes ponta a ponta com login real, SMTP, servidor publicado e navegação visual no navegador.
- A suíte global `go test ./...` não completou dentro do prazo desta execução; os testes dos pacotes internos executaram com sucesso.

**Antes de produção**: instalar dependências em ambiente conectado, executar `bash scripts/build-web.sh`, homologar com servidor de teste e conferir login, conta/CPF institucional conforme a configuração local, turmas, avaliações e tutoriais em cada perfil.
