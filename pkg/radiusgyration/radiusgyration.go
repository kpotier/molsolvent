// Package radiusgyration calculates the radius of gyration of a molecule.
package radiusgyration

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
var Type = "radius_gyration"

// RadiusGyration is a structure containing the parameters that can be parsed from
// a TOML configuration file. This structure can be instanced through the New
// method. It also contains other unexported informations like the number of
// atoms, and the number of columns.
// AtomStart must be lower than AtomEnd. Same for CfgStart and CfgEnd.
type RadiusGyration struct {
	FileIn  string `toml:"radius_gyration.file_in"`
	FileOut string `toml:"radius_gyration.file_out"`

	CfgStart int `toml:"radius_gyration.cfg_start"`
	CfgEnd   int `toml:"radius_gyration.cfg_end"`

	AtomStart int                `toml:"radius_gyration.atom_start"`
	AtomEnd   int                `toml:"radius_gyration.atom_end"`
	Masses    map[string]float64 `toml:"radius_gyration.masses"`

	Dt float64 `toml:"radius_gyration.dt"`

	atoms   int
	cols    [4]int
	colsLen int
}

// New returns an instance of the RadiusGyration structure. It reads and parses
// the configuration file given in argument. The file must be a TOML file.
func New(path string) (*RadiusGyration, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var radiusgyration RadiusGyration
	dec := toml.NewDecoder(f)
	err = dec.Decode(&radiusgyration)
	if err != nil {
		return nil, err
	}

	if radiusgyration.CfgStart >= radiusgyration.CfgEnd {
		return nil, errors.New("CfgStart is greater or equal than CfgEnd")
	}

	if radiusgyration.AtomStart >= radiusgyration.AtomEnd {
		return nil, errors.New("AtomStart is greater or equal than AtomEnd")
	}

	return &radiusgyration, nil
}

// Start performs the calculation. It is a thread blocking method. It is a very
// fast calculation. This calculation only use one thread.
func (r *RadiusGyration) Start() error {
	f, err := os.Open(r.FileIn)
	if err != nil {
		return err
	}
	defer f.Close()
	rd := bufio.NewReader(f)

	out, err := util.Write(r.FileOut, r)
	if err != nil {
		return fmt.Errorf("Write: %w", err)
	}
	defer out.Close()
	out.WriteString("cfg t radius\n")

	err = util.ReadCfgNonCvg(rd, r.CfgStart)
	if err != nil {
		return fmt.Errorf("ReadCfgNonCvg: %w", err)
	}

	xyz, types, err := r.readCfgFirst(rd)
	if err != nil {
		return fmt.Errorf("readCfgFirst: %w", err)
	}
	r.calc(out, 0, xyz, types)

	for i := 1; i < (r.CfgEnd - r.CfgStart); i++ {
		xyz, types, err := r.readCfg(rd)
		if err != nil {
			return fmt.Errorf("readCfg (step %d): %w", i, err)
		}
		r.calc(out, i, xyz, types)
	}

	return nil
}

// calc calculates the radius of gyration and writes the result into a file.
func (r *RadiusGyration) calc(w io.Writer, cfg int, xyz [][3]float64, types []string) error {
	var (
		com     [3]float64
		massTot float64
	)

	for key, v := range xyz {
		mass, ok := r.Masses[types[key]]
		if !ok {
			return fmt.Errorf("mass for atom type `%s` doesn't exist", types[key])
		}
		massTot += mass

		for k := 0; k < 3; k++ {
			com[k] += v[k] * mass
		}
	}

	for k := 0; k < 3; k++ {
		com[k] /= massTot
	}

	// MSD between COM & each XYZ
	var radius float64
	for _, v := range xyz {
		for k := 0; k < 3; k++ {
			mult := v[k] - com[k]
			radius += mult * mult
		}
	}

	radius /= float64(len(xyz) * 3)
	radius = math.Sqrt(radius)

	fmt.Fprintf(w, "%d %g %g\n",
		(cfg + r.CfgStart), (float64(cfg+r.CfgStart) * r.Dt), radius)

	return nil
}
