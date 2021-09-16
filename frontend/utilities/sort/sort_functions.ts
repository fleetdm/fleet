const caseInsensitiveAsc = (a: string, b: string): number => {
  a = a.toLowerCase();
  b = b.toLowerCase();

  if (b > a) {
    return 1;
  }
  if (b < a) {
    return -1;
  }
  return 0;
};

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
  if (parsedB > parsedA) {
    return 1;
  }
  if (parsedB < parsedA) {
    return -1;
  }
  return 0;
};

export default { caseInsensitiveAsc, dateStringsAsc };
