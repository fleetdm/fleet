import React from "react";
import { Params } from "react-router/lib/Router";

import SandboxGate from "components/Sandbox/SandboxGate";
import SandboxDemoMessage from "components/Sandbox/SandboxDemoMessage";
import OrgSettingsForm from "./components/OrgSettingsForm";

interface IAppSettingsPageProps {
  params: Params;
}

export const baseClass = "app-settings";

const AppSettingsPage = ({ params }: IAppSettingsPageProps): JSX.Element => {
  const { section } = params;

  return (
    <div className={`${baseClass} body-wrap`}>
      <p className={`${baseClass}__page-description`}>
        Set your organization information and configure SSO and SMTP
      </p>
      <SandboxGate
        fallbackComponent={() => (
          <SandboxDemoMessage
            message="Organization settings are only available in self-managed Fleet"
            utmSource="fleet-ui-organization-settings-page"
          />
        )}
      >
        <OrgSettingsForm section={section} />
      </SandboxGate>
    </div>
  );
};

export default AppSettingsPage;
