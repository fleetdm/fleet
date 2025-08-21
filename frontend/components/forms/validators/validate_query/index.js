import { Parser } from "utilities/node-sql-parser/sqlite";
import { includes, some } from "lodash";

const invalidQueryResponse = (message) => {
  return { valid: false, error: message };
};
const validQueryResponse = { valid: true, error: null };
const parser = new Parser();

export const validateQuery = (queryText) => {
  if (!queryText) {
    return invalidQueryResponse("Query text must be present");
  }

  try {
    parser.astify(queryText, { database: "sqlite" });
    return validQueryResponse;
  } catch (error) {
    return invalidQueryResponse(
      "There is a syntax error in your query; please resolve in order to save."
    );
  }
};

export default validateQuery;
