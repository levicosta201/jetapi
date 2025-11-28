# Build stage
FROM golang:1.23-alpine AS builder

# Instalar dependências de build
RUN apk add --no-cache git

# Definir diretório de trabalho
WORKDIR /app

# Copiar arquivos de dependências
COPY go.mod go.sum ./

# Baixar dependências
RUN go mod download

# Copiar código fonte
COPY . .

# Compilar a aplicação
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o /app/bin/jetapi ./cmd/jetapi

# Runtime stage
FROM alpine:latest

# Instalar ca-certificates e wget para healthcheck
RUN apk --no-cache add ca-certificates wget

WORKDIR /root/

# Copiar o binário compilado do stage de build
COPY --from=builder /app/bin/jetapi .

# Copiar arquivos estáticos e templates necessários
COPY --from=builder /app/ui ./ui

# Expor a porta padrão
EXPOSE 8080

# Variáveis de ambiente padrão
ENV HOST=0.0.0.0
ENV PORT=8080

# Comando para executar a aplicação
CMD ["./jetapi"]

