# Premises (Control Server)

"Premises" is a software to build on-demand Minecraft cloud server.

Premises consists of the following 2 parts:

1. Frontend that determine configuration of the server, and creates VM.
2. Software to actually configure game and monitor that the server alive.

## Set Up

1. Create your `.env` file (please refer to `.env.example` for keys and description).
2. Run `docker compose up` and the server will listen on `:8000`.
3. Add user by the following command
```shell
$ docker compose exec web pmctl user add -u "${user}" -p "${password}"
```

## World Archives

Premises always stores world archives to Mega in the form of Zstandard compressed tar archive,
but it can recognize `.tar.xz`, `.zip` files in addition to `.tar.zst`.

Each archives should have the following files or directries in the root:

Name            | Type | Required | Description
----------------|------|----------|----------------------------------------------------------
world           | Dir  | Yes      | World data loaded by vanilla Minecraft server
world\_nether   | Dir  | No       | World data for nether loaded by spigot Minecraft server
world\_the\_end | Dir  | No       | World data for the end loaded by spigot Minecraft server
