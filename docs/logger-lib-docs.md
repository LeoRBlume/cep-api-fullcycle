# Logger Library — `pkg/logger`

> Biblioteca de logs estruturados em Go, baseada em `slog` (stdlib), com suporte a contexto obrigatório, middleware HTTP e saída JSON.

---

## Visão Geral

| Atributo | Decisão |
|---|---|
| Backend | `slog` (stdlib Go 1.21+) |
| Formato | JSON, stdout fixo |
| Contexto | Obrigatório em todas as chamadas |
| Níveis | DEBUG, INFO, WARN, ERROR, FATAL |
| Campos automáticos | `timestamp`, `level`, `service`, `src`, `trace_id`, `user_id` |
| Fatal | Loga + `os.Exit(1)` |
| Testes | `NewNop()` — handler que descarta output |
| Dependência externa | `github.com/google/uuid` (geração de UUID no middleware) |

---

## Estrutura de Arquivos

```
pkg/logger/
  options.go     # Config e Level
  logger.go      # Global instance + funções públicas
  context.go     # WithTraceID, WithUserID, extração do ctx
  middleware.go  # TraceMiddleware (http.Handler)
  nop.go         # NewNop() para testes
```

---

## User Stories

---

### US-01 — Configuração da lib

**Como** desenvolvedor,  
**quero** inicializar o logger uma única vez no `main`,  
**para** que toda a aplicação use a mesma configuração sem precisar passar instâncias.

#### Critérios de Aceite

- `logger.Setup(Config{})` deve ser chamável uma vez na inicialização da aplicação
- `Config` deve aceitar `ServiceName` e `Level`
- Se `Setup` não for chamado, o logger deve funcionar com valores default (`INFO`, service `"app"`)
- Chamadas subsequentes a `Setup` devem sobrescrever a configuração anterior
- O `ServiceName` deve aparecer em todos os logs como campo `"service"`

#### Exemplo de Uso

```go
logger.Setup(logger.Config{
    ServiceName: "payments-api",
    Level:       logger.LevelDebug,
})
```

---

### US-02 — Log sem formatação (Info, Debug, Warn, Error, Fatal)

**Como** desenvolvedor,  
**quero** logar mensagens simples passando contexto, origem e mensagem,  
**para** ter rastreabilidade sem verbosidade desnecessária.

#### Critérios de Aceite

- Funções disponíveis: `Debug`, `Info`, `Warn`, `Error`, `Fatal`
- Todas recebem `(ctx context.Context, src string, msg string)`
- `Error` e `Fatal` recebem adicionalmente `err error`
- O campo `src` deve aparecer no JSON como `"src"`
- `Fatal` deve chamar `os.Exit(1)` após logar
- Logs abaixo do nível configurado devem ser silenciados
- O output deve ser JSON no stdout

#### Exemplo de Uso

```go
logger.Info(ctx, "UserService.Create", "usuário criado")
logger.Warn(ctx, "UserService.Create", "tentativa duplicada")
logger.Error(ctx, "UserService.Create", "falha ao salvar", err)
logger.Fatal(ctx, "main", "banco indisponível", err)
```

#### Output JSON Esperado

```json
{
  "timestamp": "2026-03-26T14:00:00-03:00",
  "level": "ERROR",
  "service": "payments-api",
  "src": "UserService.Create",
  "trace_id": "abc-123",
  "user_id": "usr-456",
  "message": "falha ao salvar",
  "error": "duplicate key violation"
}
```

---

### US-03 — Log com formatação (Infof, Debugf, Warnf, Errorf, Fatalf)

**Como** desenvolvedor,  
**quero** logar mensagens com interpolação de variáveis no estilo `fmt.Sprintf`,  
**para** construir mensagens dinâmicas sem concatenação manual de strings.

#### Critérios de Aceite

- Funções disponíveis: `Debugf`, `Infof`, `Warnf`, `Errorf`, `Fatalf`
- `Debugf`, `Infof`, `Warnf` recebem `(ctx, src, msg string, args ...any)`
- `Errorf` e `Fatalf` recebem `(ctx, src, msg string, err error, args ...any)`
- A mensagem final deve ser formatada via `fmt.Sprintf(msg, args...)`
- `Fatalf` deve chamar `os.Exit(1)` após logar
- O campo `error` deve conter `err.Error()` como string

#### Exemplo de Uso

```go
logger.Infof(ctx, "UserService.Create", "usuário %s criado", userName)
logger.Errorf(ctx, "UserService.Create", "falha ao salvar usuário %s", err, userName)
logger.Fatalf(ctx, "main", "porta %d indisponível", err, port)
```

---

### US-04 — Enriquecimento de contexto

**Como** desenvolvedor,  
**quero** injetar `trace_id` e `user_id` no contexto,  
**para** que esses campos apareçam automaticamente em todos os logs daquele fluxo.

#### Critérios de Aceite

- `logger.WithTraceID(ctx, id)` retorna um novo contexto com o `trace_id` injetado
- `logger.WithUserID(ctx, id)` retorna um novo contexto com o `user_id` injetado
- Ambos os campos devem aparecer automaticamente em qualquer log que receba esse contexto
- As chaves de contexto devem ser privadas (unexported) para evitar colisão com outros pacotes
- Em caso de colisão, os valores do contexto têm prioridade sobre qualquer outro campo
- Se o contexto não tiver `trace_id` ou `user_id`, os campos devem ser omitidos do JSON

#### Exemplo de Uso

```go
ctx = logger.WithTraceID(ctx, "abc-123")
ctx = logger.WithUserID(ctx, "usr-456")

logger.Info(ctx, "OrderService.Create", "pedido criado")
// → inclui "trace_id": "abc-123" e "user_id": "usr-456" automaticamente
```

---

### US-05 — Middleware HTTP de rastreabilidade

**Como** desenvolvedor,  
**quero** um middleware HTTP que injete automaticamente um `trace_id` no contexto de cada requisição,  
**para** rastrear todas as chamadas de uma request sem instrumentação manual.

#### Critérios de Aceite

- O middleware deve ler o header `X-Correlation-ID` da requisição
- Se o header existir, usa o valor recebido como `trace_id`
- Se não existir, gera um UUID v4 aleatório
- O `trace_id` deve ser injetado no contexto via `WithTraceID`
- O header `X-Correlation-ID` deve ser devolvido na response com o mesmo valor
- O middleware deve ser compatível com `http.Handler` padrão do Go
- Deve ser utilizável com qualquer router que aceite `http.Handler`

#### Exemplo de Uso

```go
mux := http.NewServeMux()
mux.Handle("/", logger.TraceMiddleware(mux))

// Request com header → reutiliza
// GET /users  X-Correlation-ID: abc-123
// → ctx terá trace_id = "abc-123"
// → response terá X-Correlation-ID: abc-123

// Request sem header → gera UUID
// GET /users  (sem header)
// → ctx terá trace_id = "f47ac10b-58cc-4372-a567-0e02b2c3d479"
// → response terá X-Correlation-ID: f47ac10b-58cc-4372-a567-0e02b2c3d479
```

---

### US-06 — Logger silencioso para testes

**Como** desenvolvedor,  
**quero** um logger que descarte todo output durante testes unitários,  
**para** não poluir o output do `go test` com logs irrelevantes.

#### Critérios de Aceite

- `logger.NewNop()` deve retornar um `*Logger` que descarta todo output
- O Nop deve respeitar a mesma interface das funções públicas
- Não deve haver nenhuma escrita em stdout ou stderr ao usar o Nop
- Deve ser usável para substituir o logger global em testes

#### Exemplo de Uso

```go
func TestUserService(t *testing.T) {
    log := logger.NewNop()
    svc := NewUserService(log)
    // nenhum log aparece no output do teste
}
```

---

## Output JSON — Referência Completa

### Campos fixos (sempre presentes)

| Campo | Tipo | Origem |
|---|---|---|
| `timestamp` | string (RFC3339 com timezone) | gerado automaticamente |
| `level` | string | nível da chamada |
| `service` | string | `Config.ServiceName` |
| `src` | string | parâmetro `src` da chamada |
| `message` | string | parâmetro `msg` (formatado se `f`) |

### Campos de contexto (presentes se injetados)

| Campo | Tipo | Origem |
|---|---|---|
| `trace_id` | string | `WithTraceID` ou middleware |
| `user_id` | string | `WithUserID` |

### Campos condicionais

| Campo | Tipo | Presente quando |
|---|---|---|
| `error` | string | chamadas `Error`, `Errorf`, `Fatal`, `Fatalf` |

---

## Regras de Precedência de Campos

Em caso de colisão de chaves:

```
contexto  >  args explícitos  >  qualquer outro campo
```

---

## Dependências

| Pacote | Uso |
|---|---|
| `log/slog` | Backend de logging (stdlib Go 1.21+) |
| `context` | Propagação de valores (stdlib) |
| `fmt` | Formatação das mensagens `f` (stdlib) |
| `os` | `os.Exit(1)` no Fatal (stdlib) |
| `net/http` | Middleware HTTP (stdlib) |
| `github.com/google/uuid` | Geração de UUID v4 no middleware |
