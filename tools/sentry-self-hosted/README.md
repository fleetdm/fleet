# Running self-hosted Sentry

It may be useful to run a local, self-hosted version of Sentry for tests or to aid in monitoring a local development environment.

It is possible to do so by following the [steps documented on Sentry's website](https://develop.sentry.dev/self-hosted/).

While Sentry's documentation is canonical, the high-level steps are documented here and annotated with Fleet specific information:

1. `git clone` the [Sentry self-hosted repository](https://github.com/getsentry/self-hosted)
2. `git checkout` a specific version (e.g. `git checkout 24.2.0`)
3. Run `sudo ./install.sh` script (you may want to review the install scripts first, this takes a while to complete - maybe 30 minutes or so, you'll be prompted to create a Sentry user and password towards the end)
4. Once done, you should be able to run `docker-compose up -d` to bring up the self-hosted Sentry stack (that's a lot of containers to start)
5. Once the stack is up, you should be able to login at `http://localhost:9000` (on Google Chrome, after login I was met with a CSRF protection failure page, but it worked on Firefox)
6. In the "Issues" page, you should see a button labelled "Installation Instructions"; clicking on it will bring a page with the DSN that you can copy to use with Fleet (e.g. `http://<base64-data>@localhost:9000/1`)
7. Start `fleet serve`, passing the `--sentry_dsn http://<sentry-dsn>` flag to enable Sentry

You may now login to Fleet and any errors should show up in this local self-hosted version of Sentry.
