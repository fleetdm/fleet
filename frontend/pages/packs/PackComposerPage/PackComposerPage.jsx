import React, { Component } from "react";
import PropTypes from "prop-types";
import { connect } from "react-redux";
import { noop } from "lodash";
import { push } from "react-router-redux";

import packActions from "redux/nodes/entities/packs/actions";
import PackForm from "components/forms/packs/PackForm";
import PackInfoSidePanel from "components/side_panels/PackInfoSidePanel";
import PATHS from "router/paths";

const baseClass = "pack-composer";

export class PackComposerPage extends Component {
  static propTypes = {
    dispatch: PropTypes.func,
    serverErrors: PropTypes.shape({
      base: PropTypes.string,
    }),
  };

  static defaultProps = {
    dispatch: noop,
  };

  constructor(props) {
    super(props);

    this.state = { selectedTargetsCount: 0 };
  }

  onFetchTargets = (query, targetsResponse) => {
    const { targets_count: selectedTargetsCount } = targetsResponse;

    this.setState({ selectedTargetsCount });

    return false;
  };

  visitPackPage = (packID) => {
    const { dispatch } = this.props;

    dispatch(push(PATHS.PACK({ id: packID })));

    return false;
  };

  handleSubmit = (formData) => {
    const { create } = packActions;
    const { dispatch } = this.props;
    const { visitPackPage } = this;

    return dispatch(create(formData)).then((pack) => {
      const { id: packID } = pack;

      return visitPackPage(packID);
    });
  };

  render() {
    const { handleSubmit, onFetchTargets } = this;
    const { selectedTargetsCount } = this.state;
    const { serverErrors } = this.props;

    return (
      <div className="has-sidebar">
        <PackForm
          className={`${baseClass}__pack-form body-wrap`}
          handleSubmit={handleSubmit}
          onFetchTargets={onFetchTargets}
          selectedTargetsCount={selectedTargetsCount}
          serverErrors={serverErrors}
        />
        <PackInfoSidePanel />
      </div>
    );
  }
}

const mapStateToProps = (state) => {
  const { errors: serverErrors } = state.entities.packs;

  return { serverErrors };
};

export default connect(mapStateToProps)(PackComposerPage);
