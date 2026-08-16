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
- Integração de inteligência artificial para assistência aos usuários

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
- Assistente inteligente Otis
- Integração com inteligência artificial local

### Inteligência Artificial — Otis

O **Otis** é o assistente inteligente integrado ao ANT-Stock, desenvolvido para auxiliar os usuários durante a utilização do sistema.

Na primeira versão, o Otis atua como um assistente conversacional capaz de:

- Responder dúvidas sobre o funcionamento do sistema
- Orientar usuários sobre funcionalidades
- Explicar processos e recursos disponíveis
- Auxiliar na utilização da aplicação

A arquitetura foi projetada para permitir futuras integrações com ferramentas do sistema e automações controladas pelo backend.

---

## Arquitetura

### Frontend

- HTML
- CSS
- JavaScript
- SPA sem framework

### Backend

- Go
- Gin
- Arquitetura em camadas:
  - Routes
  - Handlers
  - Services
  - Repositories
  - AI

### Banco de dados

- PostgreSQL
- Supabase

### Inteligência Artificial

- Ollama
- Qwen3 1.7B
- Execução local no servidor

### Autenticação

- JWT
- bcrypt

### Infraestrutura

- VPS Linux
- Ubuntu
- Hostinger

---

## Fluxo do sistema

### Fluxo tradicional

Frontend → API Go → Middleware de autenticação → Handlers → Services → Repositories → PostgreSQL (Supabase)

### Fluxo da IA

Frontend → API Go → Serviço de IA → Ollama → Qwen3 → API Go → Frontend

O modelo de IA não possui acesso direto ao banco de dados. A comunicação com os recursos do sistema é controlada pelo backend.

---

## Objetivo

Sistema desenvolvido para gestão de estoque escolar e apoio didático, com foco em organização de materiais, controle de movimentações e estrutura escalável para expansão futura.

A arquitetura também foi projetada para permitir a utilização de inteligência artificial na assistência e automação de processos relacionados ao gerenciamento de estoque.

---

## Tecnologias

- Go
- Gin
- JavaScript
- HTML
- CSS
- PostgreSQL
- Supabase
- JWT
- bcrypt
- Ollama
- Qwen3 1.7B

---

## Status

Sistema funcional e em evolução contínua.

O assistente inteligente **Otis** está sendo desenvolvido de forma incremental, começando pela interação conversacional e posteriormente evoluindo para ferramentas e automações controladas pelo backend.

---

## Nota

O projeto passou por uma reestruturação arquitetural significativa. Parte da base original foi preservada, porém o backend e a organização geral do sistema foram reformulados para melhorar escalabilidade, manutenção e clareza da arquitetura.
