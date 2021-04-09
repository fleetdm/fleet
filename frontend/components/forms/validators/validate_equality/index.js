import { isEqual } from "lodash";

export default (actual, expected) => {
  return isEqual(actual, expected);
};
