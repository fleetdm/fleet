export const isValidNumber = (
  value: any,
  min?: number,
  max?: number
): value is number => {
  // Check if the value is a number and not NaN
  const isNumber = typeof value === "number" && !isNaN(value);

  // If min or max is provided, check if the number is within the range
  const withinRange =
    (min === undefined || value >= min) && (max === undefined || value <= max);

  return isNumber && withinRange;
};
