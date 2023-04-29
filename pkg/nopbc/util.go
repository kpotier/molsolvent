package nopbc

import (
	"bufio"
	"io"
	"strconv"
)

// readSlice reads until \n and writes it into a file. It also returns the line
// that have been read.
func readSlice(r *bufio.Reader, w io.Writer) []byte {
	b, _ := r.ReadSlice('\n')
	w.Write(b)
	return b
}

func (n *NoPBC) write(w io.Writer, fields []string, xyz [3]float64) {
	var bytes []byte
	for k, v := range fields {
		switch k {
		case n.cols[0]:
			bytes = strconv.AppendFloat(bytes, xyz[0], 'g', -1, 64)
		case n.cols[1]:
			bytes = strconv.AppendFloat(bytes, xyz[1], 'g', -1, 64)
		case n.cols[2]:
			bytes = strconv.AppendFloat(bytes, xyz[2], 'g', -1, 64)
		default:
			bytes = append(bytes, []byte(v)...)
		}
		bytes = append(bytes, ' ')
	}
	bytes = append(bytes, '\n')
	w.Write(bytes)
}
