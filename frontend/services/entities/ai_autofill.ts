// import sendRequest from "services";
import { sendRequest } from "services/mock_service/service/service";
import { buildQueryStringFromParams } from "utilities/url";

export default {
  getHumanInterpretationFromSQL: (sql: string): Promise<any> => {
    // const { TODO_ENDPOINT } = endpoints;

    const queryParams = {
      sql,
    };

    const queryString = buildQueryStringFromParams(queryParams);

    const path = `TODO`;

    return sendRequest("POST", path, sql);
  },
};
