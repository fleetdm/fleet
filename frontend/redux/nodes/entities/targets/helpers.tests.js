import expect from 'expect';
import helpers from './helpers';

const stubbedApiResponse = {
  targets: {
    hosts: [
      {
        id: 3,
        label: 'OS X El Capitan 10.11',
        name: 'osx-10.11',
        platform: 'darwin',
      },
      {
        id: 4,
        label: 'Jason Meller\'s Macbook Pro',
        name: 'jmeller.local',
        platform: 'darwin',
      },
    ],
    labels: [
      {
        id: 4,
        label: 'All Macs',
        name: 'macs',
        count: 1234,
      },
    ],
  },
  selected_targets_count: 1234,
};

describe('targets - helpers', () => {
  describe('#appendTargetTypeToTargets', () => {
    const { appendTargetTypeToTargets } = helpers;

    it('combines the host and label targets, adding the target_type attribute', () => {
      expect(appendTargetTypeToTargets(stubbedApiResponse)).toEqual({
        targets: [
          {
            id: 3,
            label: 'OS X El Capitan 10.11',
            name: 'osx-10.11',
            platform: 'darwin',
            target_type: 'hosts',
          },
          {
            id: 4,
            label: 'Jason Meller\'s Macbook Pro',
            name: 'jmeller.local',
            platform: 'darwin',
            target_type: 'hosts',
          },
          {
            id: 4,
            label: 'All Macs',
            name: 'macs',
            count: 1234,
            target_type: 'labels',
          },
        ],
        selected_targets_count: 1234,
      });
    });
  });
});
