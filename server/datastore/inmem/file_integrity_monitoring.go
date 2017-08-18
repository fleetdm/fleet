package inmem

import (
	"github.com/kolide/fleet/server/kolide"
)

func (d *Datastore) NewFIMSection(fp *kolide.FIMSection, opts ...kolide.OptionalArg) (*kolide.FIMSection, error) {
	d.mtx.Lock()
	defer d.mtx.Unlock()
	fp.ID = d.nextID(fp)
	d.filePaths[fp.ID] = fp
	return fp, nil
}

func (d *Datastore) FIMSections() (kolide.FIMSections, error) {
	d.mtx.Lock()
	defer d.mtx.Unlock()
	result := make(kolide.FIMSections)
	for _, filePath := range d.filePaths {
		result[filePath.SectionName] = append(result[filePath.SectionName], filePath.Paths...)
	}
	return result, nil
}

func (d *Datastore) ClearFIMSections() error {
	panic("inmem is being deprecated")
}
