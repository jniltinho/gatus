## REMOVED Requirements

### Requirement: Períodos fora do ar no gráfico
**Reason**: o gráfico passa a seguir o formato do Uptime Kuma, com colunas por resultado ou por agregado no lugar das faixas contínuas calculadas pelos eventos e pelos resultados.
**Migration**: as quedas e os Pending aparecem como colunas vermelhas e amarelas do requisito "Gráfico no formato do Uptime Kuma" da capability `response-time-chart`; o início e a duração das quedas continuam na lista de eventos.
