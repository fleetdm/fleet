import PropTypes from "prop-types";
import React, { Component } from "react";

import Button from "components/buttons/Button";
import InputField from "components/forms/fields/InputField";
import Form from "components/forms/Form";
import helpers from "components/forms/RegistrationForm/FleetDetails/helpers";
import formFieldInterface from "interfaces/form_field";

const formFields = ["server_url"];
const { validate } = helpers;

class FleetDetails extends Component {
  static propTypes = {
    className: PropTypes.string,
    currentPage: PropTypes.bool,
    fields: PropTypes.shape({
      server_url: formFieldInterface.isRequired,
    }).isRequired,
    handleSubmit: PropTypes.func.isRequired,
  };

  componentDidUpdate(prevProps) {
    if (
      this.props.currentPage &&
      this.props.currentPage !== prevProps.currentPage
    ) {
      // Component has a transition duration of 300ms set in
      // RegistrationForm/_styles.scss. We need to wait 300ms before
      // calling .focus() to preserve smooth transition.
      setTimeout(() => {
        this.firstInput.input.focus();
      }, 300);
    }
  }

  render() {
    const { className, currentPage, fields, handleSubmit } = this.props;
    const tabIndex = currentPage ? 0 : -1;

    return (
      <form onSubmit={handleSubmit} className={className} autoComplete="off">
        <InputField
          {...fields.server_url}
          label="Fleet web address"
          tabIndex={tabIndex}
          helpText={[
            "Don’t include ",
            <code key="helpText">/latest</code>,
            " or any other path.",
          ]}
          ref={(input) => {
            this.firstInput = input;
          }}
        />{" "}
        <div className="button-wrap--center">
          <Button
            type="submit"
            tabIndex={tabIndex}
            disabled={!currentPage}
            size="wide"
          >
            Next
          </Button>
        </div>
      </form>
    );
  }
}

export default Form(FleetDetails, {
  fields: formFields,
  validate,
});
