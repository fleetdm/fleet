import React from 'react';
import expect, { createSpy, restoreSpies } from 'expect';
import { mount } from 'enzyme';

import { hostStub } from 'test/stubs';
import HostDetails from 'components/hosts/HostDetails';

describe('HostDetails - component', () => {
  afterEach(restoreSpies);

  it('calls the onDestroyHost prop when the trash icon button is clicked', () => {
    const spy = createSpy();
    const component = mount(<HostDetails host={hostStub} onDestroyHost={spy} />);
    const btn = component.find('Button');

    btn.simulate('click');

    expect(spy).toHaveBeenCalled();
  });
});

