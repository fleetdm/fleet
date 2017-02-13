import nock from 'nock';

const defaultParams = {
  selected: {
    hosts: [],
    labels: [],
  },
};

export default (params = defaultParams) => {
  nock('http://localhost:8080')
    .post('/api/v1/kolide/targets', JSON.stringify(params))
    .reply(200, {
      targets: [],
    });
};
