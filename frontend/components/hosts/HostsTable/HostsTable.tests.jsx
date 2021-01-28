import React from 'react';
import { mount } from 'enzyme';

import { hostStub } from 'test/stubs';
import HostsTable from 'components/hosts/HostsTable';

describe('HostDetailsPage - component', () => {
  describe('Clicking on hostname', () => {
    it('Calls onHostClick when a hostname is clicked', () => {
      const hostClickSpy = jest.fn();

      const component = mount(
        <HostsTable
          hosts={[hostStub]}
          onHostClick={hostClickSpy}
        />,
      );

      const btn = component.find('Button');
      btn.simulate('click');

      expect(hostClickSpy).toHaveBeenCalledWith(hostStub);
    });
  });
});
