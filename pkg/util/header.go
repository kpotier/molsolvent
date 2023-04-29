package util

import (
	"bufio"
	"fmt"
	"io"
	"strconv"
	"strings"
)

// Header corresponds to the lines specific to a Lammps trajectory file. It
// contains the size of the box and the number of atoms. This method returns the
// number of atoms, the size of the box, the size of the box divided by two.
func Header(r *bufio.Reader, w io.Writer, readSlice func(r *bufio.Reader, w io.Writer) []byte) (atoms int, box [3]float64, err error) {
	for l := 0; l < 3; l++ {
		readSlice(r, w)
	}

	atomsStr := strings.TrimSpace(string(readSlice(r, w)))
	atoms, _ = strconv.Atoi(atomsStr)

	readSlice(r, w)

	box, err = HeaderBox(r, w, readSlice)
	return
}

// HeaderBox returns the box size.
func HeaderBox(r *bufio.Reader, w io.Writer, readSlice func(r *bufio.Reader, w io.Writer) []byte) (box [3]float64, err error) {
	for k := 0; k < 3; k++ {
		b := readSlice(r, w)

		fields := strings.Fields(string(b))
		if len(fields) != 2 {
			err = fmt.Errorf("unable to get the size of the box")
			return
		}

		lmin, _ := strconv.ParseFloat(fields[0], 64)
		lmax, _ := strconv.ParseFloat(fields[1], 64)

		box[k] = lmax - lmin
	}

	return
}

// HeaderWOutAtoms returns the size of the box, the size of the box divided by
// two. It is like HeaderBox but without the number of atoms.
func HeaderWOutAtoms(r *bufio.Reader, w io.Writer, readSlice func(r *bufio.Reader, w io.Writer) []byte) (box [3]float64, err error) {
	for l := 0; l < 5; l++ {
		readSlice(r, w)
	}

	return HeaderBox(r, w, readSlice)
}
