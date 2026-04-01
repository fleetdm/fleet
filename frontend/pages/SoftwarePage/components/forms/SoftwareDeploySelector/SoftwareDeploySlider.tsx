import React from "react";
import classnames from "classnames";
import Slider from "components/forms/fields/Slider";

const baseClass = "software-deploy-slider";

const LABEL_TOOLTIP =
  "Automatically install only on hosts missing this software.";

interface ISoftwareDeploySliderProps {
  deploySoftware: boolean;
  onToggleDeploySoftware: () => void;
  className?: string;
}

const SoftwareDeploySlider = ({
  deploySoftware,
  onToggleDeploySoftware,
  className,
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
      />
    </div>
  );
};

export default SoftwareDeploySlider;
