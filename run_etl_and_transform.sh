#!/bin/bash

# --- 1. CONFIGURAÇÃO ---
# Certifique-se que o BQ CLI esteja disponível no ambiente
# (No Cloud Run ele usa a conta de serviço, o que simplifica a autenticação)
PROJECT_ID="worlddata-439415"

# --- 2. EXECUÇÃO DO ETL (Carga de Dados Brutos) ---
echo "Iniciando Carga do Instagram (Go Binário)..."
# O binário Go compila e roda todos os fluxos: Perfil, Posts, Stories
/main 

# Verifica o status de saída do binário Go
if [ $? -ne 0 ]; then
    echo "ERRO FATAL: O binário Go falhou. Abortando Transformação SQL."
    exit 1
fi

echo "Carga concluída com sucesso. Iniciando Transformação SQL..."

# --- 3. EXECUÇÃO DA TRANSFORMAÇÃO SQL (Consolidação de Dados) ---

# Função auxiliar para rodar o SQL e verificar o erro
run_sql_query() {
    local sql_file=$1
    echo "Executando $sql_file..."
    
    # Roda a query no BigQuery usando o comando 'bq'
    bq --project_id=${PROJECT_ID} query --use_legacy_sql=false < $sql_file

    if [ $? -ne 0 ]; then
        echo "ERRO: A query em $sql_file falhou. Abortando."
        exit 1
    fi
}

# Roda os 3 scripts de transformação na ordem
run_sql_query "sql/perfil_transform.sql"
run_sql_query "sql/postagens_transform.sql"
run_sql_query "sql/stories_transform.sql"

echo "Processo ETL e Transformação SQL concluído com sucesso."