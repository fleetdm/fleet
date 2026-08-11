import {
  differenceInDays,
  format,
  formatDistanceToNow,
  formatDistanceToNowStrict,
  isValid,
  parseISO,
} from "date-fns";
import { formatInTimeZone } from "date-fns-tz";

/** Below this many days ago, relative timestamps are expressed in days rather
 * than months (see issue #46965). */
const DAYS_BEFORE_MONTHS = 90;

interface ITimeAgoOptions {
  addSuffix?: boolean;
  includeSeconds?: boolean;
  /** Base the out-of-window result on formatDistanceToNowStrict rather than
   * formatDistanceToNow (e.g. "1 month" instead of "about 1 month"). */
  strict?: boolean;
}

/** Relative "time ago" string that shows days (e.g. "45 days ago") for
 * anything under 90 days old, switching to months only beyond that. This is
 * the single source of truth for the day/month cutoff so it stays consistent
 * everywhere; prefer it over calling date-fns' formatDistanceToNow directly.
 *
 * NOTE: Malformed dates will result in errors. This is expected "fail loudly"
 * behavior. */
export const timeAgo = (
  date: Date,
  {
    addSuffix = false,
    includeSeconds = false,
    strict = false,
  }: ITimeAgoOptions = {}
): string => {
  // date-fns switches to the month unit in the final seconds before day 30
  // (it rounds), whereas differenceInDays truncates and reports 29 there, so
  // 29 is the lower bound that reliably captures that last-day sliver.
  const days = Math.abs(differenceInDays(new Date(), date));
  if (days >= 29 && days < DAYS_BEFORE_MONTHS) {
    return formatDistanceToNowStrict(date, { unit: "day", addSuffix });
  }
  if (strict) {
    return formatDistanceToNowStrict(date, { addSuffix });
  }
  return formatDistanceToNow(date, { addSuffix, includeSeconds });
};

/** Utility to create a string from a date in this format:
  `Uploaded .... ago`
*/
export const uploadedFromNow = (date: string) => {
  return `Uploaded ${timeAgo(new Date(date), { addSuffix: true })}`;
};

/** Utility to create a string from a date in this format:
  `Added .... ago`
*/
export const addedFromNow = (date: string) => {
  return `Added ${timeAgo(new Date(date), { addSuffix: true })}`;
};

/** Utility to create a string from a date in this format:
    `.... ago`
*/
export const dateAgo = (date: string | Date) => {
  date = date instanceof Date ? date : new Date(date);
  return `${timeAgo(date, { addSuffix: true })}`;
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
