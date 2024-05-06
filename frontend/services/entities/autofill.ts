import sendRequest from "services";
import endpoints from "utilities/endpoints";

export type IAutofillPolicy = {
  description: string;
  resolution: string;
};

export default {
  getPolicyInterpretationFromSQL: (sql: string): Promise<IAutofillPolicy> => {
    const { AUTOFILL_POLICY } = endpoints;

    // API expects JSON object with key "sql"
    const data = JSON.stringify({ sql });

    return sendRequest("POST", AUTOFILL_POLICY, data);
  },
};
