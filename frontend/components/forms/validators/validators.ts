import yaml, { YAMLException } from "js-yaml";
import { isFQDN, isIP, isPort } from "validator";
import isEmail from "validator/lib/isEmail";
import isURL, { IsURLOptions } from "validator/lib/isURL";
import isUUID from "validator/lib/isUUID";

import { astify } from "utilities/osquery_sql_parser";
import type { ParserSyntaxError } from "utilities/osquery_sql_parser";

// Naming convention:
//   isValid*  → boolean predicate ("is this thing valid?")
//   validate* → returns structured { valid/isValid, error } for richer feedback

// ---------- isEqual ----------
// Re-exported from lodash so equality checks live alongside the other form
// predicates and are easy to grep for. Same function as `lodash.isEqual`.
export { isEqual } from "lodash";

// ---------- isPresent ----------
// Treats empty / whitespace-only strings as absent. The whitespace trim is the
// only reason this exists vs. plain `!!value`.
export const isPresent = (actual: unknown): boolean => {
  return !!actual && (typeof actual !== "string" || actual.trim() !== "");
};

// ---------- isValidEmail ----------
export const isValidEmail = (email: string): boolean => {
  return isEmail(email);
};

// ---------- isValidURL ----------
interface IValidUrl {
  url: string;
  /** Validate against the protocols specified. */
  protocols?: ("http" | "https" | "file")[];
  allowLocalHost?: boolean;
}

export const isValidURL = ({
  url,
  protocols,
  allowLocalHost = false,
}: IValidUrl): boolean => {
  const options: Partial<IsURLOptions> = {
    protocols,
    require_protocol: !!protocols?.length,
    require_tld: !allowLocalHost,
  };

  if (protocols?.includes("file")) {
    options.allow_underscores = true;
    options.allow_protocol_relative_urls = true;
    options.require_host = false;
  }

  return isURL(url, options);
};

// ---------- isValidUuid ----------
export const isValidUuid = (val: string): boolean => {
  return isUUID(val);
};

// ---------- isValidHostname ----------
// Accepts FQDN, IPv4, IPv6 (with and without brackets), localhost, and any of
// the above with a trailing :port.
export const isValidHostname = (addr: string): boolean => {
  const fqdnOpts = { require_tld: false };
  const isValid = isFQDN(addr, fqdnOpts) || isIP(addr);
  if (isValid) return true;

  const lastColonIndex = addr.lastIndexOf(":");
  if (lastColonIndex <= 0) return false;

  const port = addr.substring(lastColonIndex + 1);
  let host = addr.substring(0, lastColonIndex);

  if (host.startsWith("[") && host.endsWith("]")) {
    host = host.slice(1, -1);
  }

  return isPort(port) && (isFQDN(host, fqdnOpts) || isIP(host));
};

// ---------- validatePassword ----------
export type ValidPasswordErrorCode =
  | ""
  | "too_short"
  | "too_long"
  | "invalid_format";

export interface IValidPasswordResult {
  isValid: boolean;
  error: string;
  error_code: ValidPasswordErrorCode;
}

const LETTER_PRESENT = /[a-z]+/i;
const NUMBER_PRESENT = /[0-9]+/;
const SYMBOL_PRESENT = /\W+/;

export const validatePassword = (password = ""): IValidPasswordResult => {
  let error = "";
  let error_code: ValidPasswordErrorCode = "";
  if (password.length < 12) {
    error = "Password must be at least 12 characters";
    error_code = "too_short";
  } else if (password.length > 48) {
    error = "Password is over the character limit";
    error_code = "too_long";
  } else if (
    !(
      LETTER_PRESENT.test(password) &&
      NUMBER_PRESENT.test(password) &&
      SYMBOL_PRESENT.test(password)
    )
  ) {
    error = "Password must meet the criteria below";
    error_code = "invalid_format";
  }
  return { isValid: !error, error, error_code };
};

// ---------- validateYaml ----------
export interface IYamlSyntaxError {
  name: string;
  reason: string;
  line: number;
}

export type IValidateYamlError = string | IYamlSyntaxError | null;

export interface IValidateYamlResult {
  isValid: boolean;
  error: IValidateYamlError;
}

export const validateYaml = (yamlText?: string): IValidateYamlResult => {
  if (!yamlText) {
    return { isValid: false, error: "YAML text must be present" };
  }

  try {
    yaml.load(yamlText);
    return { isValid: true, error: null };
  } catch (error) {
    if (error instanceof YAMLException) {
      return {
        isValid: false,
        error: {
          name: "Syntax Error",
          reason: error.reason,
          line: error.mark.line,
        },
      };
    }
    return {
      isValid: false,
      error: error instanceof Error ? error.message : String(error),
    };
  }
};

// ---------- validateQuery ----------
export const EMPTY_QUERY_ERR = "Query text must be present";
export const INVALID_SYNTAX_ERR = "Syntax error. Please review before saving.";

export const invalidSyntaxOnLineErr = (line: number, column: number) =>
  `Syntax error on line ${line}, column ${column}. Please review before saving.`;
export const expectedSelectErr = (line: number) =>
  `Expected a SELECT statement on line ${line}.`;

export interface IValidateQueryResult {
  isValid: boolean;
  error: string | null;
}

const describeSyntaxError = (err: unknown): string => {
  const { location, expected } = err as Partial<ParserSyntaxError>;
  if (!location) {
    return INVALID_SYNTAX_ERR;
  }

  // When the parser would have accepted a SELECT keyword and, alternatively,
  // a semicolon or the end of the input, the error is at a statement position
  // — meaning the input is something other than the SELECT statements osquery
  // executes (e.g. INSERT, DROP, a lone comment, or free text) rather than a
  // typo inside one. SELECT alone isn't enough: it is also expected inside a
  // just-opened subquery, where the generic located message is the right one.
  const expectsSelect = (expected ?? []).some(
    (e) => e.type === "literal" && e.text?.toUpperCase() === "SELECT"
  );
  const expectsStatementEnd = (expected ?? []).some(
    (e) => e.type === "end" || (e.type === "literal" && e.text === ";")
  );
  if (expectsSelect && expectsStatementEnd) {
    return expectedSelectErr(location.start.line);
  }

  return invalidSyntaxOnLineErr(location.start.line, location.start.column);
};

export const validateQuery = (queryText?: string): IValidateQueryResult => {
  if (!queryText?.trim()) {
    return { isValid: false, error: EMPTY_QUERY_ERR };
  }

  try {
    astify(queryText);
    return { isValid: true, error: null };
  } catch (err) {
    return { isValid: false, error: describeSyntaxError(err) };
  }
};
