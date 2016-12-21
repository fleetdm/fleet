import React, { PropTypes } from 'react';

import ConfigurePackQueryForm from 'components/forms/ConfigurePackQueryForm';
import queryInterface from 'interfaces/query';
import SearchPackQuery from './SearchPackQuery';
import SecondarySidePanelContainer from '../SecondarySidePanelContainer';

const baseClass = 'schedule-query-side-panel';

const ScheduleQuerySidePanel = ({ allQueries, onConfigurePackQuerySubmit, onSelectQuery, selectedQuery }) => {
  const renderForm = () => {
    if (!selectedQuery) {
      return false;
    }

    const formData = { query_id: selectedQuery.id };

    return (
      <ConfigurePackQueryForm
        formData={formData}
        handleSubmit={onConfigurePackQuerySubmit}
      />
    );
  };

  return (
    <SecondarySidePanelContainer className={baseClass}>
      <SearchPackQuery
        allQueries={allQueries}
        onSelectQuery={onSelectQuery}
        selectedQuery={selectedQuery}
      />
      {renderForm()}
    </SecondarySidePanelContainer>
  );
};

ScheduleQuerySidePanel.propTypes = {
  allQueries: PropTypes.arrayOf(queryInterface),
  onConfigurePackQuerySubmit: PropTypes.func,
  onSelectQuery: PropTypes.func,
  selectedQuery: queryInterface,
};

export default ScheduleQuerySidePanel;
