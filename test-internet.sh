#!/bin/bash
# Script para testar conectividade de internet do container Docker

echo "Testando conectividade de internet do container jetapi..."

# Verificar se o container está rodando
if ! docker ps | grep -q jetapi; then
    echo "ERRO: Container jetapi não está rodando!"
    echo "Execute: docker-compose up -d"
    exit 1
fi

echo ""
echo "1. Testando resolução DNS (google.com)..."
docker exec jetapi nslookup google.com || echo "AVISO: nslookup não disponível, tentando ping..."

echo ""
echo "2. Testando ping para 8.8.8.8 (Google DNS)..."
docker exec jetapi ping -c 3 8.8.8.8 || echo "AVISO: ping não disponível"

echo ""
echo "3. Testando conexão HTTP (httpbin.org)..."
docker exec jetapi wget -q --spider --timeout=5 http://httpbin.org/get && echo "✓ Conexão HTTP funcionando!" || echo "✗ Falha na conexão HTTP"

echo ""
echo "4. Testando conexão HTTPS (www.google.com)..."
docker exec jetapi wget -q --spider --timeout=5 https://www.google.com && echo "✓ Conexão HTTPS funcionando!" || echo "✗ Falha na conexão HTTPS"

echo ""
echo "5. Testando acesso ao JetPhotos..."
docker exec jetapi wget -q --spider --timeout=10 https://www.jetphotos.com && echo "✓ Acesso ao JetPhotos funcionando!" || echo "✗ Falha no acesso ao JetPhotos"

echo ""
echo "Teste concluído!"

