import React, { useState } from "react";
import { WithRouterProps } from "react-router";

import Spinner from "components/Spinner";
import Button from "components/buttons/Button";

import WindowsEulaPdf from "../../../assets/windows-end-user-license-agreement.pdf";

const baseClass = "mdm-windows-sso-callback-page";

const RedirectTo = ({ url }: { url: string }) => {
  window.location.href = url;
  return <Spinner />;
};

type MDMWindowsSSOCallbackPageProps = WithRouterProps<
  null,
  { redirect_uri: string }
>;

const MDMWindowsSSOCallbackPage = ({
  location,
}: MDMWindowsSSOCallbackPageProps) => {
  const [showEULA, setShowEULA] = useState(true);
  const { redirect_uri } = location.query;

  return (
    <div className={baseClass}>
      {showEULA ? (
        <div className={`${baseClass}__eula-wrapper`}>
          <h3>Terms and conditions</h3>
          <iframe src={WindowsEulaPdf} width="100%" title="eula" />
          <Button
            onClick={() => setShowEULA(false)}
            variant="oversized"
            className={`${baseClass}__agree-btn`}
          >
            Agree and continue
          </Button>
        </div>
      ) : (
        <RedirectTo url={`${redirect_uri}?IsAccepted=true`} />
      )}
    </div>
  );
};

export default MDMWindowsSSOCallbackPage;
