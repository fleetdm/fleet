import React, { useCallback, useEffect, useState } from "react";
import { forEach } from "lodash";

import { IPlatformString, SUPPORTED_PLATFORMS } from "interfaces/platform";

import PlatformSelector from "components/PlatformSelector";

export interface IPlatformSelector {
  setSelectedPlatforms: (platforms: string[]) => void;
  getSelectedPlatforms: () => ("darwin" | "linux" | "windows")[];
  isAnyPlatformSelected: boolean;
  render: () => JSX.Element;
}

const usePlatformSelector = (
  platformContext: IPlatformString | null | undefined,
  baseClass = ""
): IPlatformSelector => {
  const [checkDarwin, setCheckDarwin] = useState<boolean>(false);
  const [checkWindows, setCheckWindows] = useState<boolean>(false);
  const [checkLinux, setCheckLinux] = useState<boolean>(false);

  const checksByPlatform: Record<string, boolean> = {
    darwin: checkDarwin,
    windows: checkWindows,
    linux: checkLinux,
  };

  const settersByPlatform: Record<string, (val: boolean) => void> = {
    darwin: setCheckDarwin,
    windows: setCheckWindows,
    linux: setCheckLinux,
  };

  const setSelectedPlatforms = (platformsToCheck: string[]) => {
    forEach(settersByPlatform, (setCheck, p) => {
      platformsToCheck.includes(p) ? setCheck(true) : setCheck(false);
    });
  };

  const getSelectedPlatforms = useCallback(() => {
    return SUPPORTED_PLATFORMS.filter((p) => checksByPlatform[p]);
  }, [checksByPlatform]);

  const isAnyPlatformSelected = Object.values(checksByPlatform).includes(true);

  useEffect(() => {
    if (platformContext === "") {
      setSelectedPlatforms(["darwin", "windows", "linux"]);
    }
    platformContext && setSelectedPlatforms(platformContext.split(","));
  }, [platformContext]);

  const render = useCallback(() => {
    return (
      <PlatformSelector
        baseClass={baseClass}
        checkDarwin={checkDarwin}
        checkWindows={checkWindows}
        checkLinux={checkLinux}
        setCheckDarwin={setCheckDarwin}
        setCheckWindows={setCheckWindows}
        setCheckLinux={setCheckLinux}
      />
    );
  }, [checkDarwin, checkWindows, checkLinux]);

  return {
    setSelectedPlatforms,
    getSelectedPlatforms,
    isAnyPlatformSelected,
    render,
  };
};

export default usePlatformSelector;
