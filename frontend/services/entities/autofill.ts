// import sendRequest from "services";
import { sendRequest } from "services/mock_service/service/service";
import endpoints from "utilities/endpoints";

export type IAutofillPolicy = {
  description: string;
  resolution: string;
};

export default {
  getPolicyInterpretationFromSQL: (sql: string): Promise<IAutofillPolicy> => {
    const { AUTOFILL_POLICIES } = endpoints;

    return sendRequest("POST", AUTOFILL_POLICIES, sql);
  },
};
