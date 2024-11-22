export default (actual: any): boolean => {
  return actual !== null && actual !== undefined && actual.trim() !== "";
};
