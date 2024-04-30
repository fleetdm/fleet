// import sendRequest from "services";
import { sendRequest } from "services/mock_service/service/service";
import endpoints from "utilities/endpoints";

export default {
  getHumanInterpretationFromSQL: (sql: string): Promise<any> => {
    const { AI_AUTOFILL_POLICIES } = endpoints;

    return sendRequest("POST", AI_AUTOFILL_POLICIES, sql);
  },
};
