package cfg

import (
	"fmt"

	"github.com/kpotier/molsolvent/pkg/disttwoatoms"
	"github.com/kpotier/molsolvent/pkg/gr"
	"github.com/kpotier/molsolvent/pkg/nopbc"
	"github.com/kpotier/molsolvent/pkg/radiusgyration"
	"github.com/kpotier/molsolvent/pkg/volume"
)

// Calculation is an interface that only contains one method: Start. Every
// calculation must have a Start method that will launch the calculation. It
// must be a thread blocking method.
type Calculation interface {
	Start() error
}

// Launch launchs a specific calculation. It is a thread blocking method. The
// parameters required to launch the calculation must be in a file.
func Launch(name string, path string) error {
	var (
		err error
		cal Calculation
	)

	switch name {
	case nopbc.Type:
		cal, err = nopbc.New(path)
	case disttwoatoms.Type:
		cal, err = disttwoatoms.New(path)
	case radiusgyration.Type:
		cal, err = radiusgyration.New(path)
	case gr.Type:
		cal, err = gr.New(path)
	case volume.Type:
		cal, err = volume.New(path)
	default:
		return fmt.Errorf("calculation `%s` doesn't exist", name)
	}

	if err != nil {
		return fmt.Errorf("%s: New: %w", name, err)
	}

	err = cal.Start()
	if err != nil {
		return fmt.Errorf("%s: Start: %w", name, err)
	}

	return nil
}
