import React, { Component } from 'react';
import PropTypes from 'prop-types';
import { connect } from 'react-redux';
import { noop } from 'lodash';

import { getOsqueryOptions, updateOsqueryOptions } from 'redux/nodes/osquery/actions';
import validateYaml from 'components/forms/validators/validate_yaml';
import OsqueryOptionsForm from 'components/forms/admin/OsqueryOptionsForm';
import Icon from 'components/icons/Icon';
import { renderFlash } from 'redux/nodes/notifications/actions';

const yaml = require('js-yaml');

const baseClass = 'osquery-options';

class OsqueryOptionsPage extends Component {
  static propTypes = {
    options: PropTypes.object, // eslint-disable-line react/forbid-prop-types
    dispatch: PropTypes.func,
  };

  static defaultProps = {
    dispatch: noop,
  }

  componentDidMount() {
    const { dispatch } = this.props;
    dispatch(getOsqueryOptions())
      .catch(() => false);
  }

  onSaveOsqueryOptionsFormSubmit = (formData) => {
    const { dispatch } = this.props;
    const { error } = validateYaml(formData.osquery_options);

    if (error) {
      dispatch(renderFlash('error', error));

      return false;
    }

    dispatch(updateOsqueryOptions(formData))
      .then(() => {
        dispatch(renderFlash('success', 'Osquery options updated!'));

        return false;
      })
      .catch((errors) => {
        if (errors.base) {
          dispatch(renderFlash('error', errors.base));
        }

        return false;
      });

    return false;
  }

  render () {
    const { options } = this.props;
    const formData = {
      osquery_options: yaml.safeDump(options),
    };
    const { onSaveOsqueryOptionsFormSubmit } = this;

    return (
      <div className={`${baseClass} body-wrap`}>
        <h1>Osquery Options</h1>
        <div className={`${baseClass}__form-wrapper`}>
          <OsqueryOptionsForm
            formData={formData}
            handleSubmit={onSaveOsqueryOptionsFormSubmit}
          />
          <div className={`${baseClass}__form-details`}>
            <p>This file describes options returned to osqueryd when it checks for configuration.</p>
            <p>See Fleet documentation for an example file that includes the overrides option.</p>
            <a
              href="https://github.com/kolide/fleet/blob/master/docs/cli/file-format.md#osquery-configuration-options"
              target="_blank"
              rel="noreferrer"
              className="button button--muted"
            >
              GO TO FLEET DOCS
              <Icon name="right-arrow" />
            </a>
            <p>See Osquery documentation for all available options.</p>
            <a
              href="https://osquery.readthedocs.io/en/stable/deployment/configuration/#options"
              target="_blank"
              rel="noreferrer"
              className="button button--muted"
            >
              GO TO OSQUERY DOCS
              <Icon name="right-arrow" />
            </a>
          </div>
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
