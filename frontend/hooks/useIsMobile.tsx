import { useEffect, useState } from "react";

const MOBILE_BREAKPOINT = 768;

const useIsMobile = () => {
  const [isMobile, setIsMobile] = useState(false);

  useEffect(() => {
    const query = window.matchMedia(`(max-width: ${MOBILE_BREAKPOINT - 1}px)`);
    const updateMatch = (e: MediaQueryListEvent) => setIsMobile(e.matches);

    // Modern browsers
    if (query.addEventListener) {
      query.addEventListener("change", updateMatch);
    } else {
      // Fallback for older Safari
      query.addListener(updateMatch);
    }

    setIsMobile(query.matches);

    return () => {
      if (query.removeEventListener) {
        query.removeEventListener("change", updateMatch);
      } else {
        query.removeListener(updateMatch); // Deprecated but safe fallback
      }
    };
  }, []);

  return isMobile;
};

export default useIsMobile;
