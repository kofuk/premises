# Installing Premises

1. Create your `.env` file (please refer to `.env.example` for keys and description).
2. Populate database schema
```shell
$ docker compose run --rm web /premises migrate
```
3. Run `docker compose up` and the server will listen on `:8000`.
4. Add user by the following command
```shell
$ docker compose exec web pmctl user add -u "${user}" -p "${password}"
```

# Updating Premises

1. Stop running services
```shell
$ docker compose down web proxy
```
2. Update .env if needed
3. Run schema migration
```shell
$ docker compose run --rm web /premises migrate
```
4. Start services
```shell
$ docker compose up -d
```
