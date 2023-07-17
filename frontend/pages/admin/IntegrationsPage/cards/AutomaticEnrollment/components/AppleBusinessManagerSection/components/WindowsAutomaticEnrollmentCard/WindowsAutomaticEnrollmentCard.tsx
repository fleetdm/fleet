import React from "react";

import Card from "components/Card";
import Button from "components/buttons/Button";
import Icon from "components/Icon/Icon";

const baseClass = "windows-automatic-enrollment-card";

interface IWindowsAutomaticEnrollmentCardProps {}

const WindowsAutomaticEnrollmentCard = ({}: IWindowsAutomaticEnrollmentCardProps) => {
  return (
    <Card className={baseClass} color="gray">
      <h3>WIndows automatic enrollment</h3>
      <p>
        To use automatic enrollment for Windows hosts and Windows Autopilot you
        need to connect Fleet to Azure AD first.
      </p>
      <Button>
        Details <Icon name="chevron" direction="right" />
      </Button>
    </Card>
  );
};

export default WindowsAutomaticEnrollmentCard;
