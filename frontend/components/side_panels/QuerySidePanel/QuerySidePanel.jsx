import React, { Component } from 'react';
import PropTypes from 'prop-types';

import osqueryTableInterface from 'interfaces/osquery_table';
import { osqueryTableNames } from 'utilities/osquery_tables';
import { iconNameForPlatform } from 'utilities/icon_name';
import Dropdown from 'components/forms/fields/Dropdown';
import KolideIcon from 'components/icons/KolideIcon';
import Icon from 'components/icons/Icon';
import SecondarySidePanelContainer from '../SecondarySidePanelContainer';

import displayTypeForDataType from './helpers';

const baseClass = 'query-side-panel';

class QuerySidePanel extends Component {
  static propTypes = {
    onOsqueryTableSelect: PropTypes.func,
    onTextEditorInputChange: PropTypes.func,
    selectedOsqueryTable: osqueryTableInterface,
  };

  onSelectTable = (value) => {
    const { onOsqueryTableSelect } = this.props;

    onOsqueryTableSelect(value);

    return false;
  }

  onSuggestedQueryClick = (query) => {
    return (evt) => {
      evt.preventDefault();

      const { onTextEditorInputChange } = this.props;

      return onTextEditorInputChange(query);
    };
  };

  renderColumns = () => {
    const { selectedOsqueryTable } = this.props;
    const columns = selectedOsqueryTable.columns;
    const columnBaseClass = 'query-column-list';

    return columns.map((column) => {
      return (
        <li key={column.name} className={`${columnBaseClass}__item`}>
          <span className={`${columnBaseClass}__name`}>{column.name}</span>
          <div className={`${columnBaseClass}__description`}>
            <span className={`${columnBaseClass}__type`}>{displayTypeForDataType(column.type)}</span>
            <KolideIcon name="help-solid" className={`${columnBaseClass}__help`} title={column.description} />
          </div>
        </li>
      );
    });
  }

  renderTableSelect = () => {
    const { onSelectTable } = this;
    const { selectedOsqueryTable } = this.props;

    const tableNames = osqueryTableNames.map((name) => {
      return { label: name, value: name };
    });

    return (
      <Dropdown
        options={tableNames}
        value={selectedOsqueryTable.name}
        onChange={onSelectTable}
        placeholder="Choose Table..."
      />
    );
  }

  render () {
    const {
      renderColumns,
      renderTableSelect,
    } = this;
    const { selectedOsqueryTable: { description, platforms } } = this.props;

    return (
      <SecondarySidePanelContainer className={baseClass}>
        <div className={`${baseClass}__choose-table`}>
          <h2 className={`${baseClass}__header`}>Documentation</h2>
          {renderTableSelect()}
          <p className={`${baseClass}__description`}>{description}</p>
        </div>

        <div className={`${baseClass}__os-availability`}>
          <h2 className={`${baseClass}__header`}>OS Availability</h2>
          <ul className={`${baseClass}__platforms`}>
            {platforms.map((platform) => {
              if (platform === 'all') {
                return <li key={platform}><KolideIcon name="hosts" /> {platform}</li>;
              } else if (platform === 'freebsd') {
                return <li key={platform}><KolideIcon name="single-host" /> {platform}</li>;
              }

              return <li key={platform}><Icon name={iconNameForPlatform(platform)} size="20" /> {platform}</li>;
            })}
          </ul>
        </div>

        <div className={`${baseClass}__columns`}>
          <h2 className={`${baseClass}__header`}>Columns</h2>
          <ul className={`${baseClass}__column-list`}>
            {renderColumns()}
          </ul>
        </div>
      </SecondarySidePanelContainer>
    );
  }
}

export default QuerySidePanel;
