# JED Simulador — W6.0 / RC1.8 Web Final

W6.0 consolida a primeira versão web operacional baseada na v2.0 RC1.8.

## Aluno

- carrega empresas da conta autenticada;
- restaura rascunhos locais apenas quando a revisão remota de base ainda coincide;
- edita preço, marketing, promoção, equipe, delivery, canais e ferramentas digitais;
- processa a semana no JED Core compilado para WebAssembly;
- mantém o resultado localmente até a sincronização;
- sincroniza a empresa pelo mesmo endpoint RC1.8 usado pelo cliente online;
- mostra indicadores acumulados, Score JED, progresso e última rodada;
- permite entrar em turma por código.

A separação entre **Processar semana** e **Sincronizar com servidor** evita que uma falha de rede obrigue o Aluno a processar novamente a mesma rodada. Enquanto houver estado local pendente, ele é salvo em `localStorage` com a revisão remota de base.

## Mentor

- lista turmas e códigos de entrada;
- cria novas turmas;
- gera convites de Aluno;
- lista credenciais emitidas;
- acompanha empresas sincronizadas de suas turmas;
- calcula Score JED das empresas no próprio Core WebAssembly.

## Administrador

- lista usuários do JED Servidor;
- ativa/desativa contas;
- concede/remove permissão de Mentor para credenciar outros Mentores;
- cria novos Administradores;
- emite credenciais de Mentor;
- lista credenciais de Mentor já emitidas.

## Segurança

O token RC1.8 permanece em cookie `Secure`, `HttpOnly` e `SameSite=Lax`. O navegador chama apenas rotas `/api/...` da aplicação Next.js. Essas rotas validam o papel da sessão antes de acessar a API do JED Servidor.

A URL do servidor continua server-side na variável `JED_SERVER_URL` e não usa prefixo `NEXT_PUBLIC_`.

## Compatibilidade

A versão web mantém:

- JED Core em Go;
- protocolo RC1.8 do JED Servidor;
- JSON de `Empresa` e `RemoteCompany`;
- build Windows independente;
- build WebAssembly independente;
- build Next.js/Vercel.

## Limite de implantação

O software pode ser publicado definitivamente na Vercel. Para operação contínua, `JED_SERVER_URL` deve apontar para um endereço HTTPS permanente do JED Servidor. Um Quick Tunnel `trycloudflare.com` continua adequado apenas para validação temporária.
