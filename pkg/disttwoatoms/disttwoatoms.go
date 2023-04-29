// Package disttwoatoms calculates the distance between two atoms over time.
package disttwoatoms

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"math"
	"os"

	"github.com/kpotier/molsolvent/pkg/util"

	"github.com/pelletier/go-toml"
)

// Type is name of the calculation.
var Type = "dist_two_atoms"

// DistTwoAtoms is a structure containing the parameters that can be parsed from
// a TOML configuration file. This structure can be instanced through the New
// method. It also contains other unexported informations like the number of
// atoms, and the number of columns.
// Atom1 must be lower than Atom2. Same for CfgStart and CfgEnd.
type DistTwoAtoms struct {
	FileIn  string `toml:"dist_two_atoms.file_in"`
	FileOut string `toml:"dist_two_atoms.file_out"`

	CfgStart int `toml:"dist_two_atoms.cfg_start"`
	CfgEnd   int `toml:"dist_two_atoms.cfg_end"`

	Atom1 int `toml:"dist_two_atoms.atom_1"`
	Atom2 int `toml:"dist_two_atoms.atom_2"`

	Dt float64 `toml:"dist_two_atoms.dt"`

	atoms   int
	cols    [3]int
	colsLen int
	dist    [][3]float64
}

// New returns an instance of the DistTwoAtoms structure. It reads and parses
// the configuration file given in argument. The file must be a TOML file.
func New(path string) (*DistTwoAtoms, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var distTwoAtoms DistTwoAtoms
	dec := toml.NewDecoder(f)
	err = dec.Decode(&distTwoAtoms)
	if err != nil {
		return nil, err
	}

	if distTwoAtoms.CfgStart >= distTwoAtoms.CfgEnd {
		return nil, errors.New("CfgStart is greater or equal than CfgEnd")
	}

	if distTwoAtoms.Atom1 >= distTwoAtoms.Atom2 {
		return nil, errors.New("Atom1 is greater or equal than Atom2")
	}

	return &distTwoAtoms, nil
}

// Start performs the calculation. It is a thread blocking method. It is a very
// fast calculation. This calculation only use one thread.
func (d *DistTwoAtoms) Start() error {
	f, err := os.Open(d.FileIn)
	if err != nil {
		return err
	}
	defer f.Close()
	r := bufio.NewReader(f)

	out, err := util.Write(d.FileOut, d)
	if err != nil {
		return fmt.Errorf("Write: %w", err)
	}
	defer out.Close()
	out.WriteString("cfg t x y z dist\n")

	err = util.ReadCfgNonCvg(r, d.CfgStart)
	if err != nil {
		return fmt.Errorf("ReadCfgNonCvg: %w", err)
	}

	xyz1, xyz2, err := d.readCfgFirst(r)
	if err != nil {
		return fmt.Errorf("readCfgFirst: %w", err)
	}
	d.result(out, 0, xyz1, xyz2)

	for i := 1; i <= (d.CfgEnd - d.CfgStart - 1); i++ {
		xyz1, xyz2, err := d.readCfg(r)
		if err != nil {
			return fmt.Errorf("readCfg (step %d): %w", i, err)
		}
		d.result(out, i, xyz1, xyz2)
	}

	return nil
}

// append calculates the distance between two set of coordinates and writes it
// into a file.
func (d *DistTwoAtoms) result(w io.Writer, cfg int, xyz1, xyz2 [3]float64) {
	var (
		vec  [3]float64
		dist float64
	)

	for k := 0; k < 3; k++ {
		vec[k] = (xyz1[k] - xyz2[k])
		dist += util.Pow(vec[k], 2)
	}
	dist = math.Sqrt(dist)

	fmt.Fprintf(w, "%d %g %g %g %g %g\n",
		(cfg + d.CfgStart), (float64(cfg+d.CfgStart) * d.Dt),
		vec[0], vec[1], vec[2], dist)
}
