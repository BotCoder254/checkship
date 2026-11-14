# postgres-pgadmin example

PostgreSQL 18 with pgAdmin 4 in front of it, deployed by checkship on a local Docker target.

- `db`: Postgres, data in the `pgdata` volume, exposed on a TCP entrypoint (`5432/tcp`) so you can reach it from the host. checkship picks the host port the first time and keeps it.
- `pgadmin`: the web admin UI, served by the target proxy on the app's default url (`http://postgres.docker.localhost`). The `db` server is already registered.

Passwords are not in the compose file: they are the app's per-service variables in checkship (`db.POSTGRES_PASSWORD`, `pgadmin.PGADMIN_DEFAULT_*`).

## Deploy

```sh
CHECKSHIP_EMAIL=admin@example.com ./deploy.sh   # asks for your checkship password
```

The script creates the local target and the `postgres` app when they are missing, generates the passwords once into `.secrets.env` (git ignored), uploads this folder and waits for the deployment.

You can also do it from the dashboard: create the app, set the variables above, then deploy an archive of this folder (`tar -czf postgres.tar.gz compose.yml pgadmin`).

## Connect

- pgAdmin: open `http://postgres.docker.localhost`, sign in with the pgAdmin email and password, open **checkship postgres** and type the `POSTGRES_PASSWORD`.
- From the host: `psql "postgres://app:<POSTGRES_PASSWORD>@localhost:<host port>/app"`, where the host port is shown on the deployment.

`POSTGRES_PASSWORD` only applies when the `pgdata` volume is created; changing it later needs `ALTER USER app PASSWORD '...'`.
