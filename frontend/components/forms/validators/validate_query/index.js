import { Parser } from "utilities/node-sql-parser/sqlite";

export const EMPTY_QUERY_ERR = "Query text must be present";

const invalidQueryResponse = (message) => {
  return { valid: false, error: message };
};
const validQueryResponse = { valid: true, error: null };
const parser = new Parser();

export const validateQuery = (queryText) => {
  if (!queryText) {
    return invalidQueryResponse(EMPTY_QUERY_ERR);
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

export default { EMPTY_QUERY_ERR, validateQuery };
