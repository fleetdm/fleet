import React from 'react';
import expect from 'expect';
import { mount } from 'enzyme';
import { noop } from 'lodash';

import { ManageHostsPage } from 'pages/hosts/ManageHostsPage/ManageHostsPage';
import { createAceSpy, stubbedOsqueryTable } from 'test/helpers';

describe('ManageHostsPage - component', () => {
  const props = {
    allHostLabels: [],
    dispatch: noop,
    hosts: [],
    hostPlatformLabels: [],
    hostStatusLabels: [],
    labels: [],
    selectedOsqueryTable: stubbedOsqueryTable,
  };

  beforeEach(() => {
    createAceSpy();
  });

  it('renders a HostSidePanel when not adding a new label', () => {
    const page = mount(<ManageHostsPage {...props} />);

    expect(page.find('HostSidePanel').length).toEqual(1);
  });

  it('renders a QuerySidePanel when adding a new label', () => {
    const page = mount(<ManageHostsPage {...props} />);

    page.setState({ isAddLabel: true });

    expect(page.find('QuerySidePanel').length).toEqual(1);
  });
});
