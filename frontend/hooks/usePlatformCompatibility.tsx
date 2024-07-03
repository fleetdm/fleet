import React, { useCallback, useState } from "react";
import { useDebouncedCallback } from "use-debounce";

import { SupportedPlatform, SUPPORTED_PLATFORMS } from "interfaces/platform";
import { checkPlatformCompatibility } from "utilities/sql_tools";

import PlatformCompatibility from "components/PlatformCompatibility";

export interface IPlatformCompatibility {
  getCompatiblePlatforms: () => ("darwin" | "windows" | "linux" | "chrome")[];
  setCompatiblePlatforms: (sqlString: string) => void;
  render: () => JSX.Element;
}

const DEBOUNCE_DELAY = 300;

const usePlatformCompatibility = (): IPlatformCompatibility => {
  const [compatiblePlatforms, setCompatiblePlatforms] = useState<
    SupportedPlatform[] | null
  >(null);
  const [error, setError] = useState<Error | null>(null);

  const checkCompatibility = (sqlStr: string) => {
    const { platforms, error: compatibilityError } = checkPlatformCompatibility(
      sqlStr
    );
    setCompatiblePlatforms(platforms || []);
    setError(compatibilityError);
  };

  const debounceCompatiblePlatforms = useDebouncedCallback(
    (queryString: string) => {
      checkCompatibility(queryString);
    },
    DEBOUNCE_DELAY,
    { leading: true, trailing: true }
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
