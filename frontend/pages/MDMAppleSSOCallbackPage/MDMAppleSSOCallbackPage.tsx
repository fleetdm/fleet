import React, { useState } from "react";

import endpoints from "utilities/endpoints";

import Spinner from "components/Spinner/Spinner";
import SSOError from "components/MDM/SSOError";
import Button from "components/buttons/Button";

const baseClass = "mdm-apple-sso-callback-page";

const RedirectTo = ({ url }: { url: string }) => {
  window.location.href = url;
  return <Spinner />;
};

// appConfig.ServerSettings.ServerURL+"/api/latest/fleet/mdm/apple/setup/eula/"
// /api/mdm/apple/enroll?token=

const EnrollmentGate = () => {
  const query = new URLSearchParams(location.search);
  const [showEULA, setShowEULA] = useState(query.has("eula_token"));
  const profileToken = query.get("profile_token");
  const eulaToken = query.get("eula_token") || "";

  if (!profileToken) {
    return <SSOError />;
  }

  if (showEULA) {
    return (
      <div className={`${baseClass}__eula-wrapper`}>
        <h3>Terms and conditions</h3>
        <iframe
          src={`/api/${endpoints.MDM_APPLE_EULA_FILE(eulaToken)}`}
          width="100%"
          title="eula"
        />
        <Button onClick={() => setShowEULA(false)} variant="oversized">
          Agree and continue
        </Button>
      </div>
    );
  }

  return (
    <RedirectTo url={endpoints.MDM_APPLE_ENROLLMENT_PROFILE(profileToken)} />
  );
};

const MDMAppleSSOCallbackPage = () => {
  return (
    <div className={baseClass}>
      <EnrollmentGate />
    </div>
  );
};

export default MDMAppleSSOCallbackPage;
