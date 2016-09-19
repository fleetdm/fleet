import { debounce } from 'lodash';

const TIMEOUT = 1000; // only allow 1 click per second

export default (func) => {
  return debounce(func, TIMEOUT, {
    leading: true,
    trailing: false,
  });
};
