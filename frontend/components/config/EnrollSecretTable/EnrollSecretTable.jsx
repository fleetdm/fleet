import React, { Component } from 'react';
import PropTypes from 'prop-types';
import FileSaver from 'file-saver';

import Button from 'components/buttons/Button';
import enrollSecretInterface from 'interfaces/enroll_secret';
import InputField from 'components/forms/fields/InputField';
import Icon from 'components/icons/Icon';
import { stringToClipboard } from 'utilities/copy_text';

const baseClass = 'enroll-secrets';

class EnrollSecretRow extends Component {
  static propTypes = {
    name: PropTypes.string.isRequired,
    secret: PropTypes.string.isRequired,
  }

  constructor (props) {
    super(props);
    this.state = { showSecret: false, copyMessage: '' };
  }

  onCopySecret = (evt) => {
    evt.preventDefault();

    const { secret } = this.props;

    stringToClipboard(secret)
      .then(() => this.setState({ copyMessage: '(copied)' }))
      .catch(() => this.setState({ copyMessage: '(copy failed)' }));

    // Clear message after 1 second
    setTimeout(() => this.setState({ copyMessage: '' }), 1000);

    return false;
  }

  onDownloadSecret = (evt) => {
    evt.preventDefault();

    const { secret } = this.props;

    const filename = 'secret.txt';
    const file = new global.window.File([secret], filename);
    
    FileSaver.saveAs(file);

    return false;
  }

  
  onToggleSecret = (evt) => {
    evt.preventDefault();

    const { showSecret } = this.state;

    this.setState({ showSecret: !showSecret });
    return false;
  };

  renderLabel = () => {
    const { name } = this.props;
    const { showSecret, copyMessage } = this.state;
    const { onCopySecret, onDownloadSecret, onToggleSecret } = this;

    return (
      <span>
        {name}
        <span className="buttons">
          {copyMessage && <span>{`${copyMessage} `}</span>}
          <Button
            variant="unstyled"
            className={`${baseClass}__secret-copy-icon`}
            onClick={onCopySecret}
          >
            <Icon name="clipboard" />
          </Button>
          |
          <a
            href="#"
            variant="unstyled"
            className={`${baseClass}__secret-download-icon`}
            onClick={onDownloadSecret}
          >
            Download
          </a>
          |
          <a
            href="#showSecret"
            onClick={onToggleSecret}
            className={`${baseClass}__show-secret`}
          >
            {showSecret ? 'Hide' : 'Show'}
          </a>
        </span>
      </span>
    );
  }

  render () {
    const { secret } = this.props;
    const { showSecret } = this.state;
    const { renderLabel } = this;

    return (
      <div>
        <InputField
          disabled
          inputWrapperClass={`${baseClass}__secret-input`}
          name="osqueryd-secret"
          label={renderLabel()}
          type={showSecret ? 'text' : 'password'}
          value={secret}
        />
      </div>
    );
  }
}

class EnrollSecretTable extends Component {
  static propTypes = {
    secrets: enrollSecretInterface.isRequired,
  }

  render() {
    const { secrets } = this.props;
    const activeSecrets = secrets.filter(s => s.active);

    if (activeSecrets.length === 0) {
      return (<div className={baseClass}><em>No active enroll secrets.</em></div>);
    }

    return (
      <div className={baseClass}>
        {activeSecrets.map(({ name, secret }) =>
          <EnrollSecretRow key={name} name={name} secret={secret} />,
        )}
      </div>
    );
  }
}

export default EnrollSecretTable;
export { EnrollSecretRow };
