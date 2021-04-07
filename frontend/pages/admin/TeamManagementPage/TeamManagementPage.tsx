import React from 'react';
import { useSelector, useDispatch } from 'react-redux';

import { ITeam } from 'interfaces/team';

import TableContainer from 'components/TableContainer';

import EmptyTeams from './components/EmptyTeams';
import { generateTableHeaders, generateDataSet } from './TeamTableConfig';

const baseClass = 'team-management';

// TODO: should probably live close to the store.js file and imported in.
interface RootState {
  entities: {
    teams: {
      isLoading: boolean;
      data: { [id: number]: ITeam };
    }
  }
}

const TeamManagementPage = () => {
  const loadingTableData = useSelector((state: RootState) => state.entities.teams.isLoading);
  const tableHeaders = generateTableHeaders(() => null);
  const teams = useSelector((state: RootState) => generateDataSet(state.entities.teams.data));

  return (
    <div className={`${baseClass} body-wrap`}>
      <p className={`${baseClass}__page-description`}>
        Create, customize, and remove teams from Fleet.
      </p>
      <TableContainer
        columns={tableHeaders}
        data={teams}
        isLoading={loadingTableData}
        defaultSortHeader={'name'}
        defaultSortDirection={'desc'}
        inputPlaceHolder={'Search'}
        actionButtonText={'Create Team'}
        onActionButtonClick={() => console.log('create team')}
        onQueryChange={() => console.log('query change')}
        resultsTitle={'teams'}
        emptyComponent={EmptyTeams}
      />
    </div>
  );
};

export default TeamManagementPage;
