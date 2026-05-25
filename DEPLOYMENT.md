# Deployment (Cloud Run)

```bash
gcloud run deploy evidencevault \
  --source . \
  --region us-central1 \
  --allow-unauthenticated \
  --set-env-vars APP_ENV=production
```
