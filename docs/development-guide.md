# Development Guide

## Running Locally

1. Open editor in Dev Container
```shell
$ devcontainer open
```
2. Run all services in the container
```shell
$ ./launch_all.sh
```

Initial user's name and password is `admin/password`.

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
