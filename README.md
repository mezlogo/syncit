# syncit

## Persist, evolve, decommission, reuse your home and root configuration over time, network and even distributive.

Use cases:
- synchronize multiple similar configuration for similar machines
- reuse module between different purpose: home server, ubuntu vps, iGPU notebook, cuda machine, qemu vm
- orginize all features by module for reusing by multiple different hosts AND for better coherent
- review and migrate your system from single git repo

## Idea behind the scene

I want to declare in yaml and modules DESIRE configuration. Right now I'm totally fine with following capabilities:
- enabling systemd services `sudo systemctl enable systemd-networkd.service`
- enabling user services `systemctl --user enable ssh-agent.service`
- installing pacman packages `sudo pacman -S --needed noto-fonts-emoji`
- installing aur packages `yay -S leaf-markdown-viewer-bin`
- in user space create dirs and links only files to repo `mkdir -p ~/.config/environment.d/; ln -s $REPO/modules/ssh-agent/config/environment.d/ssh-agent.conf $CONFIG/environment.d/ssh-agent.conf`
- in root space create dirs with sudo and COPY files `sudo mkdir -p /etc/keyd/; sudo cp $REPO/modules/extended/etc/keyd/default.conf $ETC/keyd/default.conf

Using only following tools we can restore almost everything in computer:
- copy for root
- link for user
- install pacman, aur (or even apt)
- enable systemd (root and user)

## Example

For getting hand-on experienc this repository contains an example for you at `./example/`

## Highlevel design

1. Read `./port/model/model.go` - this is a list of the most valuable structures
2. Read `./port/*/*.go` - this is an any level of real isolation
3. Read `./adapter/yaml/dto.go` - this is expected models for files

## Few words about project organization

- main.go is a main entrypoint with whole program IoC
- cobra based subcommands handles help and completion generation, subcommand mapping, options and arguments mapping and validation, os env handling and default values. All cobra related import isolated in `./cmd/$subcommand/$subcommand.go`
- port and adapters for isolating external or library integrations
- commands are only IN flow, that's why all port and adapters are OUT flow - this is simple CLI tool, no need to make it complex
- core modules are in memory isolated real application logic. Given level of isolation makes it trivial for fast yet almost totaly unit test logic using only standard table based style with mocks 