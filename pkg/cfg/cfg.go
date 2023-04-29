// Package cfg dispatches several calculations. It avoids to start a
// specific program for each calculation.
package cfg

import (
	"fmt"
	"log"
	"os"
	"sync"

	"github.com/pelletier/go-toml"
)

// Cfg is a structure where the types of calculations are stored. It can be
// instanced through the New method. The length of the Files slice must be equal
// to the length of the Types files. Each calculation requires a configuration
// file where the parameters required to run the calculation are stored.
type Cfg struct {
	Types [][]string `toml:"types"`
	Files [][]string `toml:"files"`
}

// New returns an instance of the Cfg structure. It opens and reads the
// configuration file where Types and Files are stored. The configuration file
// must use the TOML format.
func New(path string) (Cfg, error) {
	f, err := os.Open(path)
	if err != nil {
		return Cfg{}, err
	}
	defer f.Close()

	var cfg Cfg
	dec := toml.NewDecoder(f)
	err = dec.Decode(&cfg)
	if err != nil {
		return Cfg{}, err
	}

	if len(cfg.Files) != len(cfg.Types) {
		return Cfg{}, fmt.Errorf("length of Files isn't equal to Types (%d vs %d)",
			len(cfg.Files), len(cfg.Types))
	}

	for k, v := range cfg.Files {
		if len(v) != len(cfg.Types[k]) {
			return Cfg{}, fmt.Errorf("length of Files isn't equal to Types (%d vs %d, step %d)",
				len(v), len(cfg.Types[k]), k)
		}
	}

	return cfg, nil
}

// Start dispatches and performs the calculations. If several calculations are
// in the same array (e.g Types: ["x", "y", "z"]), they will be performed in
// parrallel. In general, one calculation uses one thread. The length of the
// array must be in accordance with the number of threads used by the
// calculations and the number of threads available.
//
// It is a thread blocking method. If an error occurs for a specific
// calculation, the calculation will stop and log the error but the method won't
// stop.
func (c Cfg) Start(log *log.Logger) {
	var wg sync.WaitGroup
	for step, types := range c.Types {
		if len(types) == 0 {
			continue
		}

		if len(types) > 1 {
			for rtn, name := range types[1:] { // For each calculation
				wg.Add(1)
				go func(step, rtn int, name string) {
					err := Launch(name, c.Files[step][rtn])
					if err != nil {
						log.Println(fmt.Errorf("Launch (step %d, routine %d): %w", step, rtn, err))
					}

					wg.Done()
				}(step, rtn, name)

			}
		}

		err := Launch(types[0], c.Files[step][0])
		if err != nil {
			log.Println(fmt.Errorf("Launch (step %d, routine %d): %w", step, 0, err))
		}
		wg.Wait()
	}
}
