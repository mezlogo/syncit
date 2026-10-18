package repositorydeclaration

// This module contains REAL struct declarations for yaml using `gopkg.in/yaml.v3` lib

/*
Exmaple for expected yaml - all fields MUST be specified.
hosts/$host.yaml
```yaml
modules:
  - basic-linux-lts
  - basic-linux-amdgpu

tools:
  - pacman
  - aur
  - systemd
  - home
  - root

```
*/
type HostDeclarationYaml struct{}

/*
Exmaple for expected yaml - all fields are a map of string -> string[]
modules/$host/module.yaml
```yaml
pacman:
  - fish
  - nvim

aur:
  - vscode
  - leaf

systemd:
  - sytemd-networked.service
  - sytemd-resolved.service

```
*/
type ModuleDeclarationYaml struct{}
