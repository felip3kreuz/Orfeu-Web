# JED Simulador — W6.2 / fluxo do Aluno

W6.2 corrige o estado da área do Aluno quando a conta ainda não possui empresa sincronizada.

## Correções

- `MINHA EMPRESA`, `DECISÕES` e `TURMAS` passam a possuir destinos reais mesmo sem empresa remota.
- sucesso e erro ao entrar em turma são exibidos também no estado sem empresa.
- turmas já vinculadas ficam visíveis antes da primeira empresa ser sincronizada.
- entrada em turma normaliza o código, impede envio vazio e mostra estado de processamento.
- após o vínculo, a lista de turmas é recarregada imediatamente.

## Importante

O protocolo RC1.8 trata vínculo de turma e empresa como estados separados. Entrar em uma turma não cria automaticamente uma empresa. A criação inicial de empresa continua pertencendo à interface Windows; depois da sincronização, a Web libera decisões, processamento pelo Core WASM e sincronização das rodadas.

Nenhum endpoint do JED Servidor foi alterado nesta correção.
