import sqliteParser from "sqlite-parser";
import { includes, some } from "lodash";

const BLACKLISTED_ACTIONS = [];
const invalidQueryErrorMessage = "Blacklisted query action";
const invalidQueryResponse = (message) => {
  return { valid: false, error: message };
};
const validQueryResponse = { valid: true, error: null };

export const validateQuery = (queryText) => {
  if (!queryText) {
    return invalidQueryResponse("Query text must be present");
  }

  try {
    const ast = sqliteParser(queryText);
    const { statement } = ast;
    const invalidQuery = some(statement, (obj) => {
      return includes(BLACKLISTED_ACTIONS, obj.variant.toLowerCase());
    });

    if (invalidQuery) {
      return invalidQueryResponse(invalidQueryErrorMessage);
    }

    return validQueryResponse;
  } catch (error) {
    return invalidQueryResponse(error.message);
  }
};

export default validateQuery;
