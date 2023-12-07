import { isEqual } from "lodash";

export default (actual: any, expected: any) => {
  return isEqual(actual, expected);
};
