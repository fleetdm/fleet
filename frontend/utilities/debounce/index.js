import { debounce } from "lodash";

const DEFAULT_TIMEOUT = 1000; // 1 function execution per second by default

export default (func, options = {}) => {
  const {
    leading = true,
    trailing = false,
    timeout = DEFAULT_TIMEOUT,
  } = options;

  return debounce(func, timeout, {
    leading,
    trailing,
  });
};
