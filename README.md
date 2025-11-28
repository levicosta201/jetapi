# jetapi
[JetAPI](https://www.jetapi.dev)

An API to gather information from [JetPhotos](https://www.JetPhotos.com) and [FlightRadar24](https://www.FlightRadar24.com).

## Documentation
See the [documentation](https://www.jetapi.dev/documentation) for more information regarding the usage of the API.

## Getting Started On Your Own

**Importante:** Antes de executar, certifique-se de atualizar as dependências:

```bash
go mod tidy
```

Depois, você pode executar:

```
make run
```
to build and run the program.

If you do not have `make`,
```
go run ./cmd/jetapi
```
suffices.

### Modo de Desenvolvimento (Debug)

Para ver erros detalhados no console e nas respostas HTTP, execute com a variável de ambiente `DEV_MODE=true`:

```bash
DEV_MODE=true go run ./cmd/jetapi
```

Ou usando make:
```bash
DEV_MODE=true make run
```

**No modo de desenvolvimento:**
- Erros são impressos no console com destaque
- Stack traces completos são mostrados
- Respostas HTTP de erro incluem detalhes do erro
- Logs mais verbosos para facilitar debug

Then one can visit [localhost:8080](http://localhost:8080) to view the documentation and build a query for your local instance.

## Running with Docker

### Pré-requisitos

Antes de começar, certifique-se de ter instalado:
- [Docker](https://www.docker.com/get-started) (versão 20.10 ou superior)
- [Docker Compose](https://docs.docker.com/compose/install/) (versão 1.29 ou superior)

Para verificar se estão instalados:
```bash
docker --version
docker-compose --version
```

### Configuração com arquivo .env

O projeto utiliza um arquivo `.env` para configurar variáveis de ambiente. Este arquivo permite personalizar o host e a porta onde o servidor irá escutar.

**Nota:** O projeto inclui um arquivo `.env.example` como template. Se este arquivo não existir, você pode criá-lo manualmente com o conteúdo abaixo para servir como referência:

```env
# Configuração do JetAPI
# Copie este arquivo para .env e ajuste os valores conforme necessário

# Host onde o servidor irá escutar (padrão: 0.0.0.0)
HOST=0.0.0.0

# Porta onde o servidor irá escutar (padrão: 8080)
PORT=8080

# Configuração do Redis (Cache)
# Se não configurado, o cache será desabilitado
REDIS_HOST=redis
REDIS_PORT=6379
REDIS_PASSWORD=
REDIS_DB=0

# Configuração do S3 (AWS) - Para armazenamento de imagens
S3_BUCKET=alfaero
AWS_REGION=us-east-1
AWS_ACCESS_KEY_ID=
AWS_SECRET_ACCESS_KEY=
S3_BASE_PATH=jetapi

# Configuração do CloudFront (AWS)
# Se não configurado, as URLs de imagens serão retornadas como originais
CLOUDFRONT_DOMAIN=d1234567890.cloudfront.net
CLOUDFRONT_KEY_PAIR_ID=
CLOUDFRONT_PRIVATE_KEY=
CLOUDFRONT_TTL=3600
```

#### Passo 1: Criar o arquivo .env

**Se o arquivo `.env.example` existir**, copie-o para `.env`:

**Linux/Mac:**
```bash
cp .env.example .env
```

**Windows (PowerShell):**
```powershell
Copy-Item .env.example .env
```

**Windows (CMD):**
```cmd
copy .env.example .env
```

**Se o arquivo `.env.example` não existir**, crie o arquivo `.env` manualmente com o seguinte conteúdo:

**Linux/Mac:**
```bash
cat > .env << EOF
HOST=0.0.0.0
PORT=8080
EOF
```

**Windows (PowerShell):**
```powershell
@"
HOST=0.0.0.0
PORT=8080
"@ | Out-File -FilePath .env -Encoding utf8
```

**Windows (CMD):**
```cmd
echo HOST=0.0.0.0 > .env
echo PORT=8080 >> .env
```

#### Passo 2: Configurar as variáveis

Edite o arquivo `.env` criado e ajuste os valores conforme necessário:

```env
# Host onde o servidor irá escutar (padrão: 0.0.0.0)
HOST=0.0.0.0

# Porta onde o servidor irá escutar (padrão: 8080)
PORT=8080
```

**Variáveis disponíveis:**

**Servidor:**
- `HOST`: Endereço IP onde o servidor irá escutar (padrão: `0.0.0.0`)
- `PORT`: Porta onde o servidor irá escutar (padrão: `8080`)
- `DEV_MODE`: Modo de desenvolvimento - mostra erros detalhados (padrão: `false`)
- `ENV`: Ambiente de execução - use `development` para ativar modo dev (padrão: não definido)

**Redis (Cache):**
- `REDIS_HOST`: Endereço do servidor Redis (padrão: `redis` no Docker Compose, `localhost` caso contrário)
- `REDIS_PORT`: Porta do Redis (padrão: `6379`)
- `REDIS_PASSWORD`: Senha do Redis (opcional)
- `REDIS_DB`: Número do banco de dados Redis (padrão: `0`)

**CloudFront (AWS):**
- `CLOUDFRONT_DOMAIN`: Domínio do CloudFront (ex: `d1234567890.cloudfront.net`)
- `CLOUDFRONT_KEY_PAIR_ID`: ID do par de chaves para Signed URLs (opcional)
- `CLOUDFRONT_PRIVATE_KEY`: Chave privada para assinar URLs (opcional, formato PEM)
- `CLOUDFRONT_TTL`: Tempo de expiração das URLs assinadas em segundos (padrão: `3600`)

**S3 (AWS) - Para armazenamento de imagens:**
- `S3_BUCKET`: Nome do bucket S3 (ex: `alfaero`)
- `AWS_REGION`: Região do AWS (ex: `us-east-1`)
- `AWS_ACCESS_KEY_ID`: Chave de acesso AWS
- `AWS_SECRET_ACCESS_KEY`: Chave secreta AWS
- `S3_BASE_PATH`: Caminho base dentro do bucket (padrão: `jetapi`)

**Nota:** 
- Se você não criar o arquivo `.env`, os valores padrão serão utilizados automaticamente.
- O cache (Redis) é opcional. Se não configurado, a API funcionará normalmente sem cache.
- O CloudFront é opcional. Se não configurado, as URLs de imagens serão retornadas como originais.

### Usando Docker Compose (Recomendado)

Docker Compose é a forma mais simples e recomendada de executar a aplicação com Docker.

#### Passo 1: Construir e iniciar o container

Execute o seguinte comando na raiz do projeto:

```bash
docker-compose up -d
```

Este comando irá:
- Construir a imagem Docker (na primeira execução)
- Criar e iniciar o container em modo detached (background)
- Mapear a porta configurada no `.env` (ou 8080 por padrão)

#### Passo 2: Verificar se está funcionando

Acesse a aplicação no navegador:
- **URL padrão:** [http://localhost:8080](http://localhost:8080)
- **URL customizada:** Se você alterou a porta no `.env`, use `http://localhost:PORTA`

#### Passo 3: Gerenciar o container

**Parar o container:**
```bash
docker-compose down
```

**Parar e remover volumes (limpeza completa):**
```bash
docker-compose down -v
```

**Visualizar logs:**
```bash
# Ver todos os logs
docker-compose logs

# Seguir os logs em tempo real
docker-compose logs -f

# Ver apenas as últimas 100 linhas
docker-compose logs --tail=100
```

**Reiniciar o container:**
```bash
docker-compose restart
```

**Reconstruir a imagem (após alterações no código):**
```bash
docker-compose up -d --build
```

**Verificar status:**
```bash
docker-compose ps
```

### Usando Docker diretamente

Se preferir usar Docker sem Docker Compose, siga estes passos:

#### Passo 1: Construir a imagem

Na raiz do projeto, execute:

```bash
docker build -t jetapi .
```

Este comando irá:
- Criar uma imagem Docker chamada `jetapi`
- Usar o Dockerfile para construir a aplicação
- Aplicar build multi-stage para otimizar o tamanho da imagem final

#### Passo 2: Executar o container

**Opção A: Usando arquivo .env (Recomendado)**
```bash
docker run -d -p 8080:8080 --env-file .env --name jetapi jetapi
```

**Opção B: Passando variáveis diretamente**
```bash
docker run -d -p 8080:8080 -e HOST=0.0.0.0 -e PORT=8080 --name jetapi jetapi
```

**Opção C: Porta customizada**
```bash
docker run -d -p 4000:4000 -e HOST=0.0.0.0 -e PORT=4000 --name jetapi jetapi
```

**Explicação dos parâmetros:**
- `-d`: Executa em modo detached (background)
- `-p 8080:8080`: Mapeia a porta do container para a porta do host (formato: `host:container`)
- `--env-file .env`: Carrega variáveis de ambiente do arquivo `.env`
- `-e VAR=valor`: Define variáveis de ambiente individualmente
- `--name jetapi`: Define um nome para o container

#### Passo 3: Gerenciar o container

**Parar o container:**
```bash
docker stop jetapi
```

**Iniciar um container parado:**
```bash
docker start jetapi
```

**Remover o container:**
```bash
docker rm jetapi
```

**Parar e remover em um comando:**
```bash
docker stop jetapi && docker rm jetapi
```

**Visualizar logs:**
```bash
# Ver todos os logs
docker logs jetapi

# Seguir os logs em tempo real
docker logs -f jetapi

# Ver apenas as últimas 100 linhas
docker logs --tail=100 jetapi
```

**Verificar status:**
```bash
docker ps
```

**Remover a imagem:**
```bash
docker rmi jetapi
```

### Troubleshooting

**Problema: Porta já em uso**
```
Error: bind: address already in use
```

**Solução:** Altere a porta no arquivo `.env` ou use uma porta diferente:
```bash
# No .env
PORT=4000

# Ou no comando Docker
docker run -d -p 4000:4000 ...
```

**Problema: Container não inicia**
```bash
# Verifique os logs
docker-compose logs
# ou
docker logs jetapi
```

**Problema: Mudanças no código não aparecem**
```bash
# Reconstrua a imagem
docker-compose up -d --build
# ou
docker build -t jetapi . && docker restart jetapi
```

**Problema: Arquivo .env não é reconhecido**
- Certifique-se de que o arquivo está na raiz do projeto
- Verifique se o nome do arquivo é exatamente `.env` (não `.env.txt` ou similar)
- No Docker Compose, o arquivo `.env` é carregado automaticamente
- No Docker direto, use `--env-file .env`

**Problema: Redis não conecta**
```bash
# Verifique se o container Redis está rodando
docker-compose ps

# Verifique os logs do Redis
docker-compose logs redis

# Teste a conexão manualmente
docker-compose exec redis redis-cli ping
```

**Problema: CloudFront não funciona**
- Verifique se o domínio do CloudFront está correto
- Certifique-se de que o CloudFront está configurado para servir as imagens
- URLs assinadas requerem KeyPairID e PrivateKey válidos

## Cache e Performance

### Sistema de Cache (Redis)

O JetAPI utiliza Redis para cachear resultados de scraping, melhorando significativamente a performance da API. Quando uma requisição é feita:

1. **Primeira requisição**: O sistema faz scraping dos sites e armazena o resultado no Redis (TTL: 24 horas)
2. **Requisições subsequentes**: O resultado é recuperado do cache, retornando instantaneamente

**Benefícios:**
- ⚡ Respostas muito mais rápidas para requisições repetidas
- 🔄 Reduz carga nos sites de origem (JetPhotos e FlightRadar24)
- 💾 Economia de recursos do servidor
- 📊 Melhor experiência para os usuários da API

**Configuração:**
O Redis é automaticamente iniciado quando você usa `docker-compose up`. Para desenvolvimento local sem Docker, você pode instalar Redis separadamente:

```bash
# Ubuntu/Debian
sudo apt-get install redis-server

# macOS
brew install redis

# Iniciar Redis
redis-server
```

### CloudFront (AWS)

O JetAPI suporta integração com Amazon CloudFront para servir imagens através de uma CDN global, proporcionando:

- 🌍 **Distribuição global**: Imagens servidas do edge mais próximo ao usuário
- ⚡ **Performance otimizada**: Redução significativa de latência
- 💰 **Economia de banda**: Reduz custos de transferência
- 🔒 **URLs assinadas**: Suporte a URLs temporárias e seguras (opcional)

**Como funciona:**
1. Quando uma imagem é encontrada durante o scraping, sua URL original é convertida para CloudFront
2. A URL do CloudFront é retornada na resposta da API
3. Os clientes baixam as imagens diretamente do CloudFront (muito mais rápido)

**Configuração do CloudFront:**

1. **Criar uma distribuição CloudFront na AWS:**
   - Acesse o console da AWS CloudFront
   - Crie uma nova distribuição
   - Configure a origem para apontar para o bucket S3 ou servidor onde as imagens estão hospedadas
   - Anote o domínio da distribuição (ex: `d1234567890.cloudfront.net`)

2. **Configurar no .env:**
   ```env
   CLOUDFRONT_DOMAIN=d1234567890.cloudfront.net
   ```

3. **URLs Assinadas (Opcional):**
   Para usar URLs assinadas (mais seguro):
   ```env
   CLOUDFRONT_DOMAIN=d1234567890.cloudfront.net
   CLOUDFRONT_KEY_PAIR_ID=APKAIOSFODNN7EXAMPLE
   CLOUDFRONT_PRIVATE_KEY="-----BEGIN RSA PRIVATE KEY-----\n..."
   CLOUDFRONT_TTL=3600
   ```

**Nota:** O CloudFront é totalmente opcional. Se não configurado, as URLs originais das imagens serão retornadas.

## More
The API works best with commercial airliners. 
GA aircraft may cause JSON encoding errors due to the variability in FlightRadar's page. 
Registrations not found also return JSON encoding errors.

You may also specify the port and host by setting the `PORT` and `HOST` environment variables respectively:
```
HOST=0.0.0.0 PORT=4000 make run
```
will serve to `0.0.0.0:4000`.
