import React from 'react';
import expect, { createSpy, restoreSpies } from 'expect';
import { mount } from 'enzyme';

import ClickableTableRow from './index';

const clickSpy = createSpy();
const dblClickSpy = createSpy();

const props = {
  onClick: clickSpy,
  onDoubleClick: dblClickSpy,
};

describe('ClickableTableRow - component', () => {
  afterEach(restoreSpies);

  it('calls onDblClick when row is double clicked', () => {
    const queryRow = mount(<ClickableTableRow {...props} />);
    queryRow.find('tr').simulate('doubleclick');
    expect(dblClickSpy).toHaveBeenCalled();
  });

  it('calls onSelect when row is clicked', () => {
    const queryRow = mount(<ClickableTableRow {...props} />);
    queryRow.find('tr').simulate('click');
    expect(clickSpy).toHaveBeenCalled();
  });
});
