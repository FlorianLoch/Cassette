For the git revision to be available at build time in dokku's environment it is essential to configure dokku to keep the .git directory:

```bash
# keep the .git directory during builds
dokku git:set <app> keep-git-dir true
```