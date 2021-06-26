import nock from "nock";

const defaultParams = {
  query: "",
  query_id: 1,
  selected: {
    hosts: [],
    labels: [],
    teams: [],
  },
};
const defaultResponse = {
  targets: {
    hosts: [],
    labels: [],
    teams: [],
  },
};

export default (
  params = defaultParams,
  response = defaultResponse,
  responseStatus = 200
) => {
  return nock("http://localhost:8080")
    .post("/api/v1/fleet/targets", JSON.stringify(params))
    .reply(responseStatus, response);
};
