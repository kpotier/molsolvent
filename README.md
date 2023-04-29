# molsolvent [![go.dev reference](https://img.shields.io/badge/go.dev-reference-007d9c?logo=go&logoColor=white&style=flat-square)](https://pkg.go.dev/github.com/kpotier/molsolvent)
Tools to calculate different properties of a molecule in a solvent.

### Supported formats

1. Lammps Trajectory (.lammpstrj)

### Usage

1. Install ```Go 1.14^```.

2. Go to the ```cmd``` directory.

3. Execute ```go build``` or ```go install```.

### Additional information

1. The executable takes only one argument: the path of the configuration file. It must be a TOML file. An example can be found in the root directory: ```cfg.toml```.
