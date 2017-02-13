import nock from 'nock';

beforeEach(() => {
  nock('http://localhost:8080')
    .post('/api/v1/kolide/targets', () => true)
    .reply(200, {
      targets_count: 1234,
      targets: [
        {
          id: 3,
          label: 'OS X El Capitan 10.11',
          name: 'osx-10.11',
          platform: 'darwin',
          target_type: 'hosts',
        },
      ],
    });
});

afterEach(nock.cleanAll);
