import React from 'react';
import expect, { createSpy, restoreSpies } from 'expect';
import { mount } from 'enzyme';

import PacksList from 'components/packs/PacksList';
import { packStub } from 'test/stubs';

describe('PacksList - component', () => {
  afterEach(restoreSpies);

  it('renders', () => {
    expect(mount(<PacksList packs={[packStub]} />).length).toEqual(1);
  });

  it('calls the onCheckAllPacks prop when select all packs checkbox is checked', () => {
    const spy = createSpy();
    const component = mount(<PacksList onCheckAllPacks={spy} packs={[packStub]} />);

    component.find({ name: 'select-all-packs' }).simulate('change');

    expect(spy).toHaveBeenCalledWith(true);
  });

  it('calls the onCheckPack prop when a pack checkbox is checked', () => {
    const spy = createSpy();
    const component = mount(<PacksList onCheckPack={spy} packs={[packStub]} />);
    const packCheckbox = component.find({ name: `select-pack-${packStub.id}` });

    packCheckbox.simulate('change');

    expect(spy).toHaveBeenCalledWith(true, packStub.id);
  });
});
