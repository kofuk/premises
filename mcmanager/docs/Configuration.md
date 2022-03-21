# Configuration

## External dependencies

This software may execute the following command.

- `java`: To launch Minecraft server.
- `tar`: To create and/or extract world archive.
- `xz`: When create and/or extract world archive, `tar` command may use this command.

Especially, following points are tricky. We recommend you to check manually.
- Old `java` command may not able to load newer version of `server.jar`.
- Old `tar` command don't have `-J` option (use `xz` to compress archives).

## Config file

premises-mcmanager reads config file from `/opt/premises/config.json`.

This pages describes configuration spec.

- `removeMe`(bool): Remove config file after loading config file.
- `allocSize`(int): Java heap size (MiB).
- `cookie`(string): Password to stop the server.
- `serverName`(string): [Server name](#Servers) to launch.
- `world`
    - `archiveVersion`(string): Archive generation to use. Set `latest` to use latest data.
    - `migrateFromServer`(string): Migrate world data from selected server name's data.
- `motd`(string): Server's motd.
- `worldType`(string): World type to generate. `DEFAULT` or `FLAT`.
- `operators`(array of string): Operators' usernames.
- `whitelist`(array of string): Whitelisted usernames. Whitelist is enabled by default.
- `difficulty`(string): Game's difficulty.
- `mega`
    - `email`(string): Email address to use login to Mega.
    - `password`(string): Password to login to Mega.

## Servers

premises-mcmanager would work with multiple configurations of Minecraft.
To add configuration, put `server.jar` to `/opt/premises/servers.d/<server name>/server.jar`
where `<server name>` is the neme you would like to use to specify this configuration.

premises-mcmanager comes with partial support for mod server.
Only [PaperMC](https://papermc.io/) is considered.

## Environment prefix

If you wan't premises to use actual `/opt` directory, you can make premises to treat
specific directory as root.

Currently, you can set your own root directory by passing the file path as command line argument.
This may be changed in the future.
