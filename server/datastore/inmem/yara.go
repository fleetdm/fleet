package inmem

import "github.com/kolide/kolide-ose/server/kolide"

func (d *Datastore) NewYARASignatureGroup(ysg *kolide.YARASignatureGroup) (*kolide.YARASignatureGroup, error) {
	d.mtx.Lock()
	defer d.mtx.Unlock()
	ysg.ID = d.nextID(ysg)
	d.yaraSignatureGroups[ysg.ID] = ysg
	return ysg, nil
}

func (d *Datastore) NewYARAFilePath(fileSectionName, sigGroupName string) error {
	d.mtx.Lock()
	defer d.mtx.Unlock()
	d.yaraFilePaths[fileSectionName] = append(d.yaraFilePaths[fileSectionName], sigGroupName)
	return nil
}

func (d *Datastore) YARASection() (*kolide.YARASection, error) {
	d.mtx.Lock()
	defer d.mtx.Unlock()
	result := &kolide.YARASection{
		Signatures: make(map[string][]string),
		FilePaths:  make(map[string][]string),
	}
	for _, ysg := range d.yaraSignatureGroups {
		result.Signatures[ysg.SignatureName] = append(result.Signatures[ysg.SignatureName], ysg.Paths...)
	}
	for fileSection, sigSection := range d.yaraFilePaths {
		result.FilePaths[fileSection] = append(result.FilePaths[fileSection], sigSection...)
	}

	return result, nil
}
