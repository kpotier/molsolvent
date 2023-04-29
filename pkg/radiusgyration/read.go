package radiusgyration

import (
	"bufio"
	"fmt"
	"strconv"
	"strings"
)

// readCfgFirst reads the first configuration. It reads the number of atoms, the
// columns and performs the usual calculations like in readCfg.
func (r *RadiusGyration) readCfgFirst(rd *bufio.Reader) (xyz [][3]float64, types []string, err error) {
	for i := 0; i < 3; i++ {
		rd.ReadSlice('\n')
	}

	b, _ := rd.ReadSlice('\n')
	r.atoms, err = strconv.Atoi(string(b)[:len(b)-1])
	if err != nil {
		return
	}

	for i := 0; i < 4; i++ {
		rd.ReadSlice('\n')
	}

	b, _ = rd.ReadSlice('\n')
	fields := strings.Fields(string(b))

	if len(fields) <= 2 {
		err = fmt.Errorf("not enough columns (at least 3; got %d)", len(fields))
		return
	}
	fields = fields[2:]

	var found int
	r.colsLen = len(fields)
	for k, v := range fields {
		switch v {
		case "xu":
			r.cols[0] = k
		case "yu":
			r.cols[1] = k
		case "zu":
			r.cols[2] = k
		case "type":
			r.cols[3] = k
		default:
			continue
		}
		found++
	}

	if found < len(r.cols) {
		err = fmt.Errorf("cannot find the columns xu yu, zu, and type")
		return
	}

	xyz, types, err = r.fetchXYZ(rd)
	if err != nil {
		err = fmt.Errorf("fetchXYZ: %w", err)
		return
	}

	return
}

// readCfg reads a configuration of the LAMMPS trajectory. This method will call
// fetchXYZ to fetch the coordinates of the two atoms.
func (r *RadiusGyration) readCfg(rd *bufio.Reader) (xyz [][3]float64, types []string, err error) {
	for i := 0; i < 9; i++ {
		rd.ReadSlice('\n')
	}

	xyz, types, err = r.fetchXYZ(rd)
	if err != nil {
		err = fmt.Errorf("fetchXYZ: %w", err)
	}

	return
}

// fetchXYZ fetches the coordinates of the two atoms by calling readXYZ two
// times (one for the first atom, and the other for the second atom).
func (r *RadiusGyration) fetchXYZ(rd *bufio.Reader) (xyz [][3]float64, types []string, err error) {
	for i := 0; i < r.AtomStart; i++ {
		rd.ReadSlice('\n')
	}

	for i := 0; i < (r.AtomEnd - r.AtomStart); i++ {
		b, _ := rd.ReadSlice('\n')
		fields := strings.Fields(string(b))
		if len(fields) != r.colsLen {
			err = fmt.Errorf("number of columns don't match: %d (expected %d)", len(fields), r.colsLen)
			return
		}

		var xyzTmp [3]float64
		for k := 0; k < 3; k++ {
			xyzTmp[k], _ = strconv.ParseFloat(fields[r.cols[k]], 64)
		}
		types = append(types, fields[r.cols[3]])
		xyz = append(xyz, xyzTmp)
	}

	for i := 0; i < (r.atoms - r.AtomEnd); i++ {
		rd.ReadSlice('\n')
	}

	return
}
