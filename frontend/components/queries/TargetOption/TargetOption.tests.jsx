import React from 'react';
import expect, { createSpy, restoreSpies } from 'expect';
import { mount } from 'enzyme';
import { noop } from 'lodash';

import TargetOption from './TargetOption';

describe('TargetOption - component', () => {
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

  it('renders a label option for label targets', () => {
    const component = mount(<TargetOption onMoreInfoClick={noop} target={labelTarget} />);
    expect(component.find('.--is-label').length).toEqual(1);
    expect(component.text()).toContain(`${labelTarget.count} hosts`);
  });

  it('renders a host option for host targets', () => {
    const component = mount(<TargetOption onMoreInfoClick={noop} target={hostTarget} />);
    expect(component.find('.--is-host').length).toEqual(1);
    expect(component.find('i.kolidecon-single-host').length).toEqual(1);
    expect(component.find('i.kolidecon-windows').length).toEqual(1);
    expect(component.text()).toContain(hostTarget.ip);
  });

  it('renders the TargetInfoModal when shouldShowModal is true', () => {
    const component = mount(
      <TargetOption
        onMoreInfoClick={noop}
        target={hostTarget}
        shouldShowModal
      />
    );
    expect(component.find('TargetInfoModal').length).toEqual(1);
  });

  it('calls the onSelect prop when ADD button is clicked', () => {
    const onSelectSpy = createSpy();
    const component = mount(
      <TargetOption
        onMoreInfoClick={noop}
        onSelect={onSelectSpy}
        target={hostTarget}
      />
    );
    component.find('.target-option__btn').simulate('click');
    expect(onSelectSpy).toHaveBeenCalled();
  });

  it('calls the onMoreInfoClick prop when "more info" button is clicked', () => {
    const onMoreInfoClickSpy = createSpy();
    const onMoreInfoClick = () => {
      return onMoreInfoClickSpy;
    };
    const component = mount(
      <TargetOption
        onMoreInfoClick={onMoreInfoClick}
        target={hostTarget}
      />
    );
    component.find('.target-option__more-info').simulate('click');
    expect(onMoreInfoClickSpy).toHaveBeenCalled();
  });
});
