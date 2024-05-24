# getpocket-collector

This Golang program just create History for getpocket links.

Work example: [juev/links](https://github.com/juev/links)

## How to use

1. Create an application on the page [getpocket.com/developer/apps](https://getpocket.com/developer/apps/). Here you will get the `consumer_key`.
2. Creating a token on the page [Authenticate Pocket 30](https://reader.fxneumann.de/plugins/oneclickpocket/auth.php).

These values must be stored in the environment variables:

- `CONSUMER_KEY` to store the consumer_key
- `ACCESS_TOKEN` to store the access_token 
- `USERNAME` for storing username in LICENSE information. Default value: "juev"

## Github action

```yaml
name: Cronjob operations

on:
  schedule:
    - cron: "*/15 * * * *" # Runs every 15 minutes
  workflow_dispatch:
jobs:
  fetch:
    runs-on: ubuntu-latest
    container: ghcr.io/juev/getpocket-collector:latest
    steps:
      - 
        uses: actions/checkout@v3
        with:
          fetch-depth: 1
      - 
        name: 🚀 Run Automation
        run: getpocket-collector
        env:
          GETPOCKET_FEED_URL: "https://getpocket.com/users/juev/feed/all"
          USERNAME: "juev"
      - 
        name: 🐳 Commit
        uses: EndBug/add-and-commit@v9
        with:
          default_author: github_actions
          message: 'update'
```