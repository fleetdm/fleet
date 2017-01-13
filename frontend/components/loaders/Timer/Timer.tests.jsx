import React from 'react';
import { mount } from 'enzyme';
import expect, { spyOn, restoreSpies } from 'expect';

import Timer from './Timer';

describe('Timer - component', () => {
  afterEach(restoreSpies);

  it('play() and pause() function', () => {
    const timer = mount(<Timer running={false} />);

    expect(timer.node.interval).toNotExist();
    timer.setProps({ running: true });
    expect(timer.node.interval).toExist();
    timer.setProps({ running: false });
    expect(timer.node.interval).toNotExist();
  });

  it('should reset after pause', () => {
    const timer = mount(<Timer running={false} />);
    const spy = spyOn(timer.node, 'reset').andCallThrough();

    timer.setProps({ running: true });

    expect(spy).toHaveBeenCalled();
  });

  it('should not reset when stopped', () => {
    const timer = mount(<Timer running />);
    const spy = spyOn(timer.node, 'reset').andCallThrough();

    timer.setProps({ running: false });

    expect(spy).toNotHaveBeenCalled();
  });

  it('should not reset when it continues', () => {
    const timer = mount(<Timer running />);
    const spy = spyOn(timer.node, 'reset').andCallThrough();

    timer.setProps({ running: true });

    expect(spy).toNotHaveBeenCalled();
  });
});
