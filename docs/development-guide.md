# Development Guide

## Running Locally

Since I develop Premises on Linux computer, support for Mac OS and Windows and so on is not tested (or is not implemented).
If you are a Mac Os or Windows user, and want to run Premises locally, I strongly recommend to use Linux VM or WSL.

1. Build fake runner image in /dev directory (this is required only once).
```shell
$ docker build -t premises.kofuk.org/dev-runner -f Dockerfile.runner \
    --label org.kofuk.premises.managed=true \
    --label org.kofuk.premises.id=$(uuidgen) \
    --label org.kofuk.premises.name=mc-premises .
```
2. Create /controlpanel/.env by copying /controlpanel/.env.example
```ini
premises_debug_web='true'
premises_conoha_username='user'
premises_conoha_password='password'
premises_conoha_tenantId='tenantId'
premises_conoha_services_identity='http://localhost:8010/identity/v3'
premises_conoha_services_compute='http://localhost:8010/compute/v2'
premises_conoha_services_network='http://localhost:8010/network'
premises_conoha_services_volume='http://localhost:8010/volume'
premises_conoha_nameTag='mc-premises'
# s3.premises.local is magic URL to test with local MinIO.
premises_s3_endpoint='http://s3.premises.local:9000'
premises_s3_bucket='premises'
premises_aws_accessKey='premises'
premises_aws_secretKey='password'
premises_game_operators='<Your Minecraft Username>'
premises_game_whitelist='<Your Minecraft Username>'
premises_controlPanel_secret='secret'
premises_controlPanel_origin='http://localhost:8000'
premises_controlPanel_redis_address='localhost:6379'
premises_controlPanel_redis_password=''
premises_controlPanel_postgres_address='localhost'
premises_controlPanel_postgres_port=5432
premises_controlPanel_postgres_user='premises'
premises_controlPanel_postgres_password='password'
premises_controlPanel_postgres_dbName='premises'
premises_controlPanel_gameDomain='localhost'
```
3. Launch all dependencies using helper script
```shell
$ ./dev/launch_all.sh
```

## Run ControlPanel locally, runner on cloud

There are many solution to expose our local ports on the web.
We will demonstrate it with Cloudflare Tunnel.

First, create tunnels by following the official instruction.
https://developers.cloudflare.com/cloudflare-one/connections/connect-networks/get-started/create-local-tunnel/

we need to have at least 2 domains:

- Control Panel (we use `premises` subdomain)
- MinIO's S3 API (we use `s3api` subdomain)
- (Optional) MinIO console (we use `s3` subdomain)

Then, create configuration file like this:

```yaml
tunnel: ${tunnel_id}
credentials-file: /home/$(whoami)/.cloudflared/${tunnel_id}.json
ingress:
  - hostname: premises.${domain}
    service: http://localhost:8000
  - hostname: s3api.${domain}
    service: http://localhost:9000
  - hostname: s3.${domain}
    service: http://localhost:9001
  - service: http_status:404
```

After that, you can expose your control panel globally.

```shell
cloudflared tunnel run ${tunnel_name}
```

### Q&A

#### Can I use runner I'm developing locally?

Currently, you can on Linux (and hopefully on Mac OS), but you can't on Windows.

On the supported platforms, running the following commands will deploy your runner on the next launch of runner.

```shell
$ cd runner
$ make deploy-dev
```

#### Where is my game data saved?

On Linux, it is saved to /tmp/premises-data on your computer, but on the other platforms, it is inside your Docker image.

Therefore, Docker image size will become significantly large on these platforms.
