import React, { useCallback, useEffect, useState } from "react";
import { forEach } from "lodash";

import {
  CommaSeparatedPlatformString,
  QUERYABLE_PLATFORMS,
  QueryablePlatform,
} from "interfaces/platform";
import { IPolicySoftwareToInstall } from "interfaces/policy";

import PlatformSelector from "components/PlatformSelector";

export interface IPlatformSelector {
  setSelectedPlatforms: (platforms: string[]) => void;
  getSelectedPlatforms: () => QueryablePlatform[];
  isAnyPlatformSelected: boolean;
  render: () => JSX.Element;
  disabled?: boolean;
  installSoftware?: IPolicySoftwareToInstall;
  currentTeamId?: number;
}

const usePlatformSelector = (
  platformContext: CommaSeparatedPlatformString | null | undefined,
  baseClass = "",
  disabled = false,
  installSoftware: IPolicySoftwareToInstall | undefined,
  currentTeamId: number | undefined
): IPlatformSelector => {
  const [checkDarwin, setCheckDarwin] = useState(false);
  const [checkWindows, setCheckWindows] = useState(false);
  const [checkLinux, setCheckLinux] = useState(false);
  const [checkChrome, setCheckChrome] = useState(false);

  const checksByPlatform: Record<string, boolean> = {
    darwin: checkDarwin,
    windows: checkWindows,
    linux: checkLinux,
    chrome: checkChrome,
  };

  const settersByPlatform: Record<string, (val: boolean) => void> = {
    darwin: setCheckDarwin,
    windows: setCheckWindows,
    linux: setCheckLinux,
    chrome: setCheckChrome,
  };

  const setSelectedPlatforms = (platformsToCheck: string[]) => {
    forEach(settersByPlatform, (setCheck, p) => {
      platformsToCheck.includes(p) ? setCheck(true) : setCheck(false);
    });
  };

  const getSelectedPlatforms = useCallback(() => {
    return QUERYABLE_PLATFORMS.filter((p) => checksByPlatform[p]);
  }, [checksByPlatform]);

  const isAnyPlatformSelected = Object.values(checksByPlatform).includes(true);

  useEffect(() => {
    if (platformContext === "") {
      setSelectedPlatforms(["darwin", "windows", "linux", "chrome"]);
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
        checkChrome={checkChrome}
        setCheckDarwin={setCheckDarwin}
        setCheckWindows={setCheckWindows}
        setCheckLinux={setCheckLinux}
        setCheckChrome={setCheckChrome}
        disabled={disabled}
        installSoftware={installSoftware}
        currentTeamId={currentTeamId}
      />
    );
  }, [checkDarwin, checkWindows, checkLinux, checkChrome, disabled]);

  return {
    setSelectedPlatforms,
    getSelectedPlatforms,
    isAnyPlatformSelected,
    render,
  };
};

export default usePlatformSelector;
