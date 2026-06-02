/** Type guard that checks if a value is a valid number, optionally within a range. */
const isValidNumber = (
  value: unknown,
  min?: number,
  max?: number
): value is number => {
  if (typeof value !== "number" || isNaN(value)) {
    return false;
  }

  // If min or max is provided, check if the number is within the range
  return (
    (min === undefined || value >= min) && (max === undefined || value <= max)
  );
};

export default { isValidNumber };
