import nock from "nock";

const createRequestMock = ({
  bearerToken,
  endpoint,
  method,
  params,
  responseStatus = 200,
  response,
}) => {
  const reqHeaders = { Authorization: `Bearer ${bearerToken}` };
  const host = "http://localhost:8080";
  const req = bearerToken ? nock(host) : nock(host, { reqHeaders });

  if (params) {
    return req[method](endpoint, JSON.stringify(params)).reply(
      responseStatus,
      response
    );
  }

  return req[method](endpoint).reply(responseStatus, response);
};

export default createRequestMock;
