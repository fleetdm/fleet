import React from 'react';
import { mount } from 'enzyme';
import { noop } from 'lodash';

import QueryDetailsSidePanel from 'components/side_panels/QueryDetailsSidePanel';
import { queryStub } from 'test/stubs';

describe('QueryDetailsSidePanel - component', () => {
  it('renders', () => {
    const component = mount(<QueryDetailsSidePanel onEditQuery={noop} query={queryStub} />);

    expect(component.length).toEqual(1);
  });

  it('renders a read-only Kolide Ace component with the query text', () => {
    const component = mount(<QueryDetailsSidePanel onEditQuery={noop} query={queryStub} />);
    const aceEditor = component.find('KolideAce');

    expect(aceEditor.length).toEqual(1);
    expect(aceEditor.prop('value')).toEqual(queryStub.query);
    expect(aceEditor.prop('readOnly')).toEqual(true);
  });

  it('calls the onEditQuery prop when Edit/Run Query is clicked', () => {
    const spy = jest.fn();
    const component = mount(<QueryDetailsSidePanel onEditQuery={spy} query={queryStub} />);
    const button = component.find('Button');

    button.simulate('click');

    expect(spy).toHaveBeenCalledWith(queryStub);
  });
});
