# Premises (Control Server)

"Premises" is a software to build on-demand Minecraft cloud server.

Premises consists of the following 2 parts:

1. Frontend that determine configuration of the server, and creates VM.
2. Software to actually configure game and monitor that the server alive.

## Set Up

1. Create your `.env` file (please refer to `.env.example` for keys and description).
2. Run `docker compose up` and the server will listen on `:8000`.
