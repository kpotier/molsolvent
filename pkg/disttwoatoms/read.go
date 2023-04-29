package disttwoatoms

import (
	"bufio"
	"errors"
	"fmt"
	"strconv"
	"strings"
)

// readCfgFirst reads the first configuration. It reads the number of atoms, the
// columns and performs the usual calculations like in readCfg. This method
// saves a lot of runtime because it saves the number of columns. It is
// therefore non essential to re-read the number of columns and detect where the
// interesting columns are located.
func (d *DistTwoAtoms) readCfgFirst(r *bufio.Reader) (xyz1 [3]float64, xyz2 [3]float64, err error) {
	for i := 0; i < 3; i++ {
		r.ReadSlice('\n')
	}

	b, _ := r.ReadSlice('\n')
	d.atoms, err = strconv.Atoi(string(b)[:len(b)-1])
	if err != nil {
		return
	}

	for i := 0; i < 4; i++ {
		r.ReadSlice('\n')
	}

	b, _ = r.ReadSlice('\n')
	fields := strings.Fields(string(b))

	if len(fields) <= 2 {
		err = fmt.Errorf("not enough columns (at least 3; got %d)", len(fields))
		return
	}
	fields = fields[2:]

	var found int
	d.colsLen = len(fields)
	for k, v := range fields {
		switch v {
		case "xu":
			d.cols[0] = k
		case "yu":
			d.cols[1] = k
		case "zu":
			d.cols[2] = k
		default:
			continue
		}
		found++
	}

	if found < len(d.cols) {
		err = errors.New("cannot find the columns xu, yu, and zu")
		return
	}

	xyz1, xyz2, err = d.fetchXYZ(r)
	if err != nil {
		err = fmt.Errorf("fetchXYZ: %w", err)
		return
	}

	return
}

// readCfg reads a configuration of the LAMMPS trajectory. This method will call
// fetchXYZ to fetch the coordinates of the two atoms. readCfgFirst must be
// called before using this method as it doesn't read the number of atoms nor
// it analyzes the columns.
func (d *DistTwoAtoms) readCfg(r *bufio.Reader) (xyz1 [3]float64, xyz2 [3]float64, err error) {
	for i := 0; i < 9; i++ {
		r.ReadSlice('\n')
	}

	xyz1, xyz2, err = d.fetchXYZ(r)
	if err != nil {
		err = fmt.Errorf("fetchXYZ: %w", err)
	}

	return
}

// fetchXYZ fetches the coordinates of the two atoms by calling readXYZ two
// times (one for the first atom, and the other for the second atom).
func (d *DistTwoAtoms) fetchXYZ(r *bufio.Reader) (xyz1 [3]float64, xyz2 [3]float64, err error) {
	for i := 0; i < d.Atom1; i++ {
		r.ReadSlice('\n')
	}

	xyz1, err = d.readXYZ(r)
	if err != nil {
		err = fmt.Errorf("readXYZ: %w", err)
		return
	}

	for i := 0; i < (d.Atom2 - d.Atom1 - 1); i++ {
		r.ReadSlice('\n')
	}

	xyz2, err = d.readXYZ(r)
	if err != nil {
		err = fmt.Errorf("readXYZ: %w", err)
		return
	}

	for i := 0; i < (d.atoms - d.Atom2 - 1); i++ {
		r.ReadSlice('\n')
	}

	return
}

func (d *DistTwoAtoms) readXYZ(r *bufio.Reader) (xyz [3]float64, err error) {
	b, _ := r.ReadSlice('\n')
	fields := strings.Fields(string(b))
	if len(fields) != d.colsLen {
		err = fmt.Errorf("number of columns don't match: %d (expected %d)", len(fields), d.colsLen)
		return
	}

	for k := 0; k < 3; k++ {
		xyz[k], _ = strconv.ParseFloat(fields[d.cols[k]], 64)
	}

	return
}
