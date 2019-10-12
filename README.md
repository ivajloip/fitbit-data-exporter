# Fitbit Data Exporter

Fitbit Data Exporter is a project that allows me to visualize in a more
user-friendly way my fitbit data. Currently only heart rate data is supported.

## Building

### Executable

To build the executable, run `make build`.

### Docker

To build the docker image, run `make runnable-container`.

## Run

Go to [the new app page][1]. And fill in with your information. The expected
`Callback URL` is `http://127.0.0.1:5556/auth/fitbit/callback`. The 
`OAuth 2.0 Application Type` was having value `Personal` during the tests.

Take the value of `OAuth 2.0 Client ID` and `Client Secret` as they will be
needed. 

### Without docker

Example run:

```
./build/fitbit-data-exporter \
    --postgresql-dsn postgres://${POSTGRES_USER}:${POSTGRES_PASSWORD}@postgres/${POSTGRES_DB}?sslmode=disable \
    --starting-date 48h \
    api --client-id <client-id> \
        --client-secret <client-secret>
```

### With docker-compose

Put the values in `DOCKER_CLIENT_ID` and `DOCKER_CLIENT_SECRET` in `deployment/.env`.

[1]: https://dev.fitbit.com/apps/new
