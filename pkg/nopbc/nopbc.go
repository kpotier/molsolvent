// Package nopbc converts a lammps trajectory file where the atoms follow the
// periodic boundary conditions into a file where the periodic boundary
// conditions no longer exists.
package nopbc

import (
	"bufio"
	"fmt"
	"os"

	"github.com/pelletier/go-toml"
)

// Type is name of the calculation.
var Type = "no_pbc"

// NoPBC is a structure containing the parameters that can be parsed from
// a TOML configuration file. This structure can be instanced through the New
// method. It also contains other unexported informations like the number of
// atoms, the number of columns.
type NoPBC struct {
	FileIn  string               `toml:"no_pbc.file_in"`
	FileOut string               `toml:"no_pbc.file_out"`
	Size    map[string][]float64 `toml:"no_pbc.size"`

	atoms   int
	cols    [4]int
	colsBuf []byte
	colsLen int
}

// New returns an instance of the NoPBC structure. It reads and parses
// the configuration file given in argument. The file must be a TOML file.
func New(path string) (*NoPBC, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var noPBC NoPBC
	dec := toml.NewDecoder(f)
	err = dec.Decode(&noPBC)
	if err != nil {
		return nil, err
	}

	for k, v := range noPBC.Size {
		if len(v) != 3 {
			return nil, fmt.Errorf("length of size for %s isn't equal to 3 but %d",
				k, len(v))
		}
	}

	return &noPBC, nil
}

// Start performs the calculation. It is a thread blocking method. This
// calculation only use one thread.
func (n *NoPBC) Start() error {
	f, err := os.Open(n.FileIn)
	if err != nil {
		return err
	}
	defer f.Close()
	r := bufio.NewReader(f)

	out, err := os.Create(n.FileOut)
	if err != nil {
		return err
	}
	defer out.Close()

	lastXYZ, err := n.readCfgFirst(r, out)
	if err != nil {
		return fmt.Errorf("readCfgFirst: %w", err)
	}

	err = n.readCfg(r, out, lastXYZ)
	if err != nil {
		return fmt.Errorf("readCfg: %w", err)
	}

	return nil
}
