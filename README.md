# Kubernets event processor

## Configure Elasticsearch

```bash
PUT _template/k8s-events
{CONTENT OF resources/elasticsearch.template.json}
```