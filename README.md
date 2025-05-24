# Sistema de Leilão - Go Expert

Este projeto implementa um sistema de leilão online com funcionalidades de:
- Criação de leilões
- Fechamento automático de leilões após um período determinado
- Processamento de lances (bids)
- Consulta de leilões e lances

## Recursos

- API REST com Gin Web Framework
- Persistência em MongoDB
- Clean Architecture
- Concorrência com goroutines para gerenciamento automático de leilões
- Fechamento automático de leilões expirados

## Requisitos

- Go 1.20+
- Docker e Docker Compose
- MongoDB

## Executando o Projeto

### Modo mais simples: Docker Compose

```bash
# Na raiz do projeto, execute:
docker-compose up --build
```

Isso iniciará a aplicação e o MongoDB sem autenticação para facilitar os testes.
A API estará disponível em http://localhost:8080.

### Testando a aplicação

#### 1. Criando um leilão

```bash
curl -X POST http://localhost:8080/auction \
  -H "Content-Type: application/json" \
  -d '{
    "product_name": "Smartphone",
    "category": "Electronics",
    "description": "Um smartphone de última geração para testes",
    "condition": 1
  }'
```

#### 2. Listando leilões ativos

```bash
curl -X GET http://localhost:8080/auction
```

#### 3. Verificando o fechamento automático

- Crie um leilão usando o comando acima
- Aguarde pelo menos 20 segundos (tempo configurado em AUCTION_INTERVAL)
- Liste os leilões novamente para verificar se o status mudou para "Completed" (1)

#### 4. Criando um lance

```bash
# Substitua AUCTION_ID pelo ID retornado na criação do leilão
curl -X POST http://localhost:8080/bid \
  -H "Content-Type: application/json" \
  -d '{
    "auction_id": "AUCTION_ID",
    "user_id": "user123",
    "amount": 1000.00
  }'
```

## Explicação da Implementação

### Funcionalidade de Fechamento Automático

A implementação do fechamento automático de leilões foi realizada usando goroutines. Quando um leilão é criado, ele é registrado em um mapa com o tempo previsto de expiração. Uma goroutine independente monitora continuamente este mapa e fecha os leilões que já expiraram.

Principais componentes da solução:

1. **Monitoramento Contínuo**: Uma goroutine dedicada verifica periodicamente os leilões ativos.
2. **Controle de Concorrência**: Uso de mutex para acesso thread-safe ao mapa de leilões ativos.
3. **Fechamento Automático**: Atualização do status do leilão no banco de dados quando o tempo expira.

O intervalo de duração do leilão é configurável através da variável de ambiente `AUCTION_INTERVAL`.

## Estrutura do Projeto

O projeto segue a Clean Architecture:

- **cmd/auction**: Ponto de entrada da aplicação
- **configuration**: Configurações (banco de dados, logging, etc.)
- **internal/entity**: Entidades de domínio
- **internal/usecase**: Casos de uso da aplicação
- **internal/infra**: Implementações de infraestrutura

## Testes

Os testes da funcionalidade de fechamento automático podem ser executados com:

```bash
go test ./internal/infra/database/auction/... -v
```

Este comando executa testes em memória que não requerem um MongoDB real.