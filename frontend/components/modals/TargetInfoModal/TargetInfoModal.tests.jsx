import React from 'react';
import expect, { createSpy, restoreSpies } from 'expect';
import { mount } from 'enzyme';

import { createAceSpy } from '../../../test/helpers';
import TargetInfoModal from './TargetInfoModal';

describe('TargetInfoModal - component', () => {
  const hostTarget = {
    detail_updated_at: '2016-10-25T16:24:27.679472917-04:00',
    display_text: 'Jason Meller\'s Windows Note',
    hostname: 'Jason Meller\'s Windows Note',
    id: 2,
    ip: '192.168.1.11',
    mac: '0C-BA-8D-45-FD-B9',
    memory: 4145483776,
    os_version: 'Windows Vista 0.0.1',
    osquery_version: '2.0.0',
    platform: 'windows',
    status: 'offline',
    target_type: 'hosts',
    updated_at: '0001-01-01T00:00:00Z',
    uptime: 3600000000000,
    uuid: '1234-5678-9101',
  };
  const labelTarget = {
    count: 38,
    description: 'This group consists of machines utilized for developing within the WIN 10 environment',
    display_text: 'Windows 10 Development',
    hosts: [hostTarget],
    name: 'windows10',
    query: "SELECT * FROM last WHERE username = 'root' AND last.time > ((SELECT unix_time FROM time) - 3600);",
    target_type: 'labels',
  };

  afterEach(restoreSpies);

  it('renders a host modal when the target is a host', () => {
    const component = mount(<TargetInfoModal target={hostTarget} />);

    expect(component.find('.host-modal').length).toEqual(1);
  });

  it('renders a label modal when the target is a label', () => {
    createAceSpy();
    const component = mount(<TargetInfoModal target={labelTarget} />);

    expect(component.find('.label-modal').length).toEqual(1);
  });

  it('calls the onAdd prop when "ADD TO TARGETS" is clicked', () => {
    const onAddSpy = createSpy();
    const component = mount(
      <TargetInfoModal onAdd={onAddSpy} target={hostTarget} />
    );

    component.find('.target-info-modal__add-btn').simulate('click');
    expect(onAddSpy).toHaveBeenCalled();
  });

  it('calls the onExit prop when "CANCEL" is clicked', () => {
    const onExitSpy = createSpy();
    const component = mount(
      <TargetInfoModal onExit={onExitSpy} target={hostTarget} />
    );

    component.find('.target-info-modal__cancel-btn').simulate('click');
    expect(onExitSpy).toHaveBeenCalled();
  });
});
