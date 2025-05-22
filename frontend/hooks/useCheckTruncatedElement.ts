import { useLayoutEffect, useState } from "react";

/**
 * This hook checks if an element is truncated and returns a boolean value.
 */
// eslint-disable-next-line import/prefer-default-export
export const useCheckTruncatedElement = <T extends HTMLElement>(
  ref: React.RefObject<T>
) => {
  const [isTruncated, setIsTruncated] = useState(false);

  const updateIsTruncated = (element: HTMLElement) => {
    const { scrollWidth, clientWidth } = element;
    setIsTruncated(scrollWidth > clientWidth);
  };

  useLayoutEffect(() => {
    const resizeObserver = new ResizeObserver((entries) => {
      entries.forEach((entry) => {
        updateIsTruncated(entry.target as HTMLElement);
      });
    });
    const element = ref.current;
    if (element) {
      updateIsTruncated(element);
      resizeObserver.observe(ref.current as HTMLElement);
    }
    return () => {
      if (element) {
        resizeObserver.unobserve(element);
      }
    };
  }, [ref]);

  return isTruncated;
};
