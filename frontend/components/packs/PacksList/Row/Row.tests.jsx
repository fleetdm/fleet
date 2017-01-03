import React from 'react';
import expect, { createSpy, restoreSpies } from 'expect';
import { mount } from 'enzyme';

import Row from 'components/packs/PacksList/Row';
import { packStub } from 'test/stubs';

describe('PacksList - Row - component', () => {
  afterEach(restoreSpies);

  it('renders', () => {
    expect(mount(<Row pack={packStub} />).length).toEqual(1);
  });

  it('calls the onCheck prop with the value and pack id when checked', () => {
    const spy = createSpy();
    const component = mount(<Row checked onCheck={spy} pack={packStub} />);

    component.find({ name: `select-pack-${packStub.id}` }).simulate('change');

    expect(spy).toHaveBeenCalledWith(false, packStub.id);
  });
});

