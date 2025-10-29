const booleanAsc = (a: unknown, b: unknown): number => {
  if (!a && !!b) {
    return -1;
  }
  if (!!a && !b) {
    return 1;
  }
  return 0;
};

const caseInsensitiveAsc = (a: any, b: any): number => {
  a = typeof a === "string" ? a.toLowerCase() : a;
  b = typeof b === "string" ? b.toLowerCase() : b;

  if (a < b) {
    return -1;
  }
  if (a > b) {
    return 1;
  }
  return 0;
};

// Parses string representations of dates (e.g., "2021-02-21") and sorts in ascending order (i.e. "2021-02-01"
// appears before "2021-03-01").
// Values that are not parsable as dates return NaN and are sorted to appear before date-parsable values.
const dateStringsAsc = (a: string, b: string): number => {
  const parsedA = Date.parse(a);
  const parsedB = Date.parse(b);

  if (isNaN(parsedA) && isNaN(parsedB)) {
    return 0;
  }
  if (isNaN(parsedA)) {
    return -1;
  }
  if (isNaN(parsedB)) {
    return 1;
  }
  if (parsedA < parsedB) {
    return -1;
  }
  if (parsedA > parsedB) {
    return 1;
  }
  return 0;
};

const hasLength = (a: unknown[], b: unknown[]): number => {
  if (!a?.length && b?.length) {
    return -1;
  }
  if (a?.length && !b?.length) {
    return 1;
  }
  return 0;
};

const POLICY_STATUS_PRECEDENCE = ["actionRequired", "fail", "pass"];

const hostPolicyStatus = (a: unknown, b: unknown): number => {
  const [aI, bI] = [
    POLICY_STATUS_PRECEDENCE.indexOf(a as string),
    POLICY_STATUS_PRECEDENCE.indexOf(b as string),
  ];
  if (aI > bI) {
    return 1;
  }
  if (aI === bI) {
    return 0;
  }
  return -1;
};

export default {
  booleanAsc,
  caseInsensitiveAsc,
  dateStringsAsc,
  hasLength,
  hostPolicyStatus,
};
