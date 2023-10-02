# Development Guide

## Running Locally

Since I develop Premises on Linux computer, support for Mac OS and Windows and so on is not tested (or is not implemented).
If you are a Mac Os or Windows user, and want to run Premises locally, I strongly recommend to use Linux VM or WSL.

Currently, we can't run storage component (saving game data on Mega) locally.
In other words, components other than storage are working locally.
(We have a plan to replace storage from Mega to another. After that, storage component should work locally.)

1. Launch PostgreSQL and Redis using Docker Compose in /dev directory.
```shell
$ docker compose up -d
```
2. Build fake runner image in /dev directory (this required only once).
```shell
$ docker build -t premises.kofuk.org/dev-runner -f Dockerfile.runner \
    --label org.kofuk.premises.managed=true \
    --label org.kofuk.premises.id=$(uuidgen) \
    --label org.kofuk.premises.name=mc-premises .
```
3. Launch fake OpenStack in /ostack-fake directory.
```shell
$ cd ../ostack-fake
$ go run .
```
4. Create /controlpanel/.env by copying /controlpanel/.env.example
```ini
premises_debug_web='true'
premises_conoha_username='user'
premises_conoha_password='password'
premises_conoha_tenantId='tenantId'
premises_conoha_services_identity='http://localhost:8010/identity/v2.0'
premises_conoha_services_image='http://localhost:8010/image'
premises_conoha_services_compute='http://localhost:8010/compute/v2'
premises_conoha_nameTag='mc-premises'
premises_cloudflare_token=''
premises_cloudflare_zoneId=''
premises_cloudflare_gameDomain=''
premises_mega_email='<Your Mega account email>'
premises_mega_password='<Your Mega account password>'
premises_mega_folderName='worlds.dev'
premises_game_motd=''
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
premises_controlPanel_locale='ja'
```
5. Build frontend in /controlpanel directory.
```shell
$ npm install
$ npm run dev
```
6. Launch Control Panel server in /controlpanel directory.
```shell
$ go run .
```

### Q&A

#### Can I use mcmanager or exteriord I'm developing locally?

Currently, you can on Linux (and hopefully on Mac OS), but you can't on Windows.

On the supported platforms, running the following commands will deploy your mcmanager and exteriord on the next launch of runner.

```shell
$ cd exteriord
$ make deploy-dev

$ cd mcmanager
$ make deploy-dev
```

#### Where is my game data saved?

On Linux, it is saved to /tmp/premises-data on your computer, but on the other platforms, it is inside your Docker image.

Therefore, Docker image size will become significantly large on these platforms.
