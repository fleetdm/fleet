import { useLayoutEffect, useState } from "react";

/**
 * This hook checks if an element is truncated and returns a boolean value.
 */
// eslint-disable-next-line import/prefer-default-export
export const useCheckTruncatedElement = <T extends HTMLElement>(
  ref: React.RefObject<T>
) => {
  const [isTruncated, setIsTruncated] = useState(false);

  useLayoutEffect(() => {
    const element = ref.current;
    function updateIsTruncated() {
      if (element) {
        const { scrollWidth, clientWidth } = element;
        setIsTruncated(scrollWidth > clientWidth);
      }
    }
    window.addEventListener("resize", updateIsTruncated);
    updateIsTruncated();
    return () => window.removeEventListener("resize", updateIsTruncated);
  }, [ref]);

  return isTruncated;
};
