// Package gr calculates the radial distribution function and its integral.
package gr

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"math"
	"os"
	"runtime"
	"sync"

	"github.com/kpotier/molsolvent/pkg/util"

	"github.com/pelletier/go-toml"
)

// Type is the type of calculation.
var Type = "gr"

// XYZ is a type that represents the coordinates for each atom.
type XYZ map[string][][3]float64

// GR is a structure containing the parameters that can be parsed from
// a TOML configuration file. This structure can be instanced through the New
// method. It also contains other unexported informations like the number of
// atoms, the number of columns, the size of the box, the average size of the
// box, ...
// CfgStart must be lower than CfgEnd.
type GR struct {
	FileIn  string `toml:"gr.file_in"`
	FileOut string `toml:"gr.file_out"`

	CfgStart int `toml:"gr.cfg_start"`
	CfgEnd   int `toml:"gr.cfg_end"`

	Atoms map[string][]string `toml:"gr.atoms"`

	RMax float64 `toml:"gr.rmax"`
	Dr   float64 `toml:"gr.dr"`

	bins  int
	rmax2 float64

	atomsTyp []string
	atoms    int
	vol      float64

	hstg  map[[2]string][][]float64
	order []string

	cols    [4]int
	colsLen int

	xyzLen map[string]float64

	cfg int
	err error
	mux sync.Mutex
	wg  sync.WaitGroup
}

// New returns an instance of the GR structure. It reads and parses
// the configuration file given in argument. The file must be a TOML file.
func New(path string) (*GR, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var gr GR
	dec := toml.NewDecoder(f)
	err = dec.Decode(&gr)
	if err != nil {
		return nil, err
	}

	if gr.CfgStart >= gr.CfgEnd {
		return nil, errors.New("CfgStart is greater or equal than CfgEnd")
	}

	gr.bins = int(gr.RMax / gr.Dr)

	if gr.bins <= 1 {
		return nil, errors.New("the number of bins must be greater than 1")
	}

	gr.rmax2 = util.Pow(gr.RMax, 2)

	var combinaisons int
	for at1, arrAt2 := range gr.Atoms {
		gr.atomsTyp = append(gr.atomsTyp, at1)
		for _, at2 := range arrAt2 {
			var found bool
			for _, v := range gr.atomsTyp {
				if v == at2 {
					found = true
					break
				}
			}

			if !found {
				gr.atomsTyp = append(gr.atomsTyp, at2)
			}

			combinaisons++
		}
	}

	gr.hstg = make(map[[2]string][][]float64, combinaisons)
	gr.xyzLen = make(map[string]float64, len(gr.atomsTyp))

	return &gr, nil
}

// Start performs the calculation. It is a thread blocking method. This
// calculation will use all the threads available.
func (g *GR) Start() error {
	f, err := os.Open(g.FileIn)
	if err != nil {
		return err
	}
	defer f.Close()
	r := bufio.NewReader(f)

	err = util.ReadCfgNonCvg(r, g.CfgStart)
	if err != nil {
		return fmt.Errorf("ReadCfgNonCvg: %w", err)
	}

	box, xyz, err := g.readCfgFirst(r)
	if err != nil {
		return fmt.Errorf("readCfgFirst: %w", err)
	}

	for at1, arrAt2 := range g.Atoms { // Initialize the histogram map
		for _, at2 := range arrAt2 {
			g.hstg[[2]string{at1, at2}] = make([][]float64, len(xyz[at1]))
			for i := 0; i < len(xyz[at1]); i++ {
				g.hstg[[2]string{at1, at2}][i] = make([]float64, g.bins)
			}
		}
	}

	for k, v := range xyz {
		g.xyzLen[k] = float64(len(v))
	}

	g.calc(box, xyz)
	g.cfg = g.CfgStart

	for i := 0; i < (runtime.NumCPU() - 1); i++ {
		g.wg.Add(1)
		go g.start(r)
	}

	g.wg.Add(1)
	g.start(r)
	g.wg.Wait()

	if g.err != nil {
		return g.err
	}

	out, err := util.Write(g.FileOut, g)
	if err != nil {
		return fmt.Errorf("Write: %w", err)
	}
	defer out.Close()
	g.write(out)

	return nil
}

func (g *GR) start(r *bufio.Reader) {
	for {
		g.mux.Lock()
		g.cfg++
		if g.cfg >= g.CfgEnd || g.err != nil {
			break
		}

		box, xyz, err := g.readCfg(r)
		if err != nil {
			if g.err == nil {
				g.err = fmt.Errorf("readCfg (step %d): %w", g.cfg, err)
			}
			break
		}
		g.mux.Unlock()
		g.calc(box, xyz)
	}

	g.mux.Unlock()
	g.wg.Done()
}

// calc increments the histogram.
func (g *GR) calc(box [3]float64, xyz XYZ) {
	for at1, arrAt2 := range g.Atoms {
		for xyz1, xyzAt1 := range xyz[at1] {
			for _, at2 := range arrAt2 {
				for _, xyzAt2 := range xyz[at2] { // For each combinaison
					var dist float64
					for k := 0; k < 3; k++ {
						distatt := xyzAt1[k] - xyzAt2[k]
						dist += util.Pow((distatt - box[k]*math.Round(distatt/box[k])), 2)
					}

					if dist <= g.rmax2 {
						dist = math.Sqrt(dist)
						index := int(dist / g.Dr)
						g.mux.Lock()
						g.hstg[[2]string{at1, at2}][xyz1][index] += 1.
						g.mux.Unlock()
					}
				}
			}
		}
	}

	g.mux.Lock()
	defer g.mux.Unlock()
	g.vol += box[0] * box[1] * box[2]
}

// write writes the results of this calculation into a file.
func (g *GR) write(w io.Writer) error {
	// Volume for each bin
	var vol []float64
	for i := 0; i < g.bins; i++ {
		vol = append(vol, (4. / 3. * math.Pi *
			(util.Pow((float64(i+1)*g.Dr), 3) - util.Pow((float64(i)*g.Dr), 3))))
	}

	// Average of the volume
	g.vol /= float64(g.CfgEnd - g.CfgStart)

	// g(r) and its integral
	intg := make(map[[2]string][][]float64)
	nbCfg := float64(g.CfgEnd - g.CfgStart)
	for at1, arrAt2 := range g.Atoms {
		for _, at2 := range arrAt2 {
			key := [2]string{at1, at2}
			intg[key] = make([][]float64, len(g.hstg[key]))

			for atomID, bins := range g.hstg[key] {
				intg[key][atomID] = make([]float64, g.bins)

				intg[key][atomID][0] = bins[0] / nbCfg
				g.hstg[key][atomID][0] = intg[key][atomID][0] / (vol[0] * g.xyzLen[at2] / g.vol)
				for bin, hstg := range bins[1:] {
					bin++
					intg[key][atomID][bin] = hstg / nbCfg
					g.hstg[key][atomID][bin] = intg[key][atomID][bin] / (vol[bin] * g.xyzLen[at2] / g.vol)
					intg[key][atomID][bin] += intg[key][atomID][bin-1]
				}
			}

		}
	}

	// Write the results
	// Header
	fmt.Fprint(w, "dist ")

	var orderList [][2]string
	orderListIncr := make(map[[2]string]int)

	for _, order := range g.order {
		for _, v := range g.Atoms[order] {
			lit := [2]string{order, v}
			if _, ok := orderListIncr[lit]; !ok {
				orderListIncr[lit] = 0
			}

			fmt.Fprint(w, order, "-", v, "(", orderListIncr[lit], ")-intg ")
			fmt.Fprint(w, order, "-", v, "(", orderListIncr[lit], ")-hstg ")
			orderList = append(orderList, lit)
			orderListIncr[lit]++
		}
	}
	fmt.Fprint(w, "\n")

	// Results for each bin
	for i := 0; i < g.bins; i++ {
		orderListIncr := make(map[[2]string]int)
		fmt.Fprint(w, ((float64(i+1) - 0.5) * g.Dr), " ")

		for _, v := range orderList {
			if _, ok := orderListIncr[v]; !ok {
				orderListIncr[v] = 0
			}
			fmt.Fprint(w, intg[v][orderListIncr[v]][i], " ", g.hstg[v][orderListIncr[v]][i], " ")
			orderListIncr[v]++
		}
		fmt.Fprint(w, "\n")
	}

	return nil
}
