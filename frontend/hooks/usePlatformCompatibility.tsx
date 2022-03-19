import React, { useCallback, useState } from "react";
import { useDebouncedCallback } from "use-debounce/lib";

import { IOsqueryPlatform, SUPPORTED_PLATFORMS } from "interfaces/platform";
import checkPlatformCompatibility from "utilities/sql_tools";

import PlatformCompatibility from "components/PlatformCompatibility";

export interface IPlatformCompatibility {
  getCompatiblePlatforms: () => ("darwin" | "windows" | "linux")[];
  setCompatiblePlatforms: (sqlString: string) => void;
  render: () => JSX.Element;
}

const DEBOUNCE_DELAY = 300;

const usePlatformCompatibility = (): IPlatformCompatibility => {
  const [compatiblePlatforms, setCompatiblePlatforms] = useState<
    IOsqueryPlatform[] | null
  >(null);
  const [error, setError] = useState<Error | null>(null);

  const tryCheckCompatibility = (sqlStr: string) => {
    try {
      const platforms = checkPlatformCompatibility(sqlStr);
      setCompatiblePlatforms(platforms);
      setError(null);
      return;
    } catch (err: unknown) {
      setError(new Error(`Invalid usage: ${err}`));
    }
  };

  const debounceCompatiblePlatforms = useDebouncedCallback(
    (queryString: string) => {
      tryCheckCompatibility(queryString);
    },
    DEBOUNCE_DELAY,
    { leading: true }
  );

  const getCompatiblePlatforms = useCallback(
    () => SUPPORTED_PLATFORMS.filter((p) => compatiblePlatforms?.includes(p)),
    [compatiblePlatforms]
  );

  const render = useCallback(() => {
    return (
      <PlatformCompatibility
        compatiblePlatforms={compatiblePlatforms}
        error={error}
      />
    );
  }, [compatiblePlatforms, error]);

  return {
    getCompatiblePlatforms,
    setCompatiblePlatforms: debounceCompatiblePlatforms,
    render,
  };
};

export default usePlatformCompatibility;
