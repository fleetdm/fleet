import React, { Component } from "react";
import PropTypes from "prop-types";
import { noop } from "lodash";

import Button from "components/buttons/Button";
import EditPackForm from "components/forms/packs/EditPackForm";
import packInterface from "interfaces/pack";
import SelectTargetsDropdown from "components/forms/fields/SelectTargetsDropdown";
import targetInterface from "interfaces/target";

const baseClass = "edit-pack-form";

class EditPackFormWrapper extends Component {
  static propTypes = {
    className: PropTypes.string,
    handleSubmit: PropTypes.func,
    onCancelEditPack: PropTypes.func.isRequired,
    onFetchTargets: PropTypes.func,
    pack: packInterface.isRequired,
    packTargets: PropTypes.arrayOf(targetInterface),
    targetsCount: PropTypes.number,
    isPremiumTier: PropTypes.bool,
  };

  render() {
    const {
      className,
      handleSubmit,
      onCancelEditPack,
      onFetchTargets,
      pack,
      packTargets,
      targetsCount,
      isPremiumTier,
    } = this.props;

    return (
      <EditPackForm
        className={className}
        formData={{ ...pack, targets: packTargets }}
        handleSubmit={handleSubmit}
        onCancel={onCancelEditPack}
        onFetchTargets={onFetchTargets}
        targetsCount={targetsCount}
        isPremiumTier={isPremiumTier}
      />
    );
  }
}

export default EditPackFormWrapper;
