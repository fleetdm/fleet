import React, { Component } from "react";
import PropTypes from "prop-types";

import Button from "components/buttons/Button";
import enrollSecretInterface from "interfaces/enroll_secret";
import InputField from "components/forms/fields/InputField";
import FleetIcon from "components/icons/FleetIcon";
import { stringToClipboard } from "utilities/copy_text";
<<<<<<< HEAD:frontend/components/EnrollSecretTable/EnrollSecretTable.jsx
import EyeIcon from "../../../assets/images/icon-eye-16x16@2x.png";
import DownloadIcon from "../../../assets/images/icon-download-12x12@2x.png";
=======
import EyeIcon from "../../../../assets/images/icon-eye-16x16@2x.png";
import EditIcon from "../../../../assets/images/icon-pencil-14x14@2x.png";
import DeleteIcon from "../../../../assets/images/icon-trash-14x14@2x.png";
>>>>>>> 46530194 (Styling):frontend/components/config/EnrollSecretTable/EnrollSecretTable.jsx

const baseClass = "enroll-secrets";

class EnrollSecretRow extends Component {
  static propTypes = {
    secret: PropTypes.string.isRequired,
    toggleSecretEditorModal: PropTypes.func,
    toggleDeleteSecretModal: PropTypes.func,
  };

  constructor(props) {
    super(props);
    this.state = { showSecret: false, copyMessage: "" };
  }

  onCopySecret = (evt) => {
    evt.preventDefault();

    const { secret } = this.props;

    stringToClipboard(secret)
      .then(() => this.setState({ copyMessage: "Copied!" }))
      .catch(() => this.setState({ copyMessage: "Copy failed" }));

    // Clear message after 1 second
    setTimeout(() => this.setState({ copyMessage: "" }), 1000);

    return false;
  };

  onToggleSecret = (evt) => {
    evt.preventDefault();

    const { showSecret } = this.state;

    this.setState({ showSecret: !showSecret });
    return false;
  };

  renderLabel = () => {
    const { copyMessage } = this.state;
    const { onCopySecret, onToggleSecret } = this;

    return (
      <span className={`${baseClass}__name`}>
        <span className="buttons">
          {copyMessage && <span>{`${copyMessage} `}</span>}
          <Button
            variant="unstyled"
            className={`${baseClass}__secret-copy-icon`}
            onClick={onCopySecret}
          >
            <FleetIcon name="clipboard" />
          </Button>
          <a
            href="#showSecret"
            onClick={onToggleSecret}
            className={`${baseClass}__show-secret`}
          >
            <img src={EyeIcon} alt="show/hide" />
          </a>
        </span>
      </span>
    );
  };

  render() {
    const {
      secret,
      toggleSecretEditorModal,
      toggleDeleteSecretModal,
    } = this.props;
    const { showSecret } = this.state;
    const { renderLabel } = this;

    return (
      <div className={`${baseClass}__secret`}>
        <InputField
          disabled
          inputWrapperClass={`${baseClass}__secret-input`}
          name="osqueryd-secret"
          label={renderLabel()}
          type={showSecret ? "text" : "password"}
          value={secret}
        />
        <Button
          onClick={toggleSecretEditorModal}
          className={`${baseClass}__edit-secret-btn`}
          variant="text-icon"
        >
          <>
            <img src={EditIcon} alt="Edit secret icon" />
          </>
        </Button>
        <Button
          onClick={toggleDeleteSecretModal}
          className={`${baseClass}__delete-secret-btn`}
          variant="text-icon"
        >
          <>
            <img src={DeleteIcon} alt="Delete secret icon" />
          </>
        </Button>
      </div>
    );
  }
}

class EnrollSecretTable extends Component {
  static propTypes = {
    secrets: PropTypes.arrayOf(enrollSecretInterface).isRequired,
  };

  render() {
    const { secrets } = this.props;

    let enrollSecretsClass = baseClass;
    if (secrets.length === 0) {
      return (
        <div className={baseClass}>
          <em>No active enroll secrets.</em>
        </div>
      );
    } else if (secrets.length > 1)
      enrollSecretsClass += ` ${baseClass}--multiple-secrets`;

    return (
      <div className={enrollSecretsClass}>
        {secrets.map(({ secret }) => (
          <EnrollSecretRow key={secret} secret={secret} />
        ))}
      </div>
    );
  }
}

export default EnrollSecretTable;
export { EnrollSecretRow };
