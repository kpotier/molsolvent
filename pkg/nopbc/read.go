package nopbc

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"io"
	"strconv"
	"strings"

	"github.com/kpotier/molsolvent/pkg/util"
)

func (n *NoPBC) readCfgFirst(r *bufio.Reader, w io.Writer) ([][3]float64, error) {
	atoms, box, err := util.Header(r, w, readSlice)
	if err != nil {
		return nil, fmt.Errorf("Header: %w", err)
	}

	n.atoms = atoms
	var box2 [3]float64
	for k := 0; k < 3; k++ {
		box2[k] = box[k] / 2.
	}

	b, _ := r.ReadSlice('\n')
	fields := strings.Fields(string(b))

	if len(fields) <= 2 {
		return nil, fmt.Errorf("not enough columns (at least 3; got %d)", len(fields))
	}

	var buf bytes.Buffer
	for k := 0; k < 2; k++ {
		buf.WriteString(fields[k])
		buf.WriteByte(' ')
	}

	var found int
	fields = fields[2:] // Omission of ITEM: ATOMS
	n.colsLen = len(fields)

	for k, v := range fields {
		switch v {
		case "x":
			n.cols[0] = k
			buf.WriteString("xu") // unwrapped (see Lammps doc)
		case "y":
			n.cols[1] = k
			buf.WriteString("yu")
		case "z":
			n.cols[2] = k
			buf.WriteString("zu")
		case "mol":
			n.cols[3] = k
			buf.WriteString(v)
		default:
			buf.WriteString(v)
			buf.WriteByte(' ')
			continue
		}
		buf.WriteByte(' ')
		found++
	}

	buf.WriteByte('\n')
	w.Write(buf.Bytes())
	n.colsBuf = buf.Bytes()

	if found < len(n.cols) {
		return nil, fmt.Errorf("cannot find the columns x, y, z, and mol")
	}

	// Check PBC for each atom in each molecule
	var (
		xyz     [][3]float64
		mol     string
		lastXYZ [3]float64
		size    [3]float64
	)

	for i := 0; i < atoms; i++ {
		b, _ := r.ReadSlice('\n')

		fields := strings.Fields(string(b))
		if len(fields) != n.colsLen {
			return nil, fmt.Errorf("number of columns don't match (id %d, got %d, expected %d)", i, len(fields), n.colsLen)
		}

		if fields[n.cols[3]] != mol {
			mol = fields[n.cols[3]]
			for k := 0; k < 3; k++ {
				lastXYZ[k], _ = strconv.ParseFloat(fields[n.cols[k]], 64)
			}

			size = box2
			sizeMap, ok := n.Size[fields[n.cols[3]]]
			if ok {
				for k := 0; k < 3; k++ {
					size[k] = sizeMap[k]
				}
			}
		} else {
			for k := 0; k < 3; k++ {
				xyz, _ := strconv.ParseFloat(fields[n.cols[k]], 64)
				dist := lastXYZ[k] - xyz
				if dist > size[k] {
					xyz += box[k]
				} else if dist < -size[k] {
					xyz -= box[k]
				}

				lastXYZ[k] = xyz
			}
		}

		xyz = append(xyz, lastXYZ)
		n.write(w, fields, lastXYZ)
	}

	_, err = r.ReadByte()
	if err != nil {
		if !errors.Is(err, io.EOF) {
			return nil, err
		}
	}
	r.UnreadByte()
	return xyz, nil
}

func (n *NoPBC) readCfg(r *bufio.Reader, w io.Writer, lastXYZ [][3]float64) error {
	corr := make([][3]float64, n.atoms)

	for {
		box, err := util.HeaderWOutAtoms(r, w, readSlice)
		if err != nil {
			return fmt.Errorf("HeaderWOutAtoms: %w", err)
		}

		var box2 [3]float64
		for k := 0; k < 3; k++ {
			box2[k] = box[k] / 2.
		}

		r.ReadSlice('\n')
		w.Write(n.colsBuf)

		for i := 0; i < n.atoms; i++ {
			l, _ := r.ReadSlice('\n')

			fields := strings.Fields(string(l))
			if len(fields) != n.colsLen {
				return fmt.Errorf("number of columns don't match (id %d, got %d, expected %d)", i, len(fields), n.colsLen)
			}

			for k := 0; k < 3; k++ {
				xyz, _ := strconv.ParseFloat(fields[n.cols[k]], 64)
				xyz += corr[i][k]

				dist := lastXYZ[i][k] - xyz
				if dist > box2[k] {
					corr[i][k] += box[k]
					xyz += box[k]
				} else if dist < -box2[k] {
					corr[i][k] -= box[k]
					xyz -= box[k]
				}
				lastXYZ[i][k] = xyz
			}

			n.write(w, fields, lastXYZ[i])
		}

		_, err = r.ReadByte()
		if err != nil {
			if errors.Is(err, io.EOF) {
				break
			}
			return err
		}
		r.UnreadByte()
	}

	return nil
}
