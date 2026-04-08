import { format, formatDistanceToNow, isValid, parseISO } from "date-fns";
import { formatInTimeZone } from "date-fns-tz";

/** Utility to create a string from a date in this format:
  `Uploaded .... ago`
*/
export const uploadedFromNow = (date: string) => {
  // NOTE: Malformed dates will result in errors. This is expected "fail loudly" behavior.
  return `Uploaded ${formatDistanceToNow(new Date(date), { addSuffix: true })}`;
};

/** Utility to create a string from a date in this format:
  `Added .... ago`
*/
export const addedFromNow = (date: string) => {
  // NOTE: Malformed dates will result in errors. This is expected "fail loudly" behavior.
  return `Added ${formatDistanceToNow(new Date(date), { addSuffix: true })}`;
};

/** Utility to create a string from a date in this format:
    `.... ago`
*/
export const dateAgo = (date: string | Date) => {
  // NOTE: Malformed dates will result in errors. This is expected "fail loudly" behavior.
  date = date instanceof Date ? date : new Date(date);
  return `${formatDistanceToNow(date, { addSuffix: true })}`;
};

/**
 * returns a date in the format of 'MonthName Date, Year'
 * @example "January 01, 2024"
 */
export const monthDayYearFormat = (date: string) => {
  return formatInTimeZone(parseISO(date), "UTC", "MMMM d, yyyy");
};

/**
 * Formats a date as abbreviated month, day, and time
 * @example "Mar 20, 1:35 PM"
 * @returns formatted date string, or empty string if invalid
 */
export const monthDayTimeFormat = (isoDate: string) => {
  const date = parseISO(isoDate);
  if (!isValid(date)) {
    return "";
  }
  return format(date, "MMM d, p");
};
