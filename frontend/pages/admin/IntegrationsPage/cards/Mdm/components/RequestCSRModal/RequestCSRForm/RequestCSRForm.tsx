import React, { FormEvent, useState, useContext } from "react";

import { AppContext } from "context/app";

import { IRequestCSRFormData } from "interfaces/request_csr";

import Button from "components/buttons/Button";
// @ts-ignore
import InputField from "components/forms/fields/InputField";
import Spinner from "components/Spinner";
import requestCSR from "services/entities/mdm_csr";

const baseClass = "request-csr";

interface IRequestCSRFormProps {
  onCancel: () => void;
  testingConnection?: boolean;
}

interface IFormField {
  name: string;
  value: string;
}

const RequestCSRForm = ({
  onCancel,
  testingConnection,
}: IRequestCSRFormProps): JSX.Element => {
  const { currentUser, config } = useContext(AppContext);

  const [formData, setFormData] = useState<IRequestCSRFormData>({
    email: currentUser?.email || "",
    orgName: config?.org_info?.org_name || "",
  });

  const [requestState, setRequestState] = useState<
    "loading" | "error" | "success" | undefined
  >(undefined);

  const { email, orgName } = formData;

  // destructure change event to its name and value
  const onInputChange = ({ name, value }: IFormField) => {
    setFormData({ ...formData, [name]: value });
  };

  const onFormSubmit = (evt: FormEvent) => {
    evt.preventDefault();
    return requestCSR({ email, orgName }, setRequestState);
  };

  return (
    <>
      {testingConnection ? (
        <div className={`${baseClass}__testing-connection`}>
          <b>Testing connection</b>
          <Spinner />
        </div>
      ) : (
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
      )}
    </>
  );
};

export default RequestCSRForm;
