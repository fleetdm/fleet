import React, { Component } from 'react';
import PropTypes from 'prop-types';
import { Link } from 'react-router';

import Button from 'components/buttons/Button';
import Icon from 'components/icons/Icon';
import KolideAce from 'components/KolideAce';
import queryInterface from 'interfaces/query';
import SecondarySidePanelContainer from 'components/side_panels/SecondarySidePanelContainer';

const baseClass = 'query-details-side-panel';

class QueryDetailsSidePanel extends Component {
  static propTypes = {
    onEditQuery: PropTypes.func.isRequired,
    query: queryInterface.isRequired,
  };

  handleEditQueryClick = (evt) => {
    evt.preventDefault();

    const { onEditQuery, query } = this.props;

    return onEditQuery(query);
  }

  renderPacks = () => {
    const { query } = this.props;
    const { packs } = query;

    if (!packs || (packs && !packs.length)) {
      return <p>There are no packs associated with this query</p>;
    }

    return (
      <ul className={`${baseClass}__packs`}>
        {packs.map((pack) => {
          return (
            <li className={`${baseClass}__pack-item`} key={`query-side-panel-pack-${pack.id}`}>
              <Icon name="packs" />
              <Link to={`/packs/${pack.id}`} className={`${baseClass}__pack-name`}>{pack.name}</Link>
            </li>
          );
        })}
      </ul>
    );
  }

  render () {
    const { query } = this.props;
    const { handleEditQueryClick, renderPacks } = this;
    const { description, name, query: queryText } = query;

    return (
      <SecondarySidePanelContainer className={baseClass}>
        <h1><Icon name="query" /> {name}</h1>
        <Button onClick={handleEditQueryClick} variant="inverse">Edit/Run Query</Button>
        <KolideAce
          fontSize={12}
          name="query-details"
          readOnly
          showGutter={false}
          value={queryText}
          wrapperClassName={`${baseClass}__query-preview`}
          wrapEnabled
        />
        <h2>Description</h2>
        <p className={`${baseClass}__description`}>{description || <em>No description available</em>}</p>
        <h2>Packs</h2>
        {renderPacks()}
      </SecondarySidePanelContainer>
    );
  }
}

export default QueryDetailsSidePanel;
