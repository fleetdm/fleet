import { Parser } from "node-sql-parser";

export const EMPTY_QUERY_ERR = "Query text must be present";
export const INVALID_SYNTAX_ERR = "Syntax error. Please review before saving.";

const invalidQueryResponse = (message: string) => {
  return { valid: false, error: message };
};
const validQueryResponse = { valid: true, error: null };
const parser = new Parser();

export const validateQuery = (queryText?: string) => {
  if (!queryText?.trim()) {
    return invalidQueryResponse(EMPTY_QUERY_ERR);
  }

  try {
    parser.astify(queryText, { database: "sqlite" });
    return validQueryResponse;
  } catch (error) {
    return invalidQueryResponse(INVALID_SYNTAX_ERR);
  }
};
