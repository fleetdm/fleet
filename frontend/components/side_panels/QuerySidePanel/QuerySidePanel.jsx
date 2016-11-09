import React, { Component, PropTypes } from 'react';

import Button from '../../buttons/Button';
import {
  availability,
  columnsToRender,
  displayTypeForDataType,
  numAdditionalColumns,
  shouldShowAllColumns,
} from './helpers';
import osqueryTableInterface from '../../../interfaces/osquery_table';
import { osqueryTableNames } from '../../../utilities/osquery_tables';
import SecondarySidePanelContainer from '../SecondarySidePanelContainer';
import Dropdown from '../../../components/forms/fields/Dropdown';

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

  onSelectTable = ({ value }) => {
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

    return columns.map((column) => {
      return (
        <div key={column.name} className={`${baseClass}__column-wrapper`}>
          <span className={`${baseClass}__column-name`}>{column.name}</span>
          <div>
            <span>{displayTypeForDataType(column.type)}</span>
            <i className={`${baseClass}__help kolidecon-help`} title={column.description} />
          </div>
        </div>
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

  renderSuggestedQueries = () => {
    const { onSuggestedQueryClick } = this;
    const { selectedOsqueryTable } = this.props;

    return selectedOsqueryTable.examples.map((example) => {
      return (
        <div key={example} className={`${baseClass}__column-wrapper`}>
          <span className={`${baseClass}__suggestion`}>{example}</span>
          <Button
            onClick={onSuggestedQueryClick(example)}
            className={`${baseClass}__load-suggestion`}
            text="LOAD"
          />
        </div>
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
        onSelect={onSelectTable}
        placeholder="Choose Table..."
      />
    );
  }

  render () {
    const {
      renderColumns,
      renderMoreColumns,
      renderTableSelect,
      renderSuggestedQueries,
    } = this;
    const { selectedOsqueryTable: { description, platform } } = this.props;

    return (
      <SecondarySidePanelContainer className={baseClass}>
        <p className={`${baseClass}__header`}>Choose a Table</p>
        {renderTableSelect()}
        <p className={`${baseClass}__table`}>{description}</p>
        <div>
          <p className={`${baseClass}__header`}>OS Availability</p>
          <p className={`${baseClass}__platform`}>{availability(platform)}</p>
        </div>
        <div>
          <p className={`${baseClass}__header`}>Columns</p>
          {renderColumns()}
          {renderMoreColumns()}
        </div>
        <div>
          <p className={`${baseClass}__header`}>Joins</p>
        </div>
        <div>
          <p className={`${baseClass}__header`}>Suggested Queries</p>
          {renderSuggestedQueries()}
        </div>
      </SecondarySidePanelContainer>
    );
  }
}

export default QuerySidePanel;
