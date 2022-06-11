import { useEffect, useRef } from "react";
import { isEqual } from "lodash";

/**
 *
 * @param fn Anonymous function passing into the hook
 * @param deps What dependencies to watch for changes
 *
 * Adapted from https://betterprogramming.pub/how-to-use-the-react-hook-usedeepeffect-815818c0ad9d,
 * this hook does a deeper check for changes within objects and arrays
 */

export const useDeepEffect = (fn: () => void, deps: Array<any>) => {
  const isFirst = useRef(true);
  const prevDeps = useRef(deps);

  useEffect(() => {
    const isSame = prevDeps.current.every((obj, index) =>
      isEqual(obj, deps[index])
    );

    if (isFirst.current || !isSame) {
      fn();
    }

    isFirst.current = false;
    prevDeps.current = deps;
  }, [deps, fn]);
};

export default useDeepEffect;
