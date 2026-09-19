## MODIFIED Requirements

### Requirement: Padrões do frontend
As telas de administração MUST seguir as convenções do projeto: Vue 3 com `<script setup>`, Tailwind com variantes `dark:` em todos os componentes novos, dados passados por props (sem provide/inject) e build incluído em `web/static/`.

#### Scenario: Tema escuro
- **WHEN** o usuário usa o tema escuro
- **THEN** as telas de administração são exibidas com as cores do tema escuro

#### Scenario: Tema Bio
- **WHEN** o usuário usa o tema Bio
- **THEN** as telas de administração são exibidas com as cores do tema Bio, sem nenhuma variante do tema escuro aplicada
