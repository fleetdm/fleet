import React from 'react';
import expect from 'expect';
import { mount } from 'enzyme';

import { ElipsisMenu } from './ElipsisMenu';

describe('ElipsisMenu - component', () => {
  it('Displays children on click', () => {
    const component = mount(
      <ElipsisMenu>
        <span>ElipsisMenu Children</span>
      </ElipsisMenu>
    );

    expect(component.state().showChildren).toEqual(false);
    expect(component.text()).toNotContain('ElipsisMenu Children');

    component.find('button').simulate('click');

    expect(component.state().showChildren).toEqual(true);
    expect(component.text()).toContain('ElipsisMenu Children');
  });
});
