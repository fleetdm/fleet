import React from 'react';
import expect, { createSpy, restoreSpies } from 'expect';
import { mount } from 'enzyme';

import { hostStub } from 'test/stubs';
import HostsTable from 'components/hosts/HostsTable';

describe('HostsTable - component', () => {
  afterEach(restoreSpies);

  it('calls the onDestroyHost prop when the action button is clicked on an offline host', () => {
    const destroySpy = createSpy();
    const querySpy = createSpy();
    const offlineHost = { ...hostStub, status: 'offline' };

    const offlineComponent = mount(
      <HostsTable
        hosts={[offlineHost]}
        onDestroyHost={destroySpy}
        onQueryHost={querySpy}
      />
    );
    const btn = offlineComponent.find('Button');

    expect(btn.find('Icon').prop('name')).toEqual('trash');

    btn.simulate('click');

    expect(destroySpy).toHaveBeenCalled();
    expect(querySpy).toNotHaveBeenCalled();
  });

  it('calls the onDestroyHost prop when the action button is clicked on a mia host', () => {
    const destroySpy = createSpy();
    const querySpy = createSpy();
    const miaHost = { ...hostStub, status: 'mia' };

    const miaComponent = mount(
      <HostsTable
        hosts={[miaHost]}
        onDestroyHost={destroySpy}
        onQueryHost={querySpy}
      />
    );
    const btn = miaComponent.find('Button');

    expect(btn.find('Icon').prop('name')).toEqual('trash');

    btn.simulate('click');

    expect(destroySpy).toHaveBeenCalled();
    expect(querySpy).toNotHaveBeenCalled();
  });

  it('calls the onQueryHost prop when the action button is clicked on an online host', () => {
    const destroySpy = createSpy();
    const querySpy = createSpy();
    const onlineHost = { ...hostStub, status: 'online' };

    const onlineComponent = mount(
      <HostsTable
        hosts={[onlineHost]}
        onDestroyHost={destroySpy}
        onQueryHost={querySpy}
      />
    );
    const btn = onlineComponent.find('Button');

    expect(btn.find('Icon').prop('name')).toEqual('query');

    btn.simulate('click');

    expect(destroySpy).toNotHaveBeenCalled();
    expect(querySpy).toHaveBeenCalled();
  });
});
