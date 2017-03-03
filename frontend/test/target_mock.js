import nock from 'nock';

const defaultParams = {
  selected: {
    hosts: [],
    labels: [],
  },
};
const defaultResponse = {
  targets: {
    hosts: [],
    labels: [],
  },
};


export default (params = defaultParams, response = defaultResponse, responseStatus = 200) => {
  return nock('http://localhost:8080')
    .post('/api/v1/kolide/targets', JSON.stringify(params))
    .reply(responseStatus, response);
};
