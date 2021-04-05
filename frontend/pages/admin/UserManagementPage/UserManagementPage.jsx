import React, { Component } from 'react';
import PropTypes from 'prop-types';
import { connect } from 'react-redux';
import { isEqual } from 'lodash';
import { push } from 'react-router-redux';
import memoize from 'memoize-one';

import Button from 'components/buttons/Button';
import TableContainer from 'components/TableContainer';
import Modal from 'components/modals/Modal';
import WarningBanner from 'components/WarningBanner';
import inviteInterface from 'interfaces/invite';
import configInterface from 'interfaces/config';
import userInterface from 'interfaces/user';
import paths from 'router/paths';
import entityGetter from 'redux/utilities/entityGetter';
import inviteActions from 'redux/nodes/entities/invites/actions';
import { renderFlash } from 'redux/nodes/notifications/actions';
import { updateUser } from 'redux/nodes/auth/actions';
import userActions from 'redux/nodes/entities/users/actions';

import UserForm from './components/UserForm';
import EmptyUsers from './components/EmptyUsers';
import { generateTableHeaders, combineDataSets } from './UsersTableConfig';

const baseClass = 'user-management';

const generateUpdateData = (currentUserData, formData) => {
  const updatableFields = ['global_role', 'teams', 'name', 'email', 'sso_enabled'];
  return Object.keys(formData).reduce((updatedAttributes, attr) => {
    // attribute can be updated and is different from the current value.
    if (updatableFields.includes(attr) && !isEqual(formData[attr], currentUserData[attr])) {
      updatedAttributes[attr] = formData[attr];
    }
    return updatedAttributes;
  }, {});
};

export class UserManagementPage extends Component {
  static propTypes = {
    appConfigLoading: PropTypes.bool,
    config: configInterface,
    currentUser: userInterface,
    dispatch: PropTypes.func,
    loadingTableData: PropTypes.bool,
    invites: PropTypes.arrayOf(inviteInterface),
    inviteErrors: PropTypes.shape({
      base: PropTypes.string,
      email: PropTypes.string,
    }),
    users: PropTypes.arrayOf(userInterface),
    userErrors: PropTypes.shape({
      base: PropTypes.string,
      name: PropTypes.string,
      username: PropTypes.string,
    }),
  };

  constructor (props) {
    super(props);

    this.state = {
      showCreateUserModal: false,
      showEditUserModal: false,
      userEditing: null,
      usersEditing: [],
    };

    // done as an instance variable as these headers will not change, so dont
    // want to recalculate on re-renders.
    this.tableHeaders = generateTableHeaders(this.onActionSelect);
  }

  onEditUser = (formData) => {
    const { currentUser, users, invites, dispatch } = this.props;
    const { userEditing } = this.state;
    const { toggleEditUserModal, getUser } = this;

    const userData = getUser(userEditing.type, userEditing.id);

    const updatedAttrs = generateUpdateData(userData, formData);
    if (currentUser.id === userEditing.id) {
      return dispatch(updateUser(userData, updatedAttrs))
        .then(() => {
          dispatch(renderFlash('success', 'User updated', updateUser(formData, formData)));
          toggleEditUserModal();
        })
        .catch(() => false);
    }

    return dispatch(userActions.silentUpdate(userData, formData))
      .then(() => {
        dispatch(renderFlash('success', 'User updated', userActions.silentUpdate(formData, formData)));
        toggleEditUserModal();
      })
      .catch(() => false);
  }

  onCreateUserSubmit = (formData) => {
    const { dispatch } = this.props;
    // Do some data formatting adding `invited_by` for the request to be correct.
    const requestData = {
      ...formData,
      invited_by: formData.currentUserId,
    };
    delete requestData.currentUserId; // dont need this for the request.
    dispatch(inviteActions.silentCreate(requestData))
      .then(() => {
        this.toggleCreateUserModal();
      })
      .catch(() => false);
  }

  onCreateCancel = (evt) => {
    evt.preventDefault();
    this.toggleCreateUserModal();
  }

  // NOTE: this is called once on the initial rendering. The initial render of
  // the TableContainer child component calls this handler.
  onTableQueryChange = (queryData) => {
    const { dispatch } = this.props;
    const { pageIndex, pageSize, searchQuery, sortHeader, sortDirection } = queryData;
    let sortBy = [];
    if (sortHeader !== '') {
      sortBy = [{ id: sortHeader, direction: sortDirection }];
    }
    dispatch(userActions.loadAll(pageIndex, pageSize, undefined, searchQuery, sortBy));
    dispatch(inviteActions.loadAll(pageIndex, pageSize, searchQuery, sortBy)); // TODO: add search params when API supports it
  }

  onActionSelect = (action, user) => {
    const { dispatch } = this.props;
    const { toggleEditUserModal } = this;
    const { requirePasswordReset } = userActions;

    switch (action) {
      case ('edit'):
        toggleEditUserModal(user);
        break;
      case ('delete'):
        break;
      case ('passwordReset'):
        return dispatch(requirePasswordReset(user.id, { require: true }))
          .then(() => {
            return dispatch(
              renderFlash(
                'success', 'User required to reset password',
                requirePasswordReset(user.id, { require: false }), // this is an undo action.
              ),
            );
          });
      default:
    }
  }

  getUser = (type, id) => {
    const { users, invites } = this.props;
    let userData;
    if (type === 'user') {
      userData = users.find(user => user.id === id);
    } else {
      userData = invites.find(invite => invite.id === id);
    }
    return userData;
  }

  toggleCreateUserModal = () => {
    const { showCreateUserModal } = this.state;
    this.setState({
      showCreateUserModal: !showCreateUserModal,
    });
  }

  toggleEditUserModal = (user) => {
    const { showEditUserModal } = this.state;
    this.setState({
      showEditUserModal: !showEditUserModal,
      userEditing: !showEditUserModal ? user : null,
    });
  }

  combineUsersAndInvites = memoize(
    (users, invites, currentUserId, onActionSelect) => {
      return combineDataSets(users, invites, currentUserId, onActionSelect);
    },
  )

  goToAppConfigPage = (evt) => {
    evt.preventDefault();

    const { ADMIN_SETTINGS } = paths;
    const { dispatch } = this.props;

    dispatch(push(ADMIN_SETTINGS));
  }

  renderEditUserModal = () => {
    const { currentUser, inviteErrors, config } = this.props;
    const { showEditUserModal, userEditing } = this.state;
    const { onEditUser, toggleEditUserModal, getUser } = this;

    if (!showEditUserModal) return null;

    const userData = getUser(userEditing.type, userEditing.id);

    return (
      <Modal
        title="Edit user"
        onExit={toggleEditUserModal}
        className={`${baseClass}__edit-user-modal`}
      >
        <UserForm
          serverErrors={inviteErrors}
          defaultEmail={userData.email}
          defaultName={userData.name}
          defaultGlobalRole={userData.global_role}
          defaultTeams={userData.teams}
          createdBy={currentUser}
          currentUserId={currentUser.id}
          onCancel={toggleEditUserModal}
          onSubmit={onEditUser}
          canUseSSO={config.enable_sso}
          availableTeams={userData.teams}
          submitText={'Edit'}
        />
      </Modal>
    );
  }

  renderCreateUserModal = () => {
    const { currentUser, inviteErrors, config } = this.props;
    const { showCreateUserModal } = this.state;
    const { onCreateUserSubmit, toggleCreateUserModal } = this;

    if (!showCreateUserModal) {
      return false;
    }

    return (
      <Modal
        title="Create user"
        onExit={toggleCreateUserModal}
        className={`${baseClass}__create-user-modal`}
      >
        <UserForm
          serverErrors={inviteErrors}
          createdBy={currentUser}
          currentUserId={currentUser.id}
          onCancel={toggleCreateUserModal}
          onSubmit={onCreateUserSubmit}
          canUseSSO={config.enable_sso}
          availableTeams={currentUser.teams}
          defaultTeams={[]}
          submitText={'Create'}
        />
      </Modal>
    );
  };

  renderSmtpWarning = () => {
    const { appConfigLoading, config } = this.props;
    const { goToAppConfigPage } = this;

    if (appConfigLoading) {
      return false;
    }

    return (
      <div className={`${baseClass}__smtp-warning-wrapper`}>
        <WarningBanner
          shouldShowWarning={!config.configured}
        >
          <span>SMTP is not currently configured in Fleet. The &quot;Create User&quot; feature requires that SMTP is configured in order to send invitation emails.</span>
          <Button
            className={`${baseClass}__config-button`}
            onClick={goToAppConfigPage}
            variant={'unstyled'}
          >
            Configure SMTP
          </Button>
        </WarningBanner>
      </div>
    );
  }

  render () {
    const {
      tableHeaders,
      renderCreateUserModal,
      renderEditUserModal,
      renderSmtpWarning,
      toggleCreateUserModal,
      onTableQueryChange,
      onActionSelect,
    } = this;
    const { config, loadingTableData, users, invites, currentUser } = this.props;

    let tableData = [];
    if (!loadingTableData) {
      tableData = this.combineUsersAndInvites(users, invites, currentUser.id, onActionSelect);
    }

    return (
      <div className={`${baseClass} body-wrap`}>
        <p className={`${baseClass}__page-description`}>Create new users, customize user permissions, and remove users from Fleet.</p>
        {renderSmtpWarning()}
        {/* TODO: find a way to move these controls into the table component */}
        <TableContainer
          columns={tableHeaders}
          data={tableData}
          isLoading={loadingTableData}
          defaultSortHeader={'name'}
          defaultSortDirection={'desc'}
          inputPlaceHolder={'Search'}
          disableActionButton={!config.configured}
          actionButtonText={'Create User'}
          onActionButtonClick={toggleCreateUserModal}
          onQueryChange={onTableQueryChange}
          resultsTitle={'rows'}
          emptyComponent={EmptyUsers}
        />
        {renderCreateUserModal()}
        {renderEditUserModal()}
      </div>
    );
  }
}

const mapStateToProps = (state) => {
  const stateEntityGetter = entityGetter(state);
  const { config } = state.app;
  const { loading: appConfigLoading } = state.app;
  const { user: currentUser } = state.auth;
  const { entities: users } = stateEntityGetter.get('users');
  // const { entities: invites } = stateEntityGetter.get('invites');
  const invites = [{
    name: 'Gabriel Fernandez', email: 'gabriel+fev@fleetdm.com', id: 2, teams: [{ name: 'Test Team', role: 'maintainer', id: 1 }], global_role: null,
  }];
  const { errors: inviteErrors, loading: loadingInvites } = state.entities.invites;
  const { errors: userErrors, loading: loadingUsers } = state.entities.users;
  const loadingTableData = loadingUsers || loadingInvites;

  return {
    appConfigLoading,
    config,
    currentUser,
    users,
    userErrors,
    invites,
    inviteErrors,
    loadingTableData,
  };
};

export default connect(mapStateToProps)(UserManagementPage);
