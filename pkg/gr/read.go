package gr

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
func (g *GR) readCfgFirst(r *bufio.Reader) (box [3]float64, xyz XYZ, err error) {
	g.atoms, box, err = util.Header(r, nil, readSlice)
	if err != nil {
		err = fmt.Errorf("Header: %w", err)
		return
	}

	b, _ := r.ReadSlice('\n')
	fields := strings.Fields(string(b))

	if len(fields) <= 2 {
		err = fmt.Errorf("not enough columns (at least 3; got %d)", len(fields))
		return
	}
	fields = fields[2:]

	var found int
	g.colsLen = len(fields)
	for k, v := range fields {
		switch v {
		case "x":
			g.cols[0] = k
		case "y":
			g.cols[1] = k
		case "z":
			g.cols[2] = k
		case "type":
			g.cols[3] = k
		default:
			continue
		}
		found++
	}

	if found < len(g.cols) {
		return box, nil, fmt.Errorf("cannot find the columns x, y, z, and type")
	}

	g.order, xyz, err = g.fetchXYZFirst(r)
	if err != nil {
		return box, nil, fmt.Errorf("fetchXYZ: %w", err)
	}

	return
}

// readCfg reads a configuration of the LAMMPS trajectory. This method will call
// fetchXYZ to fetch the coordinates of the two atoms.
func (g *GR) readCfg(r *bufio.Reader) (box [3]float64, xyz XYZ, err error) {
	box, err = util.HeaderWOutAtoms(r, nil, readSlice)
	if err != nil {
		err = fmt.Errorf("HeaderWOutAtoms: %w", err)
		return
	}

	r.ReadSlice('\n')

	xyz, err = g.fetchXYZ(r)
	if err != nil {
		err = fmt.Errorf("fetchXYZ: %w", err)
		return
	}

	return
}

// fetchXYZ fetches the coordinates of the two atoms by calling readXYZ two
// times (one for the first atom, and the other for the second atom). This
// method is like fetchXYZ but it returns the order of the atoms.
func (g *GR) fetchXYZFirst(r *bufio.Reader) (order []string, xyz XYZ, err error) {
	xyz = make(XYZ, len(g.atomsTyp))
	nbat := g.atoms / len(g.atomsTyp)
	for _, v := range g.atomsTyp {
		xyz[v] = make([][3]float64, 0, nbat)
	}

	for i := 0; i < g.atoms; i++ {
		var typ string
		typ, err = g.readXYZ(r, xyz)
		if err != nil {
			return
		}

		if _, ok := g.Atoms[typ]; ok {
			order = append(order, typ)
		}
	}

	return
}

// fetchXYZ fetches the coordinates of the two atoms by calling readXYZ two
// times (one for the first atom, and the other for the second atom).
func (g *GR) fetchXYZ(r *bufio.Reader) (xyz XYZ, err error) {
	xyz = make(map[string][][3]float64, len(g.atomsTyp))
	nbat := g.atoms / len(g.atomsTyp)
	for _, v := range g.atomsTyp {
		xyz[v] = make([][3]float64, 0, nbat)
	}

	for i := 0; i < g.atoms; i++ {
		_, err = g.readXYZ(r, xyz)
		if err != nil {
			return
		}
	}

	return
}

// readXYZ reads the coordinates for each atom. If the atom type exists in XYZ,
// it is added to the map. It returns the type of the atom.
func (g *GR) readXYZ(r *bufio.Reader, xyz XYZ) (typ string, err error) {
	b, _ := r.ReadSlice('\n')
	fields := strings.Fields(string(b))
	if len(fields) != g.colsLen {
		err = fmt.Errorf("number of columns don't match: %d (expected %d)", len(fields), g.colsLen)
		return
	}

	typ = fields[g.cols[3]]
	xyzTyp, ok := xyz[typ]
	if !ok {
		return
	}

	var xyzTmp [3]float64
	for k := 0; k < 3; k++ {
		xyzTmp[k], _ = strconv.ParseFloat(fields[g.cols[k]], 64)
	}

	xyz[typ] = append(xyzTyp, xyzTmp)
	return
}

func readSlice(r *bufio.Reader, w io.Writer) []byte {
	b, _ := r.ReadSlice('\n')
	return b
}
