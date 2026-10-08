# W7.1.1 — correção dos acessos da tela inicial

- `CADASTRO CENTRALIZADO` agora é um link real para `/admin`; sem sessão, o fluxo segue para o login e, após autenticação de Administrador, abre o painel administrativo.
- `PRIMEIRO ACESSO` agora abre `/primeiro-acesso`, com autenticação pela senha temporária e redirecionamento automático para a troca obrigatória de senha.
- A tela de login passou a oferecer atalhos explícitos para Primeiro Acesso e Cadastro Centralizado.
- Nenhuma alteração no protocolo ou no JED Servidor.
