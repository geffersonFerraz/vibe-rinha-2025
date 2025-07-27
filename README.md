# 🚀 Vibe Rinha 2025 - Sistema de Processamento de Pagamentos

Um sistema de processamento de pagamentos de alta performance desenvolvido em Go, projetado para lidar com alta concorrência e garantir resiliência através de múltiplas estratégias de fallback.

## 📋 Visão Geral

Este projeto implementa um sistema distribuído de processamento de pagamentos com duas funcionalidades principais:

- **Payment Processor**: Recebe e processa pagamentos com estratégias de fallback inteligentes
- **Payment Summary**: Fornece resumos agregados de pagamentos processados

## 🏗️ Arquitetura

### Componentes Principais

```
┌─────────────────┐    ┌─────────────────┐    ┌─────────────────┐
│ LB Caddy Proxy  │    │ Payment Summary │    │  KeyDB/Redis    │
│   (Port 9999)   │    │   (Port 9092)   │    │   (Port 6379)   │
└─────────────────┘    └─────────────────┘    └─────────────────┘
         │                       │                       │
         └───────────────────────┼───────────────────────┘
                                 │
                    ┌──────────────────┐
                    │ Payment Processor│
                    │   (Port 9091)    │
                    └──────────────────┘
```

### Tecnologias Utilizadas

- **Go 1.24.5**: Linguagem principal com otimizações de performance
- **KeyDB**: Banco de dados em memória para pub/sub e cache
- **Caddy**: Proxy reverso com compressão gzip
- **Docker & Docker Compose**: Containerização e orquestração
- **HTTP/2**: Protocolo de comunicação otimizado

## 🚀 Funcionalidades

### Payment Processor (`pp`)

- **Endpoint**: `POST /payments`
- **Porta**: 9091
- **Funcionalidades**:
  - Processamento assíncrono de pagamentos
  - Sistema de fallback inteligente com múltiplos processadores
  - Health check automático dos processadores
  - Pool de conexões HTTP reutilizáveis
  - Graceful shutdown
  - Garbage collector otimizado

### Payment Summary (`ps`)

- **Endpoint**: `GET /payments-summary`
- **Porta**: 9092
- **Funcionalidades**:
  - Agregação de dados em tempo real
  - Filtros por período (from/to)
  - Limpeza automática de dados antigos
  - Separação por valor (default/fallback)
  - Thread-safe com mutex de leitura/escrita

## 🔧 Configuração e Deploy

### Pré-requisitos

- Docker e Docker Compose
- Go 1.24.5+ (para desenvolvimento local)

### Variáveis de Ambiente

```bash
# Processadores de pagamento
PAYMENT_PROCESSOR_URL=http://processor1:8080
PAYMENT_PROCESSOR_FALLBACK_URL=http://processor2:8080

# Logs
LOG_ON=true

# Limites de memória
GOMEMLIMIT=20MiB  # Payment Processor
GOMEMLIMIT=30MiB  # Payment Summary
```

### Execução Local

```bash
# Clonar o repositório
git clone <repository-url>
cd vibe-rinha-2025

# Executar com Docker Compose
docker-compose up -d

# Ou executar localmente
go run main.go pp  # Payment Processor
go run main.go ps  # Payment Summary
```

### Execução Individual

```bash
# Payment Processor
go run main.go pp

# Payment Summary
go run main.go ps
```

## 📊 API Endpoints

### Processar Pagamento

```http
POST /payments
Content-Type: application/json

{
  "correlation_id": "uuid-123",
  "amount": 150.50
}
```

**Resposta**: `204 No Content`

### Obter Resumo de Pagamentos

```http
GET /payments-summary?from=2024-01-01T00:00:00Z&to=2024-01-01T23:59:59Z
```

**Resposta**:
```json
{
  "default": {
    "totalRequests": 150,
    "totalAmount": 7500.00
  },
  "fallback": {
    "totalRequests": 25,
    "totalAmount": 5000.00
  }
}
```

## ⚡ Otimizações de Performance

### Garbage Collector
- **GC Percent**: 100 (força GC a cada 100MB)
- **Max Stack**: 32MB por goroutine
- **GOMAXPROCS**: Utiliza todas as CPUs disponíveis

### Pool de Conexões
- **HTTP Client Pool**: Reutilização de clientes HTTP
- **KeyDB Pool**: 10 conexões com 5 idle mínimas
- **Timeout Configurações**: Otimizadas para baixa latência

### Memória
- **Limpeza Automática**: Dados antigos removidos a cada 5 minutos
- **Tamanho Máximo**: 10k entradas no summary
- **Retenção**: Máximo 24 horas de dados

### Concorrência
- **HTTP/2**: Até 100 streams concorrentes (processor)
- **HTTP/2**: Até 1000 streams concorrentes (summary)
- **Canal Buffer**: 5000 mensagens em buffer

## 🔄 Estratégia de Fallback

O sistema implementa uma estratégia inteligente de fallback:

1. **Health Check**: Verifica processadores a cada ~5 segundos
2. **Critérios de Seleção**:
   - Processador principal: < 3s de resposta
   - Processador fallback: < 3s de resposta
   - Fila local: Quando ambos falham
3. **Transição Automática**: Muda entre processadores sem downtime

## 📈 Monitoramento

### Métricas Disponíveis
- Tempo de resposta dos processadores
- Status de falha dos serviços
- Número de requisições processadas
- Uso de memória e CPU

### Logs
- Logs de erro configuráveis via `LOG_ON`
- Graceful shutdown com logs informativos
- Health check status

## 🐳 Docker

### Imagem
```dockerfile
FROM golang:1.24-alpine AS builder
# Build multi-stage para imagem otimizada
```

### Recursos Limitados
- **Payment Processor**: 0.45 CPU, 60MB RAM
- **Payment Summary**: 0.25 CPU, 50MB RAM
- **KeyDB**: 0.2 CPU, 140MB RAM
- **Caddy**: 0.6 CPU, 100MB RAM

## 🔒 Segurança

- **Timeout Configurados**: Prevenção de DoS
- **Validação de Input**: Parsing seguro de JSON
- **Graceful Shutdown**: Finalização limpa de conexões
- **Resource Limits**: Controle de uso de recursos

## 🧪 Testes

Para executar os testes:

```bash
go test ./...
```

## 📝 Licença

Este projeto foi desenvolvido para a Rinha de Backend 2025.

---

*Desenvolvido com ❤️ em Go para alta performance e resiliência com apoio do vibe Cursor AI  `¯\_(ツ)_/¯`* 