# goexpert-desafio5-stresstest

CLI em Go para realizar testes de carga em serviços web. Você informa a URL, o número total de requisições e o nível de concorrência; ao final é exibido um relatório com o tempo total, a quantidade de requests, quantos retornaram HTTP 200 e a distribuição dos demais status.

## Parâmetros

| Flag            | Obrigatório | Padrão | Descrição                              |
|-----------------|-------------|--------|----------------------------------------|
| `--url`         | sim         | –      | URL do serviço a ser testado           |
| `--requests`    | sim         | –      | Número total de requisições            |
| `--concurrency` | não         | `1`    | Número de chamadas simultâneas         |
| `--timeout`     | não         | `30s`  | Timeout de cada requisição (ex: `5s`)  |

## Executando com Docker

### 1. Buildar a imagem

Na raiz do repositório:

```bash
docker build -t stresstest .
```

### 2. Executar o teste

```bash
docker run --rm stresstest --url=http://google.com --requests=1000 --concurrency=10
```

> Para testar um serviço rodando na sua própria máquina, use `host.docker.internal` no lugar de `localhost`
> (ex.: `--url=http://host.docker.internal:8080`). No Linux, adicione `--add-host=host.docker.internal:host-gateway` ao `docker run`.

### Exemplo de saída

```
Iniciando teste: 1000 requests, concorrência 10 -> http://google.com

==================================================
              RELATÓRIO DO TESTE DE CARGA
==================================================
URL:                         http://google.com
Tempo total:                 1m11.264s
Total de requests:           1000
Requests com status 200:     862
Requests por segundo:        14.03
==================================================
Distribuição de outros status HTTP:
  301 Moved Permanently        4
  302 Found                    134
==================================================
```

Requisições que falham sem obter resposta HTTP (timeout, conexão recusada, DNS etc.) também entram no total de requests e aparecem numa seção separada, **Falhas sem resposta HTTP**.

## Executando sem Docker

Requer Go 1.26+.

```bash
go run . --url=http://google.com --requests=100 --concurrency=10
```

Testes:

```bash
go test ./...
```

## Como funciona

- São criados `min(concurrency, requests)` workers, que consomem uma fila com exatamente `--requests` itens. Assim, o total de requisições é cumprido exatamente e nunca há mais que `--concurrency` requisições simultâneas.
- As conexões são reaproveitadas (keep-alive) e o corpo de cada resposta é lido e descartado.
- Redirecionamentos são seguidos (até 10). Em caso de loop de redirecionamento, é contabilizado o status da última resposta recebida (ex.: `302`).
- `Ctrl+C` interrompe o teste; as requisições pendentes são contabilizadas como `cancelado`.

## Estrutura

```
.
├── main.go                         # parsing das flags e execução
├── internal/stresstest/
│   ├── runner.go                   # pool de workers e coleta dos resultados
│   ├── report.go                   # formatação do relatório
│   └── runner_test.go
└── Dockerfile                      # build multi-stage, imagem final baseada em scratch
```
