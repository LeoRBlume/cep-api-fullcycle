# CEP Lookup API

API REST em Go que consulta CEPs brasileiros realizando uma corrida entre duas APIs externas simultaneamente e retornando a resposta mais rápida. Construída com goroutines, channels e `select` para implementar uma corrida concorrente real.

---

## Como Funciona

Ao receber uma requisição de consulta de CEP, o serviço dispara duas requisições HTTP **em paralelo**:

- **BrasilAPI** — `https://brasilapi.com.br/api/cep/v1/{cep}`
- **ViaCEP** — `http://viacep.com.br/ws/{cep}/json/`

Ambas rodam em goroutines separadas e enviam seus resultados para um channel com buffer. Um `select` captura a **primeira resposta bem-sucedida** e descarta a outra. Se a resposta vencedora for um erro, o sistema aguarda o segundo resultado. Se nenhuma das APIs responder dentro de **1 segundo**, um erro de timeout é retornado.

```
Requisição ──► Goroutine A (BrasilAPI) ──┐
                                          ├──► channel ──► select ──► primeira a chegar vence
              ► Goroutine B (ViaCEP)   ──┘
```

O resultado é impresso no terminal **e** retornado como resposta HTTP em JSON.

---

## Regras de Negócio

| Regra | Comportamento |
|-------|---------------|
| Requisições paralelas | As duas APIs são chamadas simultaneamente, nunca sequencialmente |
| Corrida | Somente a resposta bem-sucedida mais rápida é utilizada |
| Timeout | Limite de 1 segundo — retorna `504 Gateway Timeout` se excedido |
| Primeira falha | Se a resposta mais rápida for um erro, o sistema aguarda a segunda |
| Ambas falham | Retorna `500 Internal Server Error` |
| Saída no terminal | Cada consulta imprime o resultado (ou erro) no stdout |

---

## API

### `GET /cep/:cep`

Consulta os dados de endereço para o CEP informado.

**Exemplo:**
```
GET http://localhost:8080/cep/01001000
```

**Resposta de sucesso (`200 OK`):**
```json
{
  "provider": "BrasilAPI",
  "data": {
    "cep": "01001000",
    "state": "SP",
    "city": "São Paulo",
    "neighborhood": "Sé",
    "street": "Praça da Sé",
    "service": "open-cep"
  }
}
```

**Resposta de timeout (`504 Gateway Timeout`):**
```json
{
  "error": "timeout: nenhuma API respondeu dentro de 1 segundo"
}
```

**Saída no terminal (em toda consulta):**
```
========================================
CEP Consultado: 01001000
Resposta de: BrasilAPI
----------------------------------------
  cep: 01001000
  state: SP
  city: São Paulo
  ...
========================================
```

---

## Estrutura de Pastas

```
cep-api/
├── cmd/
│   └── api/
│       └── main.go               # Ponto de entrada da aplicação
├── config/
│   └── config.go                 # Configurações (URLs, timeout, porta)
├── internal/
│   ├── app/
│   │   └── app.go                # Inicialização e injeção de dependências
│   ├── controller/
│   │   ├── cep_controller.go     # Handler HTTP e saída no terminal
│   │   └── cep_controller_test.go
│   ├── model/
│   │   └── cep_result.go         # Struct CEPResult (provider + dados JSON brutos)
│   ├── ports/
│   │   └── cep_service.go        # Interface CEPService (porta)
│   ├── router/
│   │   └── router.go             # Definição de rotas com Gin
│   └── service/
│       ├── cep_service.go        # Lógica principal da corrida (goroutines + channel + select)
│       └── cep_service_test.go   # Testes unitários com servidores httptest
├── tests/
│   └── mocks/
│       └── CEPService.go         # Mock da interface CEPService
├── go.mod
└── go.sum
```

### Arquitetura

O projeto segue o estilo **ports and adapters** (hexagonal):

- `ports/` define a interface `CEPService`
- `service/` implementa a lógica de negócio
- `controller/` adapta as requisições HTTP para chamadas ao serviço
- `config/` centraliza todas as configurações, facilitando testes com URLs e timeouts customizados

---

## Pré-requisitos

- [Go 1.21+](https://golang.org/dl/)

---

## Executando

```bash
# Clone o repositório
git clone https://github.com/seu-usuario/cep-api.git
cd cep-api

# Baixe as dependências
go mod tidy

# Inicie o servidor
go run ./cmd/api/main.go
```

O servidor sobe em `http://localhost:8080`.

---

## Testes

```bash
# Executa todos os testes
go test ./...

# Com saída detalhada
go test -v ./...
```

### Cenários cobertos

| Teste | Descrição |
|-------|-----------|
| `BrasilAPIRespondsFirst` | BrasilAPI vence a corrida |
| `ViaCEPRespondsFirst` | ViaCEP vence a corrida |
| `BothTimeout` | Nenhuma API responde a tempo → erro de timeout |
| `OneFailsOtherResponds` | Primeira API retorna erro, segunda responde com sucesso |
| `BothFail` | Ambas as APIs retornam erro → falha geral |

Os testes utilizam servidores `net/http/httptest` para simular cada cenário sem realizar chamadas reais à rede.

---

## Configuração

Todos os valores padrão estão em `config/config.go`:

| Configuração | Padrão |
|--------------|--------|
| URL BrasilAPI | `https://brasilapi.com.br/api/cep/v1/%s` |
| URL ViaCEP | `http://viacep.com.br/ws/%s/json/` |
| Timeout | `1 segundo` |
| Porta | `:8080` |

---

## Tecnologias

- **Linguagem:** Go
- **Framework HTTP:** [Gin](https://github.com/gin-gonic/gin)
- **Concorrência:** Goroutines, Channels, `select`
- **Testes:** `testing`, `net/http/httptest`, [testify](https://github.com/stretchr/testify)
