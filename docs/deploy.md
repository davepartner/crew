# Deploying Crew

Crew is one static binary with the UI embedded. It needs a writable directory for SQLite and
something in front of it that terminates TLS.

## Build

```bash
make build                                   # local
GOOS=linux GOARCH=amd64 CGO_ENABLED=0 \
  go build -trimpath -ldflags "-s -w" -o crew ./cmd/crew
```

`make build` runs the UI build first and copies `web/dist` into `internal/webui/dist`, which is
what gets embedded.

## Server layout

```
/home/investorpath/crew/
  crew                 # binary
  crew.env             # 0600, secrets
  data/crew.db         # SQLite, WAL, owned by the service user
```

`crew.env`:

```dotenv
CREW_ADDR=127.0.0.1:8790
CREW_DB=/home/investorpath/crew/data/crew.db
PAYTRACKIFY_URL=https://paytrackify.com
CREW_ENV=production
CREW_SECURE_COOKIES=true
CREW_TRUST_PROXY=true
CREW_SECRET_KEY=<openssl rand -hex 32>
```

`CREW_SECRET_KEY` encrypts provider API keys. If it is missing, Crew creates a `0600` file
beside the database — fine locally, not what you want on a shared host.

## systemd

```bash
sudo cp deploy/crew.service /etc/systemd/system/crew.service
sudo systemctl daemon-reload
sudo systemctl enable --now crew
curl -s http://127.0.0.1:8790/api/health
```

Logs: `journalctl -u crew -f`.

## Reverse proxy (cPanel / Apache)

Create the subdomain, then add the same include to both the plain and TLS vhosts:

`/etc/apache2/conf.d/userdata/{std,ssl}/2_4/<user>/<domain>/proxy.conf`

```apache
ProxyPreserveHost On
ProxyPass "/" "http://127.0.0.1:8790/" retry=0
ProxyPassReverse "/" "http://127.0.0.1:8790/"
```

In the `std` (port 80) include, replace the proxy lines with a redirect so cookies never travel
over plain HTTP:

```apache
Redirect permanent / https://<domain>/
```

```bash
/scripts/rebuildhttpdconf
httpd -t && systemctl reload httpd
```

TLS comes from cPanel AutoSSL. A wildcard `*.example.com` certificate covers the subdomain.

## Updating

```bash
GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -trimpath -ldflags "-s -w" -o crew ./cmd/crew
scp crew user@host:/home/user/crew/crew.new
ssh host 'mv /home/user/crew/crew.new /home/user/crew/crew && systemctl restart crew'
```

The SQLite schema migrates itself on start; take a copy of `data/crew.db` first.

## The other deployment: paytrackify.com

The Laravel app has its own one-command deploy:

```bash
pull-paytrackify              # pull, composer, npm build, migrate, cache, queue restart
pull-paytrackify --no-frontend
```

It uses `/usr/local/bin/php` explicitly, backs up `public/build` before a rebuild, and restores
it if the build fails.
