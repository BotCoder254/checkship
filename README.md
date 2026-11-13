<p align="center">
  <picture>
    <source media="(prefers-color-scheme: dark)" srcset="docs/public/logo-dark.svg">
    <img src="docs/public/logo-light.svg" alt="checkship" height="56">
  </picture>
</p>

<p align="center">Painless self-hosted deployment platform.</p>

## Goal

Got an already working docker compose file for your project? Send it to your **checkship** instance and it goes live on your own infrastructure, with every service deployed and exposed on nice URLs, with certificates when you use `https`.

Docker is the only backend supported at the moment.

## Getting started

```sh
docker run -d \
  -e "ADMIN_EMAIL=admin@example.com" \
  -e "ADMIN_PASSWORD=admin" \
  -v "/var/run/docker.sock:/var/run/docker.sock" \
  -v "checkshipdata:/checkship/data" \
  -v "checkshipssh:/root/.ssh" \
  -p "8080:8080" \
  ghcr.io/checkship-deploy/checkship
```

Then open http://localhost:8080 and sign in with the credentials above. The [quickstart](docs/guide/quickstart.md) walks you through your first target, application and deployment.

## Documentation

- [Quickstart](docs/guide/quickstart.md)
- [Installation](docs/guide/installation.md) and [configuration](docs/guide/configuration.md)
- [Targets](docs/reference/targets.md), [applications](docs/reference/applications.md), [deployments](docs/reference/deployments.md), [Docker provider](docs/reference/providers/docker.md)
- [Continuous integration](docs/guide/continuous-integration-deployment.md) and [API](docs/reference/api.md)
- [AGENT.md](AGENT.md): architecture and system design for contributors and coding agents

## Development

```sh
make serve-back    # API on :8080 with an admin@example.com / admin account
make serve-front   # dashboard dev server, proxies /api/v1 to the API
make test          # frontend and backend tests
make build         # builds the dashboard, then the ./checkship binary
npm run build:icons  # regenerates assets/brand/icon.png and icon.ico from the brand mark
```

## License

checkship is free software released under the [GNU General Public License v3.0](LICENSE).
