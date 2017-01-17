import expect from 'expect';

import helpers from 'pages/hosts/ManageHostsPage/helpers';
import { hostStub, labelStub } from 'test/stubs';

const macHost = { ...hostStub, id: 1, platform: 'darwin', status: 'mia' };
const ubuntuHost = { ...hostStub, id: 2, platform: 'ubuntu', status: 'offline' };
const windowsHost = { ...hostStub, id: 3, platform: 'windows', status: 'online' };
const allHosts = [macHost, ubuntuHost, windowsHost];

describe('ManageHostsPage - helpers', () => {
  describe('#filterHosts', () => {
    it('filters the all hosts label', () => {
      const allHostsLabel = { ...labelStub, type: 'all' };

      expect(helpers.filterHosts(allHosts, allHostsLabel)).toEqual(allHosts);
    });

    it('filters the platform label', () => {
      const platformLabel = { ...labelStub, type: 'platform', platform: 'ubuntu' };

      expect(helpers.filterHosts(allHosts, platformLabel)).toEqual([ubuntuHost]);
    });

    it('filters the status label', () => {
      const statusLabel = { ...labelStub, type: 'status', slug: 'online' };

      expect(helpers.filterHosts(allHosts, statusLabel)).toEqual([windowsHost]);
    });

    it('filters the custom label', () => {
      const customLabel = { ...labelStub, type: 'custom', host_ids: [1, 3] };

      expect(helpers.filterHosts(allHosts, customLabel)).toEqual([macHost, windowsHost]);
    });
  });
});
