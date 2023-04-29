package volume

import (
	"bufio"
	"fmt"
	"io"
	"strconv"
	"strings"

	"github.com/kpotier/molsolvent/pkg/util"
)

// readCfgFirst reads the first configuration. It reads the number of atoms, the
// columns and performs the usual calculations like in readCfg.
func (v *Volume) readCfgFirst(r *bufio.Reader) (XYZ, [3]float64, error) {
	var err error
	var box [3]float64
	v.atoms, box, err = util.Header(r, nil, readSlice)
	if err != nil {
		return nil, box, fmt.Errorf("Header: %w", err)
	}

	b, _ := r.ReadSlice('\n')
	fields := strings.Fields(string(b))

	if len(fields) <= 2 {
		return nil, box, fmt.Errorf("not enough columns (at least 3, got %d)", len(fields))
	}
	fields = fields[2:]

	var found int
	v.colsLen = len(fields)
	for k, val := range fields {
		switch val {
		case "x":
			v.cols[0] = k
		case "y":
			v.cols[1] = k
		case "z":
			v.cols[2] = k
		case "type":
			v.cols[3] = k
		default:
			continue
		}
		found++
	}

	if found < len(v.cols) {
		return nil, box, fmt.Errorf("cannot find the columns x y z, and type")
	}

	xyz, err := v.fetchXYZ(r)
	if err != nil {
		return nil, box, fmt.Errorf("fetchXYZ: %w", err)
	}

	return xyz, box, nil
}

// readCfg reads a configuration of the LAMMPS trajectory. This method will call
// fetchXYZ to fetch the coordinates of the two atoms.
func (v *Volume) readCfg(r *bufio.Reader) (XYZ, [3]float64, error) {
	var err error
	var box [3]float64
	box, err = util.HeaderWOutAtoms(r, nil, readSlice)
	if err != nil {
		return nil, box, fmt.Errorf("HeaderWOutAtoms: %w", err)
	}

	r.ReadSlice('\n')

	xyz, err := v.fetchXYZ(r)
	if err != nil {
		return nil, box, fmt.Errorf("fetchXYZ: %w", err)
	}

	return xyz, box, nil
}

// fetchXYZ fetches the coordinates of the two atoms by calling readXYZ two
// times (one for the first atom, and the other for the second atom).
func (v *Volume) fetchXYZ(r *bufio.Reader) (XYZ, error) {
	xyz := make(XYZ, len(v.Sigma))
	nbat := v.atoms / len(v.Sigma)
	for k := range v.Sigma {
		xyz[k] = make([][3]float64, 0, nbat)
	}

	for i := 0; i < v.atoms; i++ {
		b, _ := r.ReadSlice('\n')
		fields := strings.Fields(string(b))
		if len(fields) != v.colsLen {
			return nil, fmt.Errorf("number of columns don't match: %d (expected %d)", len(fields), v.colsLen)
		}

		typ := fields[v.cols[3]]
		_, ok := v.Sigma[typ]
		if !ok {
			continue
		}

		var xyzt [3]float64
		for k := 0; k < 3; k++ {
			xyzt[k], _ = strconv.ParseFloat(fields[v.cols[k]], 64)
		}

		xyz[typ] = append(xyz[typ], xyzt)
	}
	return xyz, nil
}

func readSlice(r *bufio.Reader, w io.Writer) []byte {
	b, _ := r.ReadSlice('\n')
	return b
}
