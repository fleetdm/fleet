import React from "react";
import classnames from "classnames";
import Slider from "components/forms/fields/Slider";
import { isIPadOrIPhone } from "interfaces/platform";

const baseClass = "software-deploy-slider";

interface ISoftwareDeploySliderProps {
  deploySoftware: boolean;
  onToggleDeploySoftware: () => void;
  platform?: string;
  isExePackage?: boolean;
  isTarballPackage?: boolean;
  isScriptPackage?: boolean;
  disableOptions?: boolean;
  className?: string;
}

const SoftwareDeploySlider = ({
  deploySoftware,
  onToggleDeploySoftware,
  platform,
  isExePackage,
  isTarballPackage,
  isScriptPackage,
  disableOptions = false,
  className,
}: ISoftwareDeploySliderProps) => {
  // This should never happen with platform gate in parent component
  const isPlatformIosOrIpados = isIPadOrIPhone(platform || "") || false;

  const isDeploySoftwareDisabled =
    disableOptions ||
    isPlatformIosOrIpados ||
    isExePackage ||
    isTarballPackage ||
    isScriptPackage;

  /** Tooltip only shows when enabled or for exe/tar.gz/sh/ps1 packages */
  const showDeploySoftwareTooltip =
    !isDeploySoftwareDisabled ||
    isExePackage ||
    isTarballPackage ||
    isScriptPackage;

  const getDeploySoftwareLabelTooltip = (): JSX.Element => {
    if (isExePackage || isTarballPackage) {
      return (
        <>
          Fleet can&apos;t create a policy to detect existing installations for{" "}
          {isExePackage ? ".exe packages" : ".tar.gz archives"}. To
          automatically install{" "}
          {isExePackage ? ".exe packages" : ".tar.gz archives"}, add a custom
          policy and enable the install software automation on the{" "}
          <b>Policies</b> page.
        </>
      );
    }

    if (isScriptPackage) {
      return (
        <>
          Fleet can&apos;t create a policy to detect existing installations of
          script-only packages. To automatically install these packages, add a
          custom policy and enable the install software automation on the{" "}
          <b>Policies</b> page.
        </>
      );
    }
    return <>Automatically install only on hosts missing this software.</>;
  };

  const deploySoftwareLabelTooltip = showDeploySoftwareTooltip
    ? getDeploySoftwareLabelTooltip()
    : undefined;

  return (
    <div
      className={classnames(`${baseClass}__deploy-slider-container`, className)}
    >
      <Slider
        value={deploySoftware}
        onChange={onToggleDeploySoftware}
        activeText="Deploy"
        inactiveText="Deploy"
        className={`${baseClass}__deploy-slider`}
        labelTooltip={deploySoftwareLabelTooltip}
        disabled={isDeploySoftwareDisabled}
      />
    </div>
  );
};

export default SoftwareDeploySlider;
