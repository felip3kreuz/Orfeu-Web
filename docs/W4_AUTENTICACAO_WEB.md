# W4 — autenticação Web e JED Servidor

A W4 conecta o JED Web à API existente do JED Servidor RC1.8 sem expor o token de sessão ao JavaScript do navegador.

## Arquitetura

Navegador → Route Handlers do Next.js → JED Servidor RC1.8.

O endpoint `/api/auth/login` encaminha e-mail e senha para `/api/v1/login`. O token opaco retornado pelo servidor é gravado em cookie `Secure`, `HttpOnly`, `SameSite=Lax`, com duração máxima de 12 horas — igual à sessão do servidor.

As páginas `/admin`, `/mentor` e `/aluno` consultam `/api/v1/me` no lado do servidor e redirecionam usuários para o painel correspondente ao papel autenticado. O papel legado `tutor` é normalizado para `mentor` apenas na interface web; o protocolo RC1.8 permanece compatível.

## Variável de ambiente obrigatória

Na Vercel, configure uma variável server-side (sem prefixo `NEXT_PUBLIC_`):

    JED_SERVER_URL=https://seu-servidor-jed.exemplo

Em produção, a W4 exige HTTPS para impedir o envio de credenciais por conexão não criptografada.

A variável deve ser adicionada aos ambientes Production e Preview quando ambos precisarem acessar o servidor. Depois de alterar uma variável na Vercel, faça um novo deploy.

## Requisito de rede

`localhost`, `127.0.0.1` e endereços privados da máquina onde JED_Servidor.exe roda não são alcançáveis pela Vercel. O JED Servidor precisa ter um endereço HTTPS publicamente acessível (diretamente ou por túnel/reverse proxy).

## Endpoints Web adicionados

- `GET /api/server/health`
- `POST /api/auth/login`
- `GET /api/auth/me`
- `POST /api/auth/logout`

## Painéis W4

- `/admin`
- `/mentor`
- `/aluno`

Nesta etapa os painéis comprovam autenticação, autorização por papel e sessão. Funcionalidades de negócio do Aluno entram no W5; funcionalidades administrativas e de Mentoria entram no W6.
