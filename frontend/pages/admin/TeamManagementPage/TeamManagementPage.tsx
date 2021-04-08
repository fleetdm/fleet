import React, { useState, useCallback } from 'react';
import { useSelector, useDispatch } from 'react-redux';


import { ITeam } from 'interfaces/team';
import teamActions from 'redux/nodes/entities/teams/actions';
import TableContainer from 'components/TableContainer';
import {ICreateTeamFormData} from './components/CreateTeamModal/CreateTeamModal';

import EmptyTeams from './components/EmptyTeams';
import CreateTeamModal from './components/CreateTeamModal';
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

const TeamManagementPage = (): JSX.Element => {
  const dispatch = useDispatch();
  const loadingTableData = useSelector((state: RootState) => state.entities.teams.isLoading);
  const tableHeaders = generateTableHeaders(() => null);
  const teams = useSelector((state: RootState) => generateDataSet(state.entities.teams.data));

  const [showCreateTeamModal, setShowCreateTeamModal] = useState(false);

  const onQueryChange = useCallback((queryData) => {
    const { pageIndex, pageSize, searchQuery } = queryData;
    dispatch(teamActions.loadAll(pageIndex, pageSize, searchQuery));
  }, [dispatch]);

  const toggleCreateTeamModal = useCallback(() => {
    setShowCreateTeamModal(!showCreateTeamModal);
  }, [showCreateTeamModal, setShowCreateTeamModal]);

  const onCreateSubmit = useCallback((formData: ICreateTeamFormData) => {
    dispatch(teamActions.create(formData)).then(() => {
      dispatch(teamActions.loadAll());
      // TODO: error handling
    });
    setShowCreateTeamModal(false);
  }, [dispatch, setShowCreateTeamModal]);

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
        onActionButtonClick={toggleCreateTeamModal}
        onQueryChange={onQueryChange}
        resultsTitle={'teams'}
        emptyComponent={EmptyTeams}
      />
      {showCreateTeamModal ?
        <CreateTeamModal
          onExit={toggleCreateTeamModal}
          onSubmit={onCreateSubmit}
        /> :
        null
      }
    </div>
  );
};

export default TeamManagementPage;
