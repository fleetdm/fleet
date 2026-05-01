// Addition that empty strings return as false
export default (actual: unknown): boolean => {
  return !!actual && (typeof actual !== "string" || actual.trim() !== "");
};
