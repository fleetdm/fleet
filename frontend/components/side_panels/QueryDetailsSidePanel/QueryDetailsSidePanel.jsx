import React, { Component } from "react";
import PropTypes from "prop-types";
import { Link } from "react-router";

import Button from "components/buttons/Button";
import KolideAce from "components/KolideAce";
import queryInterface from "interfaces/query";
import SecondarySidePanelContainer from "components/side_panels/SecondarySidePanelContainer";

const baseClass = "query-details-side-panel";

class QueryDetailsSidePanel extends Component {
  static propTypes = {
    onEditQuery: PropTypes.func.isRequired,
    query: queryInterface.isRequired,
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
          There are no packs associated with this query
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
    const { query } = this.props;
    const { handleEditQueryClick, renderPacks } = this;
    const { description, name, query: queryText } = query;

    return (
      <SecondarySidePanelContainer className={baseClass}>
        <p className={`${baseClass}__label`}>Query</p>
        <h1>{name}</h1>
        <p className={`${baseClass}__label`}>SQL</p>
        <KolideAce
          fontSize={12}
          name="query-details"
          readOnly
          showGutter={false}
          value={queryText}
          wrapperClassName={`${baseClass}__query-preview`}
          wrapEnabled
        />
        <p className={`${baseClass}__label`}>Description</p>
        <p className={`${baseClass}__description`}>
          {description || <em>No description available</em>}
        </p>
        <p className={`${baseClass}__label`}>Packs</p>
        {renderPacks()}
        <Button onClick={handleEditQueryClick} variant="inverse">
          Edit or run query
        </Button>
      </SecondarySidePanelContainer>
    );
  }
}

export default QueryDetailsSidePanel;
