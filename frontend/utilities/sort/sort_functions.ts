const caseInsensitiveAsc = (a: string, b: string): number => {
  a = a.toLowerCase();
  b = b.toLowerCase();

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

export default { caseInsensitiveAsc, dateStringsAsc };
