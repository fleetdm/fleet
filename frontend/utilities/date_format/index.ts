import { format, formatDistanceToNow, intlFormat } from "date-fns";

/** Utility to create a string from a date in this format:
  `Uploaded .... ago`
*/
export const uploadedFromNow = (date: string) => {
  // NOTE: Malformed dates will result in errors. This is expected "fail loudly" behavior.
  return `Uploaded ${formatDistanceToNow(new Date(date))} ago`;
};

/** Utility to create a string from a date in this format:
    `.... ago`
*/
export const dateAgo = (date: string) => {
  // NOTE: Malformed dates will result in errors. This is expected "fail loudly" behavior.
  return `${formatDistanceToNow(new Date(date))} ago`;
};

/**
 * returns a date in the format of 'MonthName Date, Year'
 * @example "January 01, 2024"
 */
export const monthDayYearFormat = (date: string) => {
  return format(date, "MMMM d, yyyy");
};
