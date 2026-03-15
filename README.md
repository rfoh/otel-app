# Instruções de como realizar a requisição POST no Serviço A.
- Faça uma requisição HTTP POST para localhost na porta 8080 e em /zipcode. No body, insira um json contendo o parâmetro "cep".
- Exemplo de chamada com o curl:
```bash
curl --request POST \
  --url http://localhost:8080/zipcode \
  --data '{ "cep": "57052710" }'
```

# Instruções de como acessar o Zipkin para visualizar os traços.
- Para acessar o Zipkin: http://localhost:9411/zipkin/