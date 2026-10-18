# syncit

## Persist, evolve, decommission, reuse your home and root configuration over time, network and even distributive.

Use cases:
- synchronize multiple similar configuration for similar machines
- reuse module between different purpose: home server, ubuntu vps, iGPU notebook, cuda machine, qemu vm
- orginize all features by module for reusing by multiple different hosts AND for better coherent
- review and migrate your system from single git repo

## Usage

Common options for all subcommands:

- `--host $HOSTNAME` - name of file `./hosts/$host.yaml` when no option specify take from env `$SYNCIT_HOST`, when no env read hostname (cat /etc/hostname)
- `--repo $REPO_DIR` - string could contain multiple dir locations separated by ",". When not specify take from `$SYNCIT_REPO` - possible separated list of values. When none raise an error.
- `--output-home` - optional dir name. Used for testing. Default is "$HOME"
- `--output-root` - optional dir name. Used for testing. Default is "/"
- `--only` - string could contain multiple tool names separated by ",". When empty - use all tools. `--only config --only

### Command: report

This command just prints declaration for given repository and hostname.

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

### Idea of tools

Tool is a handler for file or field in `module declaration`. For example:

- fields `systemd` and `systemduser` are simple features for calling `sudo systemctl enable $service` and `systemctl --user enable $service` - both are just a bash call for each entry. Entry could have duplicates - doesn't matter, we create set or distinct operation and handle only unique once.
- fields like `pacman` and `aur` are simple features for calling `sudo pacman -S --needed $packages` and `yay -S --needed $aurs` - both are just a bash call for combined and unique entries.
- files inside `$module/` like `$module/config/nvim/lua/init.lua` would be LINKED from `$HOME/.config/nvim/lua/init.lua`
- files inside `$module/` like `$module/etc/systemd/network/20-wired.network` would be COPY with sudo to `/etc/systemd/network/20-wired.network`

I want to focus on possible lots of small improvments:
- additional `$module/ssh` for `$HOME/.ssh` mapping using links
- additional `$module/local` for `$HOME/.local`
- additional `$module/home` for `$HOME` when there is no reasons to create a specific one and mapping using links
- additional `$module/boot` for `/boot` mapping using `sudo cp`
- additional `$module/root` for `/` when there is no reasons to create a specific and mapping using `sudo cp`

That's why I want to introduce 3 basic tools:

#### core tool 1. process

All declared values are gathered into a collection, then converted into a list of unique values, and then spawn a process ethier for each value or for all.

For better control process always starts in interactive mode using PIPE for password or any other interactions. 

1. `systemd` requires call for each value
2. `pacman` requires all values separated by whitespace

#### core tool 2. home configuration

For home configuration there is no need to COPY files - links work better, cos' I can edit file direclty `vi $HOME/.config/git/config` and thouse changes would persist in git immediatly.
Moreover no sudo required.
Just create whole dir tree and link files only.

#### core tool 3. root configuration

Like home configuration we create a whole file tree using `sudo mkdir -p $DIR` and copy files using `sudo cp $SOURCE $TARGET`

### summary for core tools

| tool | core | info |
| --- | --- | --- |
| systemd | process-for-each-unique | `sudo systemctl enable {{ .Entry }}` |
| systemduser | process-for-each-unique | `systemctl --user enable {{ .Entry }}` |
| config | user-configuration | map `config/` to `--home/.config/` |
| ssh | user-configuration | map `ssh/` to `--home/.ssh/` |
| home | user-configuration | map `home/` to `--home/` |
| etc | root-configuration | map `etc/` to `--root/etc` |
| root | root-configuration | map `root/` to `--root/` |

## Example

For getting hand-on experienc this repository contains an example for you at `./example/`

## Highlevel design

1. Read `./port/*/*.go` - this is an any level of real isolation
2. Read `./coremodule/*/*.go` - this is a pure logic
3. Read `./adapter/*/*.go` - this is a window to external libraries or os 

## Few words about project organization

- main.go is a main entrypoint with whole program IoC
- cobra based subcommands handles help and completion generation, subcommand mapping, options and arguments mapping and validation, os env handling and default values. All cobra related import isolated in `./cmd/$subcommand/$subcommand.go`
- port and adapters for isolating external or library integrations
- commands are only IN flow, that's why all port and adapters are OUT flow - this is simple CLI tool, no need to make it complex
- core modules are in memory isolated real application logic. Given level of isolation makes it trivial for fast yet almost totaly unit test logic using only standard table based style with mocks 