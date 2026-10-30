import classnames from "classnames";
import React from "react";

import Slider from "components/forms/fields/Slider";

const baseClass = "software-deploy-slider";

const LABEL_TOOLTIP =
  "Automatically install only on hosts missing this software.";

interface ISoftwareDeploySliderProps {
  deploySoftware: boolean;
  onToggleDeploySoftware: () => void;
  className?: string;
  disabled?: boolean;
}

const SoftwareDeploySlider = ({
  deploySoftware,
  onToggleDeploySoftware,
  className,
  disabled,
}: ISoftwareDeploySliderProps) => {
  const sliderClassNames = classnames(`${baseClass}__container`, className);

  return (
    <div className={sliderClassNames}>
      <Slider
        value={deploySoftware}
        onChange={onToggleDeploySoftware}
        activeText="Deploy"
        inactiveText="Deploy"
        className={`${baseClass}__deploy-slider`}
        labelTooltip={LABEL_TOOLTIP}
        disabled={disabled}
      />
    </div>
  );
};

export default SoftwareDeploySlider;
