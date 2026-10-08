# Configuração do SignPath Foundation

Este repositório já contém a estrutura necessária para você começar.

## 1. Antes de publicar

Procure por:

`SEU_USUARIO_GITHUB`

e substitua pelo seu usuário ou organização real do GitHub nos arquivos:

- `README.md`
- `CODE_SIGNING_POLICY.md`

## 2. Crie o repositório

Nome sugerido:

`JED-Simulador`

Defina-o como **Public**.

Envie todo o conteúdo deste pacote para a raiz do repositório.

## 3. Ative autenticação em dois fatores

Ative 2FA/MFA na conta do GitHub que manterá o projeto.

## 4. Faça o primeiro push

O workflow `Build Windows` deverá:

1. testar o código;
2. executar o self-test;
3. gerar recursos do Windows;
4. compilar x64 e x86;
5. publicar um artefato não assinado na aba Actions.

O workflow utiliza `go-winres v0.3.3` para adicionar ícone e metadados do Windows aos executáveis.

## 5. Publique uma primeira release não assinada

Crie uma tag, por exemplo:

`v2.0.0-rc1.8`

Use o artefato produzido pelo GitHub Actions para criar uma Release pública.

A ideia é mostrar à SignPath que o projeto já existe e já é distribuído na forma que será posteriormente assinada.

## 6. Solicite o SignPath Foundation

Preencha o formulário da SignPath Foundation informando:

- URL do repositório público;
- URL da release;
- licença GPLv3;
- finalidade educacional;
- responsável pelo projeto;
- link para a seção `Code signing policy`;
- link para a política de privacidade.

Há um modelo de texto em `docs/SIGNPATH_APPLICATION_TEMPLATE.md`.

## 7. Depois da aprovação

No GitHub, abra:

`Settings → Secrets and variables → Actions`

Crie o secret:

- `SIGNPATH_API_TOKEN`

Crie as variables:

- `SIGNPATH_ENABLED` = `true`
- `SIGNPATH_ORGANIZATION_ID`
- `SIGNPATH_PROJECT_SLUG`
- `SIGNPATH_SIGNING_POLICY_SLUG`
- `SIGNPATH_ARTIFACT_CONFIGURATION_SLUG`

Use exatamente os valores fornecidos/configurados na sua conta SignPath.

## 8. Artifact Configuration

O XML preparado está em:

`.signpath/artifact-configurations/jed-windows.xml`

Cadastre uma Artifact Configuration no projeto SignPath usando esse conteúdo.

Ela foi preparada para assinar os seis executáveis Windows do JED e verificar que todos possuem:

- Product Name: `JED Simulador`
- Product Version: `2.0.0.18`
- File Version: `2.0.0.18`

## 9. Ative o GitHub App do SignPath

Após a aprovação, instale a integração oficial do SignPath no repositório conforme as instruções exibidas no painel da SignPath.

## 10. Primeira assinatura

Abra:

`Actions → SignPath Release → Run workflow`

O workflow só executa a assinatura quando:

`SIGNPATH_ENABLED = true`

Depois da aprovação manual da solicitação na SignPath, o workflow baixa o artefato assinado e o publica como:

`JED-Simulador-Windows-signed`

## 11. Não publique segredos

Nunca coloque no código:

- token do SignPath;
- senhas;
- credenciais de alunos;
- credenciais de tutores.

O token deve existir apenas como GitHub Actions Secret.
