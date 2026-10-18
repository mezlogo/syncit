# syncit

## Persist, evolve, decommission, reuse your home and root configuration over time, network and even distributive.

Use cases:
- synchronize multiple similar configuration for similar machines
- reuse module between different purpose: home server, ubuntu vps, iGPU notebook, cuda machine, qemu vm
- orginize all features by module for reusing by multiple different hosts AND for better coherent
- review and migrate your system from single git repo

## Highlevel design

1. Read `./port/model/model.go` - this is a list of the most valuable structures

## Few words about project organization

- main.go is a main entrypoint with whole program IoC
- cobra based subcommands handles help and completion generation, subcommand mapping, options and arguments mapping and validation, os env handling and default values. All cobra related import isolated in `./cmd/$subcommand/$subcommand.go`
- port and adapters for isolating external or library integrations
- commands are only IN flow, that's why all port and adapters are OUT flow - this is simple CLI tool, no need to make it complex
- core modules are in memory isolated real application logic. Given level of isolation makes it trivial for fast yet almost totaly unit test logic using only standard table based style with mocks 