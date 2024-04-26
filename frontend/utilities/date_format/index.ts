import { formatDistanceToNow } from "date-fns";

// eslint-disable-next-line import/prefer-default-export
export const uploadedFromNow = (date: string) => {
  return `Uploaded ${formatDistanceToNow(new Date(date))} ago`;
};
