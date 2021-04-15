import React, { Component } from "react";
import PropTypes from "prop-types";
import { connect } from "react-redux";
import { noop } from "lodash";
import yaml from "js-yaml";

import osqueryOptionsActions from "redux/nodes/osquery/actions";
import validateYaml from "components/forms/validators/validate_yaml";
import OsqueryOptionsForm from "components/forms/admin/OsqueryOptionsForm";
import { renderFlash } from "redux/nodes/notifications/actions";
import OpenNewTabIcon from "../../../../assets/images/open-new-tab-12x12@2x.png";

const baseClass = "osquery-options";

export class OsqueryOptionsPage extends Component {
  static propTypes = {
    options: PropTypes.object, // eslint-disable-line react/forbid-prop-types
    dispatch: PropTypes.func,
  };

  static defaultProps = {
    dispatch: noop,
  };

  componentDidMount() {
    const { dispatch } = this.props;
    dispatch(osqueryOptionsActions.getOsqueryOptions()).catch(() => false);
  }

  onSaveOsqueryOptionsFormSubmit = (formData) => {
    const { dispatch } = this.props;
    const { error } = validateYaml(formData.osquery_options);

    if (error) {
      dispatch(renderFlash("error", error));

      return false;
    }

    dispatch(osqueryOptionsActions.updateOsqueryOptions(formData))
      .then(() => {
        dispatch(renderFlash("success", "Osquery options updated!"));

        return false;
      })
      .catch((errors) => {
        if (errors.base) {
          dispatch(renderFlash("error", errors.base));
        }

        return false;
      });

    return false;
  };

  render() {
    const { options } = this.props;
    const formData = {
      osquery_options: yaml.dump(options),
    };
    const { onSaveOsqueryOptionsFormSubmit } = this;

    return (
      <div className={`${baseClass} body-wrap`}>
        <p className={`${baseClass}__page-description`}>
          This file describes options returned to osquery when it checks for
          configuration.
        </p>
        <div className={`${baseClass}__info-banner`}>
          <p>
            See Fleet documentation for an example file that includes the
            overrides option.
          </p>
          <a
            href="https://github.com/fleetdm/fleet/blob/master/docs/1-Using-Fleet/2-fleetctl-CLI.md#osquery-configuration-options"
            target="_blank"
            rel="noreferrer"
          >
            Go to Fleet docs
            <img src={OpenNewTabIcon} alt="open new tab" />
          </a>
        </div>
        <div className={`${baseClass}__form-wrapper`}>
          <OsqueryOptionsForm
            formData={formData}
            handleSubmit={onSaveOsqueryOptionsFormSubmit}
          />
        </div>
      </div>
    );
  }
}

const mapStateToProps = (state) => {
  const { osquery } = state;
  const { options } = osquery;
  return {
    options,
  };
};

export default connect(mapStateToProps)(OsqueryOptionsPage);
