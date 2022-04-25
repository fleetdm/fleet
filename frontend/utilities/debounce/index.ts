import { debounce } from "lodash";

interface IOptions {
  leading: boolean;
  trailing: boolean;
  timeout: number;
}

const DEFAULT_TIMEOUT = 1000; // 1 function execution per second by default

export default (
  func: (...args: any[]) => any,
  options: IOptions = {
    leading: true,
    trailing: false,
    timeout: DEFAULT_TIMEOUT,
  }
) => {
  const { leading, trailing, timeout } = options;

  return debounce(func, timeout, {
    leading,
    trailing,
  });
};
