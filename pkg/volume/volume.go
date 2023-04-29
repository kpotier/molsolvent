// Package volume calculates the volume of a molecule in a solvent.
package volume

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"math"
	"os"
	"runtime"
	"sync"
	"time"

	"github.com/kpotier/molsolvent/pkg/util"

	"github.com/pelletier/go-toml"
)

// Type is the type of calculation.
var Type = "volume"

// XYZ is a type that represents the coordinates for each atom.
type XYZ map[string][][3]float64

// Volume is a structure containing the parameters that can be parsed from
// a TOML configuration file. This structure can be instanced through the New
// method. It also contains other unexported informations like the number of
// atoms, the number of columns, ...
// CfgStart must be lower than CfgEnd. Size of the Bloc and Blocs must be equal
// to 3.
type Volume struct {
	FileIn     string `toml:"volume.file_in"`
	FileOut    string `toml:"volume.file_out"`
	FileOutXYZ string `toml:"volume.file_out_xyz"`

	CfgStart   int `toml:"volume.cfg_start"`
	CfgEnd     int `toml:"volume.cfg_end"`
	CfgSpacing int `toml:"volume.cfg_spacing"`

	Bloc  []float64 `toml:"volume.bloc"`
	Blocs []int     `toml:"volume.blocs"` // Blocs around each atom

	Atoms []string           `toml:"volume.atoms"`
	Sigma map[string]float64 `toml:"volume.sigma"`

	Dt float64 `toml:"volume.dt"`

	atOther []string
	sigma2  map[string]float64

	atoms   int
	cols    [4]int
	colsLen int

	cfg int
	err error
	mux sync.Mutex
	wg  sync.WaitGroup
}

// New returns an instance of the Volume structure. It reads and parses
// the configuration file given in argument. The file must be a TOML file.
func New(path string) (*Volume, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var volume Volume
	dec := toml.NewDecoder(f)
	err = dec.Decode(&volume)
	if err != nil {
		return nil, err
	}

	if volume.CfgStart >= volume.CfgEnd {
		return nil, errors.New("CfgStart is greater or equal than CfgEnd")
	}

	volume.sigma2 = make(map[string]float64, len(volume.Sigma))
	for atom, sigma := range volume.Sigma {
		var found bool
		for _, i := range volume.Atoms {
			if i == atom {
				found = true
				break
			}
		}

		if !found {
			volume.atOther = append(volume.atOther, atom)
		}

		volume.sigma2[atom] = util.Pow(sigma, 2)
	}

	if len(volume.Bloc) != 3 || len(volume.Blocs) != 3 {
		return nil, errors.New("length of Blocs or Bloc is not equal to 3")
	}

	return &volume, nil
}

// Start performs the calculation. It is a thread blocking method. This
// calculation will use all the threads available.
func (v *Volume) Start() error {
	f, err := os.Open(v.FileIn)
	if err != nil {
		return err
	}
	defer f.Close()
	r := bufio.NewReader(f)

	out, err := util.Write(v.FileOut, v)
	if err != nil {
		return fmt.Errorf("Write: %w", err)
	}
	defer out.Close()
	out.WriteString("cfg t vol(atoms) vol(other)\n")

	tFirst := time.Now()

	err = util.ReadCfgNonCvg(r, v.CfgStart)
	if err != nil {
		return fmt.Errorf("ReadCfgNonCvg: %w", err)
	}

	xyz, box, err := v.readCfgFirst(r)
	if err != nil {
		return fmt.Errorf("readCfgFirst: %w", err)
	}
	v.calc(out, v.CfgStart, box, xyz)
	v.cfg = v.CfgStart

	tFirstDur := time.Since(tFirst)
	tOther := time.Now()

	for i := 0; i < (runtime.NumCPU() - 1); i++ {
		v.wg.Add(1)
		go v.start(r, out)
	}

	v.wg.Add(1)
	v.start(r, out)
	v.wg.Wait()

	tOtherDur := time.Since(tOther)
	fmt.Fprintf(out, "\nTime (first): %s\nTime (other): %s\nTime (total): %s\n", tFirstDur, tOtherDur, (tFirstDur + tOtherDur))

	if v.err != nil {
		return v.err
	}

	return nil
}

func (v *Volume) start(r *bufio.Reader, out io.Writer) {
	for {
		v.mux.Lock()
		v.cfg += v.CfgSpacing + 1
		if v.cfg >= v.CfgEnd || v.err != nil {
			break
		}

		err := util.ReadCfgNonCvg(r, v.CfgSpacing)
		if err != nil {
			if v.err == nil {
				v.err = fmt.Errorf("ReadCfgNonCvg (step %d): %w", v.cfg, err)
			}
			break
		}

		xyz, box, err := v.readCfg(r)
		if err != nil {
			if v.err == nil {
				v.err = fmt.Errorf("readCfg (step %d): %w", v.cfg, err)
			}
			break
		}

		currentCfg := v.cfg // copy
		v.mux.Unlock()

		v.calc(out, currentCfg, box, xyz)
	}

	v.mux.Unlock()
	v.wg.Done()
}

// calc calculates the volume and writes the result into a file
func (v *Volume) calc(w io.Writer, cfg int, box [3]float64, xyz XYZ) {
	var boxBlocs [3]int
	for k := 0; k < 3; k++ {
		boxBlocs[k] = int(math.Round(box[k] / v.Bloc[k]))
	}

	ptsX := make(map[float64]bool) // float64 to avoid casting
	ptsY := make(map[float64]bool)
	ptsZ := make(map[float64]bool)

	for _, atom := range v.Atoms {
		for _, xyzt := range xyz[atom] {
			var bloc [3]int // In which bloc is the molecule
			for k := 0; k < 3; k++ {
				bloc[k] = int(xyzt[k] / v.Bloc[k])
			}

			for x := (bloc[0] - v.Blocs[0]); x <= (bloc[0] + v.Blocs[0]); x++ {
				posx := x
				if posx < 0 {
					posx += boxBlocs[0]
				} else if posx >= boxBlocs[0] {
					posx -= boxBlocs[0]
				}

				ptsX[float64(posx)] = true
			}

			for y := (bloc[1] - v.Blocs[1]); y <= (bloc[1] + v.Blocs[1]); y++ {
				posy := y
				if posy < 0 {
					posy += boxBlocs[1]
				} else if posy >= boxBlocs[1] {
					posy -= boxBlocs[1]
				}

				ptsY[float64(posy)] = true
			}

			for z := (bloc[2] - v.Blocs[2]); z <= (bloc[2] + v.Blocs[2]); z++ {
				posz := z
				if posz < 0 {
					posz += boxBlocs[2]
				} else if posz >= boxBlocs[2] {
					posz -= boxBlocs[2]
				}

				ptsZ[float64(posz)] = true
			}
		}
	}

	pts := make(map[[3]float64]bool, (len(ptsX) * len(ptsY) * len(ptsZ))) // true if atoms
	for x := range ptsX {
		for y := range ptsY {
			for z := range ptsZ {
				lit := [3]float64{x, y, z}
				distTmp := math.MaxFloat64
				var ptsTmp bool // true => atoms. false = other

				var pos [3]float64
				for k := 0; k < 3; k++ {
					pos[k] = (v.Bloc[k] * lit[k]) + (v.Bloc[k] / 2.)
				}

				for _, atom := range v.Atoms {
					for _, xyzt := range xyz[atom] {
						var dist float64
						for k := 0; k < 3; k++ {
							distatt := xyzt[k] - pos[k]
							dist += util.Pow((distatt - box[k]*math.Round(distatt/box[k])), 2)
						}
						dist /= v.sigma2[atom]

						if dist < distTmp {
							distTmp = dist
							ptsTmp = true
						}
					}
				}

				for _, atom := range v.atOther {
					for _, xyzt := range xyz[atom] {
						var dist float64
						for k := 0; k < 3; k++ {
							distatt := xyzt[k] - pos[k]
							dist += util.Pow((distatt - box[k]*math.Round(distatt/box[k])), 2)
						}
						dist = math.Sqrt(dist)
						dist /= v.Sigma[atom]

						if dist < distTmp {
							distTmp = dist
							ptsTmp = false
							break // We don't longer need to check
						}
					}

					if !ptsTmp {
						break // We don't longer need to check because it's sure that ptsTmp is false.
					}
				}

				if ptsTmp {
					pts[lit] = ptsTmp
				}
			}
		}
	}

	volBloc := v.Bloc[0] * v.Bloc[1] * v.Bloc[2]
	volAt := volBloc * float64(len(pts))
	volOt := (box[0] * box[1] * box[2]) - volAt

	fmt.Fprintf(w, "%d %g %g %g\n", cfg, float64(cfg)*v.Dt, volAt, volOt)

	if cfg == v.CfgStart {
		v.xyz(pts)
	}
}

// for test purpose only.
func (v *Volume) xyz(pts map[[3]float64]bool) error {
	f, err := os.Create(v.FileOutXYZ)
	if err != nil {
		return err
	}
	defer f.Close()

	fmt.Fprintln(f, len(pts), "\n Atom C == solvent")

	for k, val := range pts {
		at := "C"
		if val {
			at = "O"
		}
		fmt.Fprintln(f, at, (k[0]*v.Bloc[0] + v.Bloc[0]/2.), (k[1]*v.Bloc[1] + v.Bloc[1]/2.), (k[2]*v.Bloc[2] + v.Bloc[2]/2.))
	}

	return nil
}
