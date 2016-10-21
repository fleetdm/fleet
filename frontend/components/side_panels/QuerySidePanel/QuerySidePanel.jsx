import React, { Component, PropTypes } from 'react';
import radium from 'radium';

import Button from '../../buttons/Button';
import {
  availability,
  columnsToRender,
  displayTypeForDataType,
  numAdditionalColumns,
  shouldShowAllColumns,
} from './helpers';
import osqueryTableInterface from '../../../interfaces/osquery_table';
import { osqueryTables } from '../../../utilities/osquery_tables';
import SecondarySidePanelContainer from '../SecondarySidePanelContainer';

const classBlock = 'query-side-panel';

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

  onSelectTable = ({ target }) => {
    const { onOsqueryTableSelect } = this.props;
    const { value: tableName } = target;

    onOsqueryTableSelect(tableName);

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
        <div key={column.name} className={`${classBlock}__column-wrapper`}>
          <span className={`${classBlock}__column-name`}>{column.name}</span>
          <div>
            <span>{displayTypeForDataType(column.type)}</span>
            <i className={`${classBlock}__help kolidecon-help`} title={column.description} />
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
      <div className={`${classBlock}__column-wrapper`}>
        <span className={`${classBlock}__more-columns`}>{numAdditionalColumns(selectedOsqueryTable)} MORE COLUMNS</span>
        <button className={`btn--unstyled ${classBlock}__show-columns`} onClick={onShowAllColumns}>SHOW</button>
      </div>
    );
  }

  renderSuggestedQueries = () => {
    const { onSuggestedQueryClick } = this;
    const { selectedOsqueryTable } = this.props;

    return selectedOsqueryTable.examples.map((example) => {
      return (
        <div key={example} className={`${classBlock}__column-wrapper`}>
          <span className={`${classBlock}__suggestion`}>{example}</span>
          <Button
            onClick={onSuggestedQueryClick(example)}
            className={`${classBlock}__load-suggestion`}
            text="LOAD"
          />
        </div>
      );
    });
  }

  renderTableSelect = () => {
    const { onSelectTable } = this;
    const { selectedOsqueryTable } = this.props;

    return (
      <div className="kolide-dropdown-wrapper">
        <select className="kolide-dropdown" onChange={onSelectTable} value={selectedOsqueryTable.name}>
          {osqueryTables.map((table) => {
            return <option key={table.name} value={table.name}>{table.name}</option>;
          })}
        </select>
      </div>
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
      <SecondarySidePanelContainer className={classBlock}>
        <p className={`${classBlock}__header`}>Choose a Table</p>
        {renderTableSelect()}
        <p className={`${classBlock}__table`}>{description}</p>
        <div>
          <p className={`${classBlock}__header`}>OS Availability</p>
          <p className={`${classBlock}__platform`}>{availability(platform)}</p>
        </div>
        <div>
          <p className={`${classBlock}__header`}>Columns</p>
          {renderColumns()}
          {renderMoreColumns()}
        </div>
        <div>
          <p className={`${classBlock}__header`}>Joins</p>
        </div>
        <div>
          <p className={`${classBlock}__header`}>Suggested Queries</p>
          {renderSuggestedQueries()}
        </div>
      </SecondarySidePanelContainer>
    );
  }
}

export default radium(QuerySidePanel);
