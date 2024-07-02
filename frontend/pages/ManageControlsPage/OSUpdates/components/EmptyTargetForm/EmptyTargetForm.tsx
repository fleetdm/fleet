import React, { useContext, useState } from "react";
import { isEmpty } from "lodash";

import { APP_CONTEXT_NO_TEAM_ID } from "interfaces/team";
import { NotificationContext } from "context/notification";
import configAPI from "services/entities/config";
import teamsAPI from "services/entities/teams";

// @ts-ignore
import InputField from "components/forms/fields/InputField";
import Button from "components/buttons/Button";
import validatePresence from "components/forms/validators/validate_presence";
import CustomLink from "components/CustomLink";

const baseClass = "empty-target-form";

interface IEmptyTargetFormProps {
  targetPlatform: string;
}

const EmptyTargetForm = ({ targetPlatform }: IEmptyTargetFormProps) => {
  return (
    <>
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
    </>
  );
};

export default EmptyTargetForm;
