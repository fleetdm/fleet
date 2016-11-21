import expect from 'expect';

import helpers from './helpers';

const label1 = { id: 1, target_type: 'labels' };
const label2 = { id: 2, target_type: 'labels' };
const host1 = { id: 6, target_type: 'hosts' };
const host2 = { id: 5, target_type: 'hosts' };

describe('SelectTargetsDropdown - helpers', () => {
  describe('#formatSelectedTargetsForApi', () => {
    const { formatSelectedTargetsForApi } = helpers;

    it('splits targets into labels and hosts', () => {
      const targets = [host1, host2, label1, label2];

      expect(formatSelectedTargetsForApi(targets)).toEqual({
        hosts: [6, 5],
        labels: [1, 2],
      });
    });
  });
});
