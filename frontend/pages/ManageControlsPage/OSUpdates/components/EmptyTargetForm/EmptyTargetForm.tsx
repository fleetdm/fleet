import React from "react";

import CustomLink from "components/CustomLink";

const baseClass = "empty-target-form";

interface IEmptyTargetFormProps {
  targetPlatform: string;
}

const EmptyTargetForm = ({ targetPlatform }: IEmptyTargetFormProps) => {
  return (
    <div className={baseClass}>
      <p>
        <b>{targetPlatform} updates are coming soon.</b>
      </p>
      <p>
        Need to remotely encourage installation of {targetPlatform} updates?{" "}
        <CustomLink
          url="https://www.fleetdm.com/support"
          text="Let us know"
          newTab
        />
      </p>
    </div>
  );
};

export default EmptyTargetForm;
