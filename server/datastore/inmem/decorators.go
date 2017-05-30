package inmem

import "github.com/kolide/kolide/server/kolide"

func (d *Datastore) SaveDecorator(dec *kolide.Decorator, opts ...kolide.OptionalArg) error {
	d.mtx.Lock()
	defer d.mtx.Unlock()
	if _, ok := d.decorators[dec.ID]; !ok {
		return notFound("Decorator")
	}
	d.decorators[dec.ID] = dec
	return nil
}

func (d *Datastore) NewDecorator(decorator *kolide.Decorator, opts ...kolide.OptionalArg) (*kolide.Decorator, error) {
	d.mtx.Lock()
	defer d.mtx.Unlock()
	decorator.ID = d.nextID(decorator)
	d.decorators[decorator.ID] = decorator
	return decorator, nil
}

func (d *Datastore) DeleteDecorator(id uint) error {
	d.mtx.Lock()
	defer d.mtx.Unlock()
	if _, ok := d.decorators[id]; !ok {
		return notFound("Decorator").WithID(id)
	}
	delete(d.decorators, id)
	return nil
}

func (d *Datastore) Decorator(id uint) (*kolide.Decorator, error) {
	d.mtx.Lock()
	defer d.mtx.Unlock()
	if result, ok := d.decorators[id]; ok {
		return result, nil
	}
	return nil, notFound("Decorator").WithID(id)
}

func (d *Datastore) ListDecorators(opts ...kolide.OptionalArg) ([]*kolide.Decorator, error) {
	d.mtx.Lock()
	defer d.mtx.Unlock()
	var result []*kolide.Decorator
	for _, dec := range d.decorators {
		result = append(result, dec)
	}
	return result, nil
}
