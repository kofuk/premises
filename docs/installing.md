# Installing Premises

1. Create your `.env` file (please refer to `.env.example` for keys and description).
2. Run `docker compose up` and the server will listen on `:8000`.
3. Add user by the following command
```shell
$ docker compose exec web pmctl user add -u "${user}" -p "${password}"
```
