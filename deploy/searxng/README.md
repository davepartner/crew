# Built-in web search

Crew calls a local SearXNG JSON endpoint. It needs no third-party API key.

```bash
mkdir -p /home/investorpath/searxng
cp deploy/searxng/settings.yml /home/investorpath/searxng/settings.yml
# edit the secret_key
podman run -d --name searxng --restart=always \
  -p 127.0.0.1:8888:8080 \
  -v /home/investorpath/searxng:/etc/searxng:Z \
  docker.io/searxng/searxng:latest
curl 'http://127.0.0.1:8888/search?q=test&format=json'
```

Persist it with `podman generate systemd` or a normal systemd unit.

Set `CREW_SEARCH_URL=http://127.0.0.1:8888/search` (the default). If the
endpoint is unavailable, Crew falls back to Google News, Wikipedia, the company
site and market data.
