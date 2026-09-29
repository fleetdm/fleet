import { isEqual } from "lodash";

export default (actual: unknown, expected: unknown) => {
  return isEqual(actual, expected);
};
