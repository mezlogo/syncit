# syncit example

This example shows you how you can create multiple repo configuration for even more control over your system, like public (ssh configuration, packages, services) and private (ssh keys, tokens, sensitive data).

This demo shows that module has a repository isolation and how to handle multi repo with override tools.

## 1. multiple hosts and modules

1. show report for one repo

```bash
syncit report --repo ./example/repoA --host hostA
```

2. show report for two repos

```bash
syncit report --repo ./example/repoA -repo ./example/repoB --host hostA
```

3. show diff for one repo

```bash
syncit diff --repo ./example/repoA --host hostA
```

4. sync without call os

```bash
syncit sync --repo ./example/repoA --dry-run --host hostA --output-config ./example/fakeconfig --output-etc /tmp/fakeetc
```

## 2. repository isolates host declaration and modules

- two repo may declare module with same name, however when repo A doesn't declare this module for given host - this module is ignored no matter what it means
