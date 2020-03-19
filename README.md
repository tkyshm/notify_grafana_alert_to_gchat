# Cloud Function to alert notification to Hangouts Chat from Grafana alert notification.

## deploy

```
$ gcloud functions deploy notify_grafana_alert_to_gchat \
    --entry-point NotifyGrafanaAlertToGChat \
    --runtime go111 \
    --set-env-vars 'WEBHOOK_URL=...' \
    --trigger-http \
    --project ... \
    --region ...
```
