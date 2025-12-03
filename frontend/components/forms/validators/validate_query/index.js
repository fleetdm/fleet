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
    return invalidQueryResponse("Syntax error. Please review before saving.");
  }
};

export default { EMPTY_QUERY_ERR, validateQuery };
