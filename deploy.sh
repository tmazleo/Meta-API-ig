
PROJECT_ID="worlddata-439415"
SERVICE_NAME="instagram-etl-job"
REGION="us-central1"

echo "Iniciando Build do Docker no Google Cloud Build..."
IMAGE_URI="gcr.io/${PROJECT_ID}/${SERVICE_NAME}"

gcloud builds submit --tag "${IMAGE_URI}"

if [ $? -ne 0 ]; then
    echo "Falha no Cloud Build. Verifique os logs."
    exit 1
fi

echo "Iniciando Deployment no Cloud Run..."

gcloud run deploy "${SERVICE_NAME}" \
    --image "${IMAGE_URI}" \
    --region "${REGION}" \
    --platform managed \
    --no-allow-unauthenticated \
    --set-env-vars PROJECT_ID="${PROJECT_ID}",API_TOKEN="SEU_TOKEN_LONGA_DURACAO",USER_ID="SEU_IG_ID" \
    --service-account="SEU_SERVICE_ACCOUNT_EMAIL" \
    --command "./run_etl_and_transform.sh" \
    --cpu "1" \
    --memory "512Mi" \
    --max-instances "1" 

if [ $? -ne 0 ]; then
    echo "Falha no Deployment do Cloud Run. Verifique as configurações."
    exit 1
fi

echo "Deployment concluído com sucesso. Serviço: ${SERVICE_NAME} em ${REGION}."