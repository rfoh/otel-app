Desenvolver um sistema distribuído em Go composto por dois microsserviços (**Serviço A** e **Serviço B**) que cooperam para consultar o clima de uma cidade baseada no CEP. O diferencial deste desafio é a implementação de **Observabilidade** utilizando **OpenTelemetry (OTEL)** e **Zipkin** para realizar o rastreamento distribuído (Distributed Tracing) das requisições.

# Arquitetura do Sistema 
O sistema é composto por:
1. **Serviço A (Input):** Recebe a requisição do usuário, valida o CEP e encaminha para o Serviço B.
2. **Serviço B (Orquestração):** Recebe o CEP, identifica a cidade, consulta a temperatura e realiza as conversões.
3. **OTEL + Zipkin:** Infraestrutura de coleta e visualização dos traços.

# Requisitos Técnicos: Serviço A (Input)
Este serviço é a porta de entrada. Ele deve ser exposto via HTTP e comunicar-se com o Serviço B.

1. **Endpoint:** Deve aceitar requisições via POST.
2. **Payload de Entrada:** O corpo da requisição deve seguir o formato JSON:
```json
{ "cep": "29902555" }
```
3. **Validação:**

    - O CEP deve ser recebido como **String**.
    - O CEP deve conter exatamente **8 dígitos**.
4. **Comportamento:**

    - **Válido:** Encaminha a requisição para o Serviço B via HTTP. 
    - **Inválido:** Se o CEP não tiver 8 dígitos ou não for string, retornar:

        - **Código HTTP:** 422
        -  **Mensagem:** invalid zipcode

# Requisitos Técnicos: Serviço B (Orquestração)
Este serviço é responsável pela lógica de negócio.

1. **Entrada:** Recebe um CEP válido de 8 dígitos (enviado pelo Serviço A).
2. **Localização:** Consulta uma API externa (como ViaCEP) para obter o nome da cidade.
3. **Clima:** Consulta uma API externa (como WeatherAPI) para obter a temperatura atual da cidade.
4. **Conversão:** Retorna a temperatura formatada em Celsius, Fahrenheit e Kelvin.
5. **Respostas (Output):**

    - **Sucesso (200):** Deve retornar a cidade e as temperaturas formatadas.
    ```json
        { "city": "São Paulo", "temp_C": 28.5, "temp_F": 83.3, "temp_K": 301.65 }
    ```
    - **Erro de Validação (422):** Caso o CEP chegue com formato inválido.
        - Mensagem: invalid zipcode
    - **Não Encontrado (404):** Caso o CEP tenha o formato correto, mas não seja encontrado.
        - Mensagem: can not find zipcode

# Requisitos de Observabilidade (OTEL + Zipkin)
Você deve instrumentar ambos os serviços para garantir o rastreamento completo da requisição.

1. **Tracing Distribuído:** Implemente o tracing de forma que seja possível visualizar no Zipkin o fluxo completo: Request -> Serviço A -> Serviço B.
2. **Spans Específicos:** Além do tracing automático das requisições web, você deve criar **Spans** manuais para medir o tempo de resposta de:

    - Busca de CEP (API externa de localização).
    - Busca de Temperatura (API externa de clima).
3. **Infraestrutura:** Utilize um **OTEL Collector** para receber os dados dos serviços e enviá-los ao Zipkin.

# Dicas e Fórmulas
- APIs Sugeridas:
    - [ViaCEP](https://viacep.com.br/) (Localização)
    - [WeatherAPI](https://www.weatherapi.com/) (Clima)

- **Fórmulas de Conversão:**
    - Celsius para Fahrenheit: F = C * 1.8 + 32
    - Celsius para Kelvin: K = C + 273

# Infraestrutura e Entrega
## Requisitos de Docker

O projeto deve ser totalmente executável via Docker Compose. O arquivo docker-compose.yaml deve subir:
1. Serviço A
2. Serviço B
3. OTEL Collector
4. Zipkin

## Entregável

1. **Código Fonte:** Repositório contendo a implementação dos serviços A e B.
2. **Docker Compose:** Arquivo configurado para rodar todo o ecossistema.
3. **Documentação (README):**
    - Instruções de como realizar a requisição POST no Serviço A.
    - Instruções de como acessar o Zipkin para visualizar os traços.

## Regras de Entrega

1. **Repositório Exclusivo:** O repositório deve conter apenas o projeto em questão.
2. **Branch Principal:** Todo o código deve estar na branch main.