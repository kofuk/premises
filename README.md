# Premises

"Premises" is a software to build on-demand Minecraft server on cloud.

Premises consists of the following 2 parts:

1. Frontend that determine configuration of the server, and creates VM. (Control Panel)
2. Software to actually configure game and monitor that the server alive. (Runner)

## Features

- Launch Minecraft server from Web.
- Save world data on S3 and select world to load.
- Monitor server status in real time.
- Monitor CPU usage of server in real time.
- Instantly take snapshot of running world and restore it.

## Deploying

### Kubernetes

You can deploy Premises to Kubernetes using the Helm chart.
You will still need to prepare environment variables to configure PostgreSQL, Valkey and Premises.
See the [test manifest](charts/premises/test) for an example.

```shell
$ helm upgrade --install premises premises \
    --repo https://premises.kofuk.org/charts
```

This Helm chart will make use of LoadBalancer service to expose Minecraft and proxy backend port
to the internet. Please make sure your Kubernetes cluster supports LoadBalancer service type.

### Docker Compose

Deploying to Kubernetes is the recommended method, but legacy methods using Docker Compose can also be used.
See [documentation](docs/installing.md) for step to setup Premises on your server.

## License

MIT
