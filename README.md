# WMS Escolar — Sistema de Gestão de Estoque

![Go](https://img.shields.io/badge/Go-1.22+-00ADD8?logo=go&logoColor=white)

Sistema web de gestão de estoque desenvolvido para ambiente escolar, com foco no controle de materiais, depósitos, movimentações e organização por turmas.

O projeto foi evoluído a partir de uma base existente, porém passou por uma reestruturação arquitetural significativa, incluindo a migração do backend para Go e reorganização completa do fluxo do sistema. Por esse motivo, este repositório representa a versão atual consolidada da aplicação.

---

## Observação sobre o histórico do projeto

Este repositório não representa o início do desenvolvimento do zero.

A base inicial já existia em outro ambiente, mas o sistema passou por uma reformulação completa de arquitetura e stack, incluindo:

- Substituição do backend anterior por Go (Gin)
- Reestruturação do fluxo de autenticação e permissões
- Revisão do modelo de dados e estrutura de estoque
- Separação clara entre frontend e backend
- Reorganização das responsabilidades das camadas do sistema

Este repositório representa a versão atual e estruturada do projeto.

---

## Funcionalidades

- Autenticação com JWT
- Aprovação de contas por usuários com perfil de gestão
- Controle de usuários por perfil (gestão e professor)
- Cadastro e gerenciamento de depósitos
- Controle de estoque de materiais
- Registro de movimentações de entrada e saída
- Organização por turmas
- Histórico de operações

---

## Arquitetura

Frontend:
- HTML, CSS e JavaScript (SPA sem framework)

Backend:
- Go (Gin)
- Arquitetura em camadas:
  - Routes
  - Handlers
  - Services
  - Repositories

Banco de dados:
- PostgreSQL (Supabase)

Autenticação:
- JWT + bcrypt

Infraestrutura:
- VPS Linux (Ubuntu / Hostinger)

---

## Fluxo do sistema

Frontend → API Go → Middleware de autenticação → Handlers → Services → Repositories → PostgreSQL (Supabase)

---

## Objetivo

Sistema desenvolvido para gestão de estoque escolar e apoio didático, com foco em organização de materiais, controle de movimentações e estrutura escalável para expansão futura.

---

## Tecnologias

- Go
- Gin
- JavaScript
- HTML
- CSS
- PostgreSQL (Supabase)
- JWT

---

## Status

Sistema funcional e em evolução contínua.

---

## Nota

O projeto passou por uma reestruturação arquitetural significativa. Parte da base original foi preservada, porém o backend e a organização geral do sistema foram reformulados para melhorar escalabilidade, manutenção e clareza da arquitetura.
