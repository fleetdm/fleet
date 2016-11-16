import React from 'react';
import expect from 'expect';
import { mount } from 'enzyme';

import { connectedComponent, reduxMockStore } from 'test/helpers';
import ConnectedRegistrationPage, { RegistrationPage } from 'pages/RegistrationPage/RegistrationPage';

const baseStore = {
  app: {},
};

describe('RegistrationPage - component', () => {
  it('displays the Kolide background triangles', () => {
    const mockStore = reduxMockStore(baseStore);

    mount(connectedComponent(ConnectedRegistrationPage, { mockStore }));

    expect(mockStore.getActions()).toInclude({
      type: 'SHOW_BACKGROUND_IMAGE',
    });
  });

  it('renders the RegistrationForm', () => {
    const mockStore = reduxMockStore(baseStore);
    const page = mount(connectedComponent(ConnectedRegistrationPage, { mockStore }));

    expect(page.find('RegistrationForm').length).toEqual(1);
  });

  it('sets the page # to 1', () => {
    const page = mount(<RegistrationPage />);

    expect(page.state()).toInclude({ page: 1 });
  });

  it('displays the setup breadcrumbs', () => {
    const mockStore = reduxMockStore(baseStore);
    const page = mount(connectedComponent(ConnectedRegistrationPage, { mockStore }));

    expect(page.find('Breadcrumbs').length).toEqual(1);
  });

  describe('#onSetPage', () => {
    it('sets state to the page number', () => {
      const page = mount(<RegistrationPage />);
      page.node.onSetPage(3);

      expect(page.state()).toInclude({ page: 3 });
    });
  });
});
