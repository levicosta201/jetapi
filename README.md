# jetapi
[JetAPI](https://www.jetapi.dev)

An API to gather information from [JetPhotos](https://www.JetPhotos.com) and [FlightRadar24](https://www.FlightRadar24.com).

## Documentation
See the [documentation](https://www.jetapi.dev/documentation) for more information regarding the usage of the API.

## Getting Started On Your Own
After cloning the project, one can simply run
```
make run
```
to build and run the program.

If you do not have `make`,
```
go run ./cmd/jetapi
```
suffices.

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
- `HOST`: Endereço IP onde o servidor irá escutar (padrão: `0.0.0.0`)
- `PORT`: Porta onde o servidor irá escutar (padrão: `8080`)

**Nota:** Se você não criar o arquivo `.env`, os valores padrão (HOST=0.0.0.0, PORT=8080) serão utilizados automaticamente.

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

## More
The API works best with commercial airliners. 
GA aircraft may cause JSON encoding errors due to the variability in FlightRadar's page. 
Registrations not found also return JSON encoding errors.

You may also specify the port and host by setting the `PORT` and `HOST` environment variables respectively:
```
HOST=0.0.0.0 PORT=4000 make run
```
will serve to `0.0.0.0:4000`.
