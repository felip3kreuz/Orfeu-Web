# JED Servidor em produção — Railway

Esta configuração publica o `JED Servidor` como serviço persistente separado do frontend Vercel.

## Variáveis

- `JED_SERVER_DATA=/data`
- `JED_BOOTSTRAP_ADMIN_NAME` — usado somente quando ainda não existe Administrador no volume.
- `JED_BOOTSTRAP_ADMIN_EMAIL` — usado somente quando ainda não existe Administrador no volume.
- `JED_BOOTSTRAP_ADMIN_PASSWORD` — usado somente quando ainda não existe Administrador no volume.
- `PORT` é fornecida automaticamente pelo Railway e agora é reconhecida pelo servidor.

SMTP continua opcional via `JED_SMTP_HOST`, `JED_SMTP_PORT`, `JED_SMTP_USER`, `JED_SMTP_PASSWORD`, `JED_SMTP_FROM`, `JED_SMTP_FROM_NAME` e `JED_SMTP_SECURITY`.

## Persistência

Anexe um Volume Railway montado em `/data`. O arquivo principal será `/data/jed_server_data.json` e os backups ficarão em `/data/backups/`.

## Rede

Gere um domínio público no Railway e configure o Healthcheck Path como `/health`.

Depois, na Vercel, defina `JED_SERVER_URL` com a URL HTTPS pública do Railway, sem `/health`.
