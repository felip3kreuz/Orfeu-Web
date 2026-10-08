# Checklist de Release

## Antes da release

- [ ] `go test ./...` passa.
- [ ] `go run . --self-test` passa.
- [ ] versão interna do programa foi conferida.
- [ ] README continua correto.
- [ ] PRIVACY.md continua descrevendo o comportamento real.
- [ ] não existem tokens, senhas ou dados pessoais no commit.
- [ ] alterações em `.github/workflows/` foram revisadas.
- [ ] alterações na configuração SignPath foram revisadas.

## Release não assinada, antes da aprovação SignPath

- [ ] criar tag;
- [ ] aguardar `Build Windows`;
- [ ] baixar o artefato gerado;
- [ ] publicar GitHub Release;
- [ ] indicar claramente que o binário ainda não é assinado.

## Release assinada, depois da aprovação SignPath

- [ ] executar `SignPath Release`;
- [ ] revisar a solicitação;
- [ ] aprovar manualmente a assinatura quando solicitado;
- [ ] baixar o artefato assinado;
- [ ] verificar a assinatura no Windows;
- [ ] publicar apenas o artefato final correspondente ao commit/tag da release.
