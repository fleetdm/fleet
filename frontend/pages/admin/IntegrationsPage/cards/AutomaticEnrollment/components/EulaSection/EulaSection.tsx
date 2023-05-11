import React from "react";
import EulaUploader from "../EulaUploader/EulaUploader";

const baseClass = "eula-section";

interface IEulaSectionProps {}

const EulaSection = ({}: IEulaSectionProps) => {
  return (
    <div className={baseClass}>
      <EulaUploader />
    </div>
  );
};

export default EulaSection;
