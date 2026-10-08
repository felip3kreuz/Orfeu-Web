# W7.3 — Gestão centralizada de turmas e matrículas

## Objetivo

A W7.3 elimina o ingresso do Aluno por código/convite e centraliza no Administrador a criação das turmas e a vinculação dos Alunos.

## Fluxo administrativo

1. O Administrador cadastra o Mentor.
2. O Administrador cria a turma, define o nome e escolhe o Mentor responsável.
3. O servidor atribui à turma um número sequencial permanente (`T1`, `T2`, ...).
4. Ao cadastrar ou importar um Aluno, o Administrador pode selecionar a turma imediatamente.
5. O servidor gera uma matrícula sequencial permanente no formato `T{turma}A{aluno}`. Exemplo: o quarto Aluno da turma 25 recebe `T25A4`.
6. Alunos cadastrados sem turma ficam em `AGUARDANDO TURMA` e podem ser vinculados posteriormente pelo Administrador.

## Regras de matrícula

- A sequência de Alunos é monotônica em cada turma e nunca é reutilizada.
- Transferir um Aluno encerra a matrícula anterior e gera uma nova matrícula na turma de destino.
- O histórico de matrículas permanece salvo na conta do Aluno.
- O vínculo atual é o único usado para novas empresas sem turma; empresas já associadas a uma turma preservam o vínculo histórico após uma transferência.

## Mentor

- Mentores não criam nem nomeiam turmas.
- O Administrador define as turmas sob responsabilidade de cada Mentor.
- O Mentor continua podendo criar cenários e alterar o cenário pedagógico das turmas que lhe foram atribuídas.
- A avaliação APROVADO/REPROVADO e o parecer pedagógico da W7.2 permanecem inalterados.

## Aluno

- Não há mais campo de código de turma nem ação ENTRAR NA TURMA.
- A tela TURMA exibe o vínculo definido pelo Administrador e a matrícula atual.
- Novos empreendimentos usam automaticamente a turma atual quando houver uma.

## CSV

O cadastro centralizado aceita a coluna opcional `turma`:

`nome,email,papel,instituicao,id_institucional,turma`

Para Alunos, `turma` aceita o identificador (`T25`), apenas o número (`25`) ou o nome exato da turma. Campo vazio mantém o Aluno em `AGUARDANDO TURMA`.
