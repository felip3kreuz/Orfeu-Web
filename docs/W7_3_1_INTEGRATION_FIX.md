# W7.3.1 — Correção de integração

A W7.3 de turmas centralizadas sobrepôs partes da atualização anterior de exclusão administrativa. O resultado foi uma inconsistência: `online_admin_test.go` ainda chamava `deleteUserAsPrimary` e `deleteClassAsPrimary`, mas esses métodos haviam desaparecido de `online_server.go`.

Esta revisão integra os dois conjuntos de mudanças em uma única base.

## Garantias

- `go test ./...` volta a compilar e passar;
- somente o Administrador Principal pode excluir contas e turmas;
- o Administrador Principal continua protegido contra autoexclusão;
- Mentor com turma não pode ser excluído antes das turmas;
- exclusão de Aluno remove seus empreendimentos persistidos;
- exclusão de turma preserva empreendimentos, desvincula-os da turma e coloca os Alunos em `AGUARDANDO TURMA`;
- matrícula ativa da turma excluída é encerrada no histórico;
- criação centralizada, transferência e IDs `T{turma}A{sequência}` permanecem válidos;
- filtros de avaliação do Mentor permanecem disponíveis.
