import React, { Component, PropTypes } from 'react';

import osqueryTableInterface from 'interfaces/osquery_table';
import { osqueryTableNames } from 'utilities/osquery_tables';
import Dropdown from 'components/forms/fields/Dropdown';
import Icon from 'components/icons/Icon';
import PlatformIcon from 'components/icons/PlatformIcon';
import SecondarySidePanelContainer from '../SecondarySidePanelContainer';

import {
  availability,
  columnsToRender,
  displayTypeForDataType,
  numAdditionalColumns,
  shouldShowAllColumns,
} from './helpers';

const baseClass = 'query-side-panel';

class QuerySidePanel extends Component {
  static propTypes = {
    onOsqueryTableSelect: PropTypes.func,
    onTextEditorInputChange: PropTypes.func,
    selectedOsqueryTable: osqueryTableInterface,
  };

  componentWillMount () {
    const { selectedOsqueryTable } = this.props;
    const showAllColumns = shouldShowAllColumns(selectedOsqueryTable);

    this.setState({ showAllColumns });
  }

  componentWillReceiveProps (nextProps) {
    const { selectedOsqueryTable } = nextProps;

    if (this.props.selectedOsqueryTable !== selectedOsqueryTable) {
      const showAllColumns = shouldShowAllColumns(selectedOsqueryTable);

      this.setState({ showAllColumns });
    }

    return false;
  }

  onSelectTable = (value) => {
    const { onOsqueryTableSelect } = this.props;

    onOsqueryTableSelect(value);

    return false;
  }

  onShowAllColumns = () => {
    this.setState({ showAllColumns: true });
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
    const { showAllColumns } = this.state;
    const columns = columnsToRender(selectedOsqueryTable, showAllColumns);
    const columnBaseClass = 'query-column-list';

    return columns.map((column) => {
      return (
        <li key={column.name} className={`${columnBaseClass}__item`}>
          <span className={`${columnBaseClass}__name`}>{column.name}</span>
          <div className={`${columnBaseClass}__description`}>
            <span className={`${columnBaseClass}__type`}>{displayTypeForDataType(column.type)}</span>
            <Icon name="help-solid" className={`${columnBaseClass}__help`} title={column.description} />
          </div>
        </li>
      );
    });
  }

  renderMoreColumns = () => {
    const { selectedOsqueryTable } = this.props;
    const { showAllColumns } = this.state;
    const { onShowAllColumns } = this;

    if (showAllColumns) {
      return false;
    }

    return (
      <div className={`${baseClass}__column-wrapper`}>
        <span className={`${baseClass}__more-columns`}>{numAdditionalColumns(selectedOsqueryTable)} MORE COLUMNS</span>
        <button className={`button--unstyled ${baseClass}__show-columns`} onClick={onShowAllColumns}>SHOW</button>
      </div>
    );
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
      renderMoreColumns,
      renderTableSelect,
    } = this;
    const { selectedOsqueryTable: { description, platform } } = this.props;
    const platformArr = availability(platform);

    return (
      <SecondarySidePanelContainer className={baseClass}>
        <div className={`${baseClass}__choose-table`}>
          <h2 className={`${baseClass}__header`}>Choose a Table</h2>
          {renderTableSelect()}
          <p className={`${baseClass}__description`}>{description}</p>
        </div>

        <div className={`${baseClass}__os-availability`}>
          <h2 className={`${baseClass}__header`}>OS Availability</h2>
          <ul className={`${baseClass}__platforms`}>
            {platformArr.map((os, idx) => {
              if (os.type === 'all') {
                return <li key={idx}><Icon name="hosts" /> {os.display_text}</li>;
              }

              return <li key={idx}><PlatformIcon name={os.display_text} /> {os.display_text}</li>;
            })}
          </ul>
        </div>

        <div className={`${baseClass}__columns`}>
          <h2 className={`${baseClass}__header`}>Columns</h2>
          <ul className={`${baseClass}__column-list`}>
            {renderColumns()}
          </ul>
          {renderMoreColumns()}
        </div>
      </SecondarySidePanelContainer>
    );
  }
}

export default QuerySidePanel;
