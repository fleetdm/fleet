import React, { Component, PropTypes } from 'react';

import Modal from '../Modal';
import SaveQueryForm from '../../forms/queries/SaveQueryForm';

class SaveQueryFormComponent extends Component {
  static propTypes = {
    onCancel: PropTypes.func,
    onSubmit: PropTypes.func,
  };

  render () {
    const { onCancel, onSubmit } = this.props;

    return (
      <Modal onExit={onCancel} title="Save Query">
        <SaveQueryForm onCancel={onCancel} onSubmit={onSubmit} />
      </Modal>
    );
  }
}

export default SaveQueryFormComponent;
