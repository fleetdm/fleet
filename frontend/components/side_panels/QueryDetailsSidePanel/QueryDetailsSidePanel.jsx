import React, { Component } from "react";
import PropTypes from "prop-types";
import { Link } from "react-router";

import permissionUtils from "utilities/permissions";
import Button from "components/buttons/Button";
import FleetAce from "components/FleetAce";
import queryInterface from "interfaces/query";
import userInterface from "interfaces/user";
import SecondarySidePanelContainer from "components/side_panels/SecondarySidePanelContainer";

const baseClass = "query-details-side-panel";

class QueryDetailsSidePanel extends Component {
  static propTypes = {
    onEditQuery: PropTypes.func.isRequired,
    query: queryInterface.isRequired,
    currentUser: userInterface,
  };

  handleEditQueryClick = (evt) => {
    evt.preventDefault();

    const { onEditQuery, query } = this.props;

    return onEditQuery(query);
  };

  renderPacks = () => {
    const { query } = this.props;
    const { packs } = query;

    if (!packs || (packs && !packs.length)) {
      return (
        <p className={`${baseClass}__description`}>
          There are no packs associated with this query.
        </p>
      );
    }

    return (
      <ul className={`${baseClass}__packs`}>
        {packs.map((pack) => {
          return (
            <li
              className={`${baseClass}__pack-item`}
              key={`query-side-panel-pack-${pack.id}`}
            >
              <Link
                to={`/packs/${pack.id}`}
                className={`${baseClass}__pack-name`}
              >
                {pack.name}
              </Link>
            </li>
          );
        })}
      </ul>
    );
  };

  render() {
    const { query, currentUser } = this.props;
    const { handleEditQueryClick, renderPacks } = this;
    const { description, name, query: queryText, observer_can_run } = query;

    const renderCTA = () => {
      if (
        permissionUtils.isGlobalAdmin(currentUser) ||
        permissionUtils.isGlobalMaintainer(currentUser)
      ) {
        return "Edit or run query";
      }
      if (
        permissionUtils.isAnyTeamMaintainer(currentUser) ||
        (permissionUtils.isOnlyObserver(currentUser) && observer_can_run)
      ) {
        return "Run query";
      }
      return "Show query";
    };

    return (
      <SecondarySidePanelContainer className={baseClass}>
        <p className={`${baseClass}__label`}>Query</p>
        <p className={`${baseClass}__description`}>{name}</p>
        {!permissionUtils.isOnlyObserver(currentUser) && (
          <>
            <p className={`${baseClass}__label`}>SQL</p>
            <FleetAce
              fontSize={12}
              name="query-details"
              readOnly
              showGutter={false}
              value={queryText}
              wrapperClassName={`${baseClass}__query-preview`}
              wrapEnabled
            />
          </>
        )}
        <p className={`${baseClass}__label`}>Description</p>
        <p className={`${baseClass}__description`}>
          {description || <>No description available.</>}
        </p>
        {!permissionUtils.isOnlyObserver(currentUser) && (
          <>
            <p className={`${baseClass}__label`}>Packs</p>
            {renderPacks()}
          </>
        )}
        <Button onClick={handleEditQueryClick} variant="brand">
          {renderCTA(currentUser)}
        </Button>
      </SecondarySidePanelContainer>
    );
  }
}

export default QueryDetailsSidePanel;
