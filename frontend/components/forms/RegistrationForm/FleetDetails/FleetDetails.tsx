import React, { Component } from "react";
import PropTypes from "prop-types";

import Form from "components/forms/Form";
import formFieldInterface from "interfaces/form_field";
import Button from "components/buttons/Button";
import helpers from "components/forms/RegistrationForm/FleetDetails/helpers";
import InputField from "components/forms/fields/InputField";

const formFields = ["server_url"];
const { validate } = helpers;

interface IFleetDetailsProps {
  className: string;
  currentPage: boolean;
  fields: {
    server_url: string;
  };
  handleSubmit: () => void;
}

const FleetDetails = ({
  className,
  currentPage,
  fields,
  handleSubmit,
}: IFleetDetailsProps) => {
  const tabIndex = currentPage ? 0 : -1;

  // TODO
  // Component has a transition duration of 300ms set in
  // RegistrationForm/_styles.scss. We need to wait 300ms before
  // calling .focus() to preserve smooth transition.
  setTimeout(() => {
    firstInput.input.focus();
  }, 300);

  return (
    <form onSubmit={handleSubmit} className={className} autoComplete="off">
      <InputField
        value={fields.server_url}
        label="Fleet web address"
        tabIndex={tabIndex}
        helpText={[
          "Don’t include ",
          <code key="helpText">/latest</code>,
          " or any other path.",
        ]}
      />
      <Button
        type="submit"
        tabIndex={tabIndex}
        disabled={!currentPage}
        variant="brand"
      >
        Next
      </Button>
    </form>
  );
};

// TODO
// export default Form(FleetDetails, {
//   fields: formFields,
//   validate,
// });

export default FleetDetails;
