// Addition that empty strings return as false
export default (actual: any): boolean => {
  return !!actual && (typeof actual !== "string" || actual.trim() !== "");
};
