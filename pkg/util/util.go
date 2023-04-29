// Package util contains some methods that can be used by every other package.
package util

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/pelletier/go-toml"
)

// Write writes the output file according to a specific scheme. It writes the
// date, parses the structure in a TOML format and writes it. This method
// returns the file for further writing. It must be closed at the end of the
// calculation.
func Write(path string, structure interface{}) (*os.File, error) {
	f, err := os.Create(path)
	if err != nil {
		return nil, err
	}

	fmt.Fprintf(f, "Date: %v\n", time.Now().Format("2006-01-02 15:04:05 -0700 MST"))

	enc := toml.NewEncoder(f)
	err = enc.Encode(structure)
	if err != nil {
		return nil, err
	}

	f.Write([]byte{'\n'})
	return f, nil
}

// ReadCfgNonCvg reads x non converged configurations. These non configurations
// will be automatically "discarded" and won't be taken into account. It is a
// very fast method.
func ReadCfgNonCvg(r *bufio.Reader, x int) error {
	if x == 0 {
		return nil
	}

	for i := 0; i < 3; i++ {
		r.ReadSlice('\n')
	}

	b, _ := r.ReadSlice('\n')
	atoms, err := strconv.Atoi(string(b)[:len(b)-1])
	if err != nil {
		return err
	}

	for i := 0; i < (5 + atoms); i++ {
		r.ReadSlice('\n')
	}

	// Other cfg until x
	for i := 0; i < ((x - 1) * (3 + 6 + atoms)); i++ {
		r.ReadSlice('\n')
	}

	return nil
}

// Pow returns x**y, the base-x exponential of y.
func Pow(x float64, n int) float64 {
	res := x
	for i := 0; i < (n - 1); i++ {
		res *= x
	}
	return res
}
