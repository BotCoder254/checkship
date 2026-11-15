# Jobs

A number of **background tasks** are managed by **checkship**. You can check the jobs page on the dashboard to see them at any time.

For the vast majority of cases, you may never have to look at them as they are processed without issues.

Jobs are processed by [runners](/guide/configuration#runners). A job starts as soon as it is queued if its runner has a free worker, otherwise as soon as one becomes free. A job shown as running is being processed by a worker.

Deployments of the same application environment run one at a time, in the order they were queued. The same goes for configurations of the same target.

::: info
By default, **jobs in error** state are retried every **15 seconds**. This is because some errors (such as the `target_configuration_in_progress`) are expected and will delay the job.
:::

## Backups

Backups, restores and uploads to the backup storage run on their own [runner](/guide/configuration#runners), and wait for any deployment of the same application environment. They differ from other jobs in one way: **a failed backup or restore is not retried on its own**. Its outcome is recorded on the backup, and you run it again from the [Backups](/reference/backups) page once the cause is fixed, since every attempt stops the application's containers.

A backup taken before a cleanup is the exception: it runs inside the cleanup job, which is retried until the backup succeeds, so nothing is deleted without a copy.

## Cancellation

Since a target on which you have, in the past, successfully deployed something can be destroyed from your side, **checkship** provides the ability to **cancel some tasks**.

From the **checkship** perspective, it has effectively deployed something and when, for example, deleting an application, **checkship** will queue a cleanup job which cannot succeed because a target is not reachable anymore.

For that **particular case**, you can press the **cancel button** on a job to allow the deletion to proceed without cleaning up resources.
