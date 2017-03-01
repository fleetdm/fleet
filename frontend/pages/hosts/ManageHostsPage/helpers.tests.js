import expect from 'expect';
import moment from 'moment';

import helpers from 'pages/hosts/ManageHostsPage/helpers';
import { hostStub, labelStub } from 'test/stubs';

const macHost = { ...hostStub, id: 1, platform: 'darwin', status: 'mia' };
const ubuntuHost = { ...hostStub, id: 2, platform: 'ubuntu', status: 'offline' };
const windowsHost = { ...hostStub, id: 3, platform: 'windows', status: 'online' };
// A (fixed) bug would have caused this host to be classified as new because
// the time difference was rounded down to 24 hours
const notNewHost = {
  ...hostStub,
  id: 4,
  platform: 'centos',
  status: 'online',
  created_at: moment().subtract(24, 'hours').subtract(40, 'minutes').toISOString(),
};
const newHost = {
  ...hostStub,
  id: 5,
  platform: 'centos',
  status: 'online',
  created_at: moment().subtract(10, 'hours'),
};
const allHosts = [macHost, ubuntuHost, windowsHost, notNewHost, newHost];

describe('ManageHostsPage - helpers', () => {
  describe('#filterHosts', () => {
    it('filters the all hosts label', () => {
      const allHostsLabel = { ...labelStub, type: 'all' };

      expect(helpers.filterHosts(allHosts, allHostsLabel)).toEqual(allHosts);
    });

    it('filters the new hosts', () => {
      const newHostsLabel = { ...labelStub, type: 'status', id: 'new' };

      expect(helpers.filterHosts(allHosts, newHostsLabel)).toEqual([newHost]);
    });

    it('filters the platform label', () => {
      const platformLabel = { ...labelStub, type: 'platform', host_ids: [2] };

      expect(helpers.filterHosts(allHosts, platformLabel)).toEqual([ubuntuHost]);
    });

    it('filters the status label', () => {
      const statusLabel = { ...labelStub, type: 'status', slug: 'online' };

      expect(helpers.filterHosts(allHosts, statusLabel)).toEqual([windowsHost, notNewHost, newHost]);
    });

    it('filters the custom label', () => {
      const customLabel = { ...labelStub, type: 'custom', host_ids: [1, 3] };

      expect(helpers.filterHosts(allHosts, customLabel)).toEqual([macHost, windowsHost]);
    });
  });
});
