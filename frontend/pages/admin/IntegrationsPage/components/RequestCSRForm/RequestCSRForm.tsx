import React, { FormEvent, useState } from "react";

import { IRequestCSRFormData } from "interfaces/request_csr";

import Button from "components/buttons/Button";
// @ts-ignore
import InputField from "components/forms/fields/InputField";
import Spinner from "components/Spinner";

const baseClass = "request-csr-form";

interface IRequestCSRFormProps {
  onCancel: () => void;
  onSubmit: (formData: IRequestCSRFormData, destination: string) => void;
  userEmail: string;
  currentOrgName: string;
  requestCSRDestination: string;
  testingConnection?: boolean;
}

interface IFormField {
  name: string;
  value: string;
}

const RequestCSRForm = ({
  onCancel,
  onSubmit,
  userEmail,
  currentOrgName,
  requestCSRDestination,
  testingConnection,
}: IRequestCSRFormProps): JSX.Element => {
  // define state
  const [formData, setFormData] = useState<IRequestCSRFormData>({
    email: userEmail,
    orgName: currentOrgName,
  });
  const [destination, setDestination] = useState(requestCSRDestination);

  const { email, orgName } = formData;

  const onInputChange = ({ name, value }: IFormField) => {
    setFormData({ ...formData, [name]: value });
  };

  const onFormSubmit = (evt: FormEvent): void => {
    evt.preventDefault();

    return onSubmit({ email, orgName }, destination);
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
          <div>
            {/* TODO: validate as work email */}
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
            value={email}
          />
          <div className="modal-cta-wrap">
            <Button type="submit" variant="brand">
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
