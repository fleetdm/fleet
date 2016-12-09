import React from 'react';
import expect from 'expect';
import { mount } from 'enzyme';

import { connectedComponent, reduxMockStore } from '../../../test/helpers';
import ConnectedNewHostPage, { NewHostPage } from './NewHostPage';

describe('New Host Page - component', () => {
  it('saves text to the clipboard when clipboard icons are clicked', () => {
    const mockStore = reduxMockStore();
    const page = mount(
      connectedComponent(ConnectedNewHostPage, { mockStore })
    );
    const icon = page.find('.kolidecon-clipboard').first();
    icon.simulate('click');

    const dispatchedActionMessages = mockStore.getActions().map((action) => { return action.payload.message; });
    expect(dispatchedActionMessages).toInclude('Text copied to clipboard');
  });

  it('saves the copied text in state', () => {
    const page = mount(<NewHostPage />);
    const method1Icon = page.find('.kolidecon-clipboard').first();

    method1Icon.simulate('click');

    expect(page.state().method1TextCopied).toEqual(true);
  });
});
