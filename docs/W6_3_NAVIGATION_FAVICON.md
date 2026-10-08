# W6.3 — Navegação dos painéis + favicon

Esta revisão é cumulativa sobre a W6.2.

## Navegação

- substitui a navegação por âncoras passivas dos painéis por um componente cliente;
- VISÃO GERAL, TURMAS e ALUNOS no painel Mentor passam a localizar e rolar explicitamente até a seção correspondente;
- MINHA EMPRESA, DECISÕES e TURMAS no painel Aluno usam o mesmo mecanismo;
- a navegação do Administrador também passa pelo mesmo componente para evitar o mesmo defeito;
- o item ativo acompanha a seção visível;
- o hash da URL é atualizado sem recarregar a página;
- o destino ALUNOS passa a usar o identificador semântico `#alunos`.

## Fluxo do Aluno herdado da W6.2

- mantém os destinos reais de MINHA EMPRESA, DECISÕES e TURMAS mesmo quando não existe empresa;
- mantém feedback de entrada em turma, erros, sucesso e lista de vínculos;
- mantém normalização do código de turma para maiúsculas.

## Favicon

- `web/app/favicon.ico` é uma cópia byte a byte de `jed_icon.ico` usado pelo executável Windows;
- o metadata do App Router declara o mesmo `/favicon.ico` como ícone e shortcut icon.
