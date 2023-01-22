# getpocket-collector

This Golang program just create History for getpocket links.

Work example: [juev/links](https://github.com/juev/links)

## How to use

You can create new ENV variables:

- `GETPOCKET_FEED_URL` for storing your RSS-feed from getpocket. This feed should be unprotected.
- `USERNAME` for storing username in LICENSE information. Default value: "juev"

Reference:

- [Can I subscribe to my list via RSS?](https://help.getpocket.com/article/1074-can-i-subscribe-to-my-list-via-rss)

## Github action

```yaml
name: Cronjob operations

on:
  schedule:
    - cron: "*/15 * * * *" # Runs every 15 minuts
  workflow_dispatch:
jobs:
  fetch:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3
        with:
          fetch-depth: 0
      - name: 💿 Setup
        run: |
          wget https://github.com/juev/getpocket-collector/releases/latest/download/getpocket-collector-linux-amd64 -O run
          chmod +x ./run
      - name: 🚀 Run Automation
        run: ./run
        env:
          GETPOCKET_FEED_URL: "https://getpocket.com/users/juev/feed/all"
          USERNAME: "juev"
      - name: 🐳 Commit
        uses: EndBug/add-and-commit@v9
        with:
          default_author: github_actions
          message: 'update'
          add: 'README.md'
```