# Continuous Integration / Deployment

Integrating **checkship** to automatically deploy your services in your pipeline is **a breeze**.

Just use one of the following methods.

## cURL

checkship doesn't publish its own GitHub Action yet. From any CI, GitHub Actions included, trigger a deployment by calling the [checkship API](/reference/api) with a program like cURL.

On the **New deployment** page, you can show the `curl` command associated with the payload represented by the form. You just have to embed this command in your CI job, update it according to your needs (filling the git branch from the environment for example and retrieve the API Key from a secret) and you're good to go.

Here is an example for the [Drone CI software](https://www.drone.io/):

```yml
kind: pipeline
type: docker
name: default
steps:
  - name: deploy
    image: curlimages/curl
    environment:
      CHECKSHIP_API_KEY:
        from_secret: checkship_api_key
    commands:
      - /bin/sh -c 'curl -X POST -H "Content-Type:application/json" -H "Authorization:Bearer $CHECKSHIP_API_KEY" -d "{\"environment\":\"staging\",\"git\":{\"branch\":\"$DRONE_BRANCH\",\"hash\":\"$DRONE_COMMIT\"}}" https://checkship.example.com/api/v1/apps/2PvP5liIhcMn59yo5q6m53QWWXM/deployments'
```

::: info
You'll need to replace the domain `checkship.example.com` and the application id `2PvP5liIhcMn59yo5q6m53QWWXM` to match your configuration.
:::

### Cancel a superseded deployment

A deployment queued by an older commit can be cancelled when a newer one comes in, so only the latest runs. The deployment number is in the response of the call that queued it (`deployment_number`):

```sh
curl -X POST -H "Authorization: Bearer $CHECKSHIP_API_KEY" \
  https://checkship.example.com/api/v1/apps/2PvP5liIhcMn59yo5q6m53QWWXM/deployments/42/cancel
```

A deployment that has already finished answers `400` with `deployment_not_cancellable`, which a pipeline can ignore. See [Cancel a deployment](/reference/deployments#cancel) for what happens to a running one.
