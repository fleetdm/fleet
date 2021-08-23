const caseInsensitiveAsc = (a: string, b: string): number => {
  a = a.toLowerCase();
  b = b.toLowerCase();

  console.log(a, b);

  if (b > a) {
    return 1;
  }
  if (b < a) {
    return -1;
  }
  return 0;
};

export default { caseInsensitiveAsc };
