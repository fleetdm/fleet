import React, { useState } from "react";

import Spinner from "components/Spinner/Spinner";
import SSOError from "components/MDM/SSOError";
import Button from "components/buttons/Button";

const baseClass = "mdm-apple-sso-callback-page";

const RedirectTo = ({ url }: { url: string }) => {
  window.location.href = url;
  return <Spinner />;
};

const EnrollmentGate = () => {
  const query = new URLSearchParams(location.search);
  const [showEULA, setShowEULA] = useState(query.has("eula_url"));
  const profileURL = query.get("profile_url");
  const eulaURL = query.get("eula_url") || "";

  if (!profileURL) {
    return <SSOError />;
  }

  if (showEULA) {
    return (
      <div className={`${baseClass}__eula-wrapper`}>
        <h3>Terms and conditions</h3>
        <iframe src={eulaURL} width="100%" title="eula" />
        <Button onClick={() => setShowEULA(false)} variant="oversized">
          Agree and continue
        </Button>
      </div>
    );
  }

  return <RedirectTo url={profileURL} />;
};

const MDMAppleSSOCallbackPage = () => {
  return (
    <div className={baseClass}>
      <EnrollmentGate />
    </div>
  );
};

export default MDMAppleSSOCallbackPage;
