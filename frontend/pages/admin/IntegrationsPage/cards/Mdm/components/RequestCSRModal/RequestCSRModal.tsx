import React, { FormEvent, useState, useContext, useEffect } from "react";

import { AppContext } from "context/app";

import { IRequestCSRFormData } from "interfaces/request_csr";

import Button from "components/buttons/Button";
// @ts-ignore
import InputField from "components/forms/fields/InputField";
import DataError from "components/DataError";
import Icon from "components/Icon";
import Modal from "components/Modal";

const baseClass = "modal request-csr-modal";
interface IRequestCSRModalProps {
  onCancel: () => void;
}

interface IFormField {
  name: string;
  value: string;
}

const RequestCSRModal = ({ onCancel }: IRequestCSRModalProps): JSX.Element => {
  const { currentUser, config } = useContext(AppContext);

  const [formData, setFormData] = useState<IRequestCSRFormData>({
    email: currentUser?.email ?? "",
    orgName: config?.org_info?.org_name ?? "",
  });

  const [requestState, setRequestState] = useState<
    "loading" | "error" | "success" | "invalid" | undefined
  >(undefined);
  const [invalidMessage, setInvalidMessage] = useState<string>("");

  useEffect(() => {
    requestState === "invalid"
      ? setInvalidMessage("Email")
      : setInvalidMessage("");
  }, [requestState]);

  const { email, orgName } = formData;

  const onInputChange = ({ name, value }: IFormField) => {
    setFormData({ ...formData, [name]: value });
  };

  const onFormSubmit = (evt: FormEvent): void => {
    evt.preventDefault();

    // TODO: once API is finished, change below to actually call it
    setRequestState("loading");
    setTimeout(() => setRequestState("success"), 1000);
  };

  const RequestCSRSuccess = () => {
    return (
      <div className="success">
        <Icon name="success" size="extra-large" />
        <h2>You&apos;re almost there</h2>
        <p>
          Go to your <strong>{email}</strong> email to download your CSR.
          <br />
          Your APNs key and SCEP certificate and key will be downloaded in the
          browser. You&apos;ll need these later.
        </p>
        <Button onClick={onCancel}>Got it</Button>
      </div>
    );
  };

  const renderRequestCSRForm = () => {
    if (requestState === "success") {
      return <RequestCSRSuccess />;
    }
    if (requestState === "error") {
      return <DataError />;
    }
    return (
      <>
        <p>
          A CSR and key for APNs and a certificate and key for SCEP are required
          to connect Fleet to Apple Developer. Apple Inc. requires the following
          information. <br />
          <br />
          fleetdm.com will send your CSR to the below email. Your APNs key and
          SCEP certificate and key will be downloaded in the browser.
        </p>
        <form
          className={`${baseClass}__form`}
          onSubmit={onFormSubmit}
          autoComplete="off"
        >
          <div className="bottom-label">
            <InputField
              name="email"
              onChange={onInputChange}
              label="Email"
              parseTarget
              value={email}
              error={invalidMessage}
            />
            <p>
              Apple Inc. requires a work email (ex. name@your-organization.com).
            </p>
          </div>
          <InputField
            name="orgName"
            onChange={onInputChange}
            label="Organization name"
            parseTarget
            value={orgName}
          />
          <div className="modal-cta-wrap">
            <Button
              type="submit"
              variant="brand"
              isLoading={requestState === "loading"}
            >
              Request
            </Button>
            <Button onClick={onCancel} variant="inverse">
              Cancel
            </Button>
          </div>
        </form>
      </>
    );
  };

  return (
    <Modal title="Request" onExit={onCancel} className={baseClass}>
      {renderRequestCSRForm()}
    </Modal>
  );
};

export default RequestCSRModal;
