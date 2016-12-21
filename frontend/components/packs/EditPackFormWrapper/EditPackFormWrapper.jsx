import React, { Component, PropTypes } from 'react';
import { noop } from 'lodash';

import Button from 'components/buttons/Button';
import EditPackForm from 'components/forms/packs/EditPackForm';
import Icon from 'components/Icon';
import packInterface from 'interfaces/pack';
import SelectTargetsDropdown from 'components/forms/fields/SelectTargetsDropdown';

class EditPackFormWrapper extends Component {
  static propTypes = {
    className: PropTypes.string,
    handleSubmit: PropTypes.func,
    isEdit: PropTypes.bool.isRequired,
    onCancelEditPack: PropTypes.func.isRequired,
    onEditPack: PropTypes.func.isRequired,
    onFetchTargets: PropTypes.func,
    pack: packInterface.isRequired,
    targetsCount: PropTypes.number,
  };

  render () {
    const {
      className,
      handleSubmit,
      isEdit,
      onCancelEditPack,
      onEditPack,
      onFetchTargets,
      pack,
      targetsCount,
    } = this.props;

    if (isEdit) {
      return (
        <EditPackForm
          className={className}
          formData={pack}
          handleSubmit={handleSubmit}
          onCancel={onCancelEditPack}
        />
      );
    }

    return (
      <div className={className}>
        <Button
          onClick={onEditPack}
          text="EDIT"
          type="button"
          variant="brand"
        />
        <h1><Icon name="packs" /> {pack.name}</h1>
        <p>{pack.description}</p>
        <SelectTargetsDropdown
          label="select pack targets"
          name="selected-pack-targets"
          onFetchTargets={onFetchTargets}
          onSelect={noop}
          targetsCount={targetsCount}
          disabled
        />
      </div>
    );
  }
}

export default EditPackFormWrapper;
