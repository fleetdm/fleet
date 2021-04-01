import React, { Component } from 'react';
import PropTypes from 'prop-types';
import { connect } from 'react-redux';
import { concat, includes, difference } from 'lodash';
import { push } from 'react-router-redux';
import memoize from 'memoize-one';

import Button from 'components/buttons/Button';
import configInterface from 'interfaces/config';
import deepDifference from 'utilities/deep_difference';
import entityGetter from 'redux/utilities/entityGetter';
import inviteActions from 'redux/nodes/entities/invites/actions';
import inviteInterface from 'interfaces/invite';
import Modal from 'components/modals/Modal';
import paths from 'router/paths';
import { renderFlash } from 'redux/nodes/notifications/actions';
import WarningBanner from 'components/WarningBanner';
import { updateUser } from 'redux/nodes/auth/actions';
import userActions from 'redux/nodes/entities/users/actions';
import userInterface from 'interfaces/user';

import CreateUserForm from './components/CreateUserForm';
import { usersTableHeaders, combineDataSets } from './UsersTableConfig';
import TableContainer from '../../../components/TableContainer';

const baseClass = 'user-management';

const EmptyUsers = () => {
  return (
    <p>no users</p>
  );
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
      usersEditing: [],
    };
  }

  onUserActionSelect = (user, action) => {
    const { currentUser, dispatch } = this.props;
    const { enableUser, updateAdmin, requirePasswordReset } = userActions;

    if (action) {
      switch (action) {
        case 'demote_user': {
          if (currentUser.id === user.id) {
            return dispatch(renderFlash('error', 'You cannot demote yourself'));
          }
          return dispatch(updateAdmin(user, { admin: false }))
            .then(() => {
              return dispatch(renderFlash('success', 'User demoted', updateAdmin(user, { admin: true })));
            });
        }
        case 'disable_account': {
          if (currentUser.id === user.id) {
            return dispatch(renderFlash('error', 'You cannot disable your own account'));
          }
          return dispatch(userActions.enableUser(user, { enabled: false }))
            .then(() => {
              return dispatch(renderFlash('success', 'User account disabled', enableUser(user, { enabled: true })));
            });
        }
        case 'enable_account':
          return dispatch(enableUser(user, { enabled: true }))
            .then(() => {
              return dispatch(renderFlash('success', 'User account enabled', enableUser(user, { enabled: false })));
            });
        case 'promote_user':
          return dispatch(updateAdmin(user, { admin: true }))
            .then(() => {
              return dispatch(renderFlash('success', 'User promoted to admin', updateAdmin(user, { admin: false })));
            });
        case 'reset_password':
          return dispatch(requirePasswordReset(user, { require: true }))
            .then(() => {
              return dispatch(renderFlash('success', 'User required to reset password', requirePasswordReset(user, { require: false })));
            });
        case 'revert_invitation':
          return dispatch(inviteActions.silentDestroy(user))
            .then(() => dispatch(renderFlash('success', 'Invite revoked')))
            .catch(() => dispatch(renderFlash('error', 'Invite could not be revoked')));
        default:
          return false;
      }
    }

    return false;
  }

  onEditUser = (user, updatedUser) => {
    const { currentUser, dispatch } = this.props;
    const { onToggleEditUser } = this;
    const { silentUpdate } = userActions;
    const updatedAttrs = deepDifference(updatedUser, user);

    if (currentUser.id === user.id) {
      return dispatch(updateUser(user, updatedAttrs))
        .then(() => {
          dispatch(renderFlash('success', 'User updated', updateUser(user, user)));
          onToggleEditUser(user);

          return false;
        })
        .catch(() => false);
    }

    return dispatch(silentUpdate(user, updatedAttrs))
      .then(() => {
        dispatch(renderFlash('success', 'User updated', silentUpdate(user, user)));
        onToggleEditUser(user);

        return false;
      })
      .catch(() => false);
  }

  onInviteUserSubmit = (formData) => {
    const { dispatch } = this.props;

    dispatch(inviteActions.silentCreate(formData))
      .then(() => {
        return this.toggleCreateUserModal();
      })
      .catch(() => false);
  }

  onInviteCancel = (evt) => {
    evt.preventDefault();

    return this.toggleCreateUserModal();
  }

  onToggleEditUser = (user) => {
    const { dispatch } = this.props;
    const { usersEditing } = this.state;
    let updatedUsersEditing = [];

    dispatch(userActions.clearErrors());

    if (includes(usersEditing, user.id)) {
      updatedUsersEditing = difference(usersEditing, [user.id]);
    } else {
      updatedUsersEditing = concat(usersEditing, [user.id]);
    }

    this.setState({ usersEditing: updatedUsersEditing });
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
    dispatch(inviteActions.loadAll());
  }

  onActionSelect = () => {
    console.log('action selected');
  }

  goToAppConfigPage = (evt) => {
    evt.preventDefault();

    const { ADMIN_SETTINGS } = paths;
    const { dispatch } = this.props;

    dispatch(push(ADMIN_SETTINGS));
  }

  toggleCreateUserModal = () => {
    const { showCreateUserModal } = this.state;

    this.setState({
      showCreateUserModal: !showCreateUserModal,
    });

    return false;
  }

  combineUsersAndInvites = memoize(
    (users, invites, currentUserId, onActionSelect) => {
      return combineDataSets(users, invites, currentUserId, onActionSelect);
    },
  )

  renderModal = () => {
    const { currentUser, inviteErrors } = this.props;
    const { showCreateUserModal } = this.state;
    const { onInviteCancel, onInviteUserSubmit, toggleCreateUserModal } = this;
    const ssoEnabledForApp = this.props.config.enable_sso;

    if (!showCreateUserModal) {
      return false;
    }

    return (
      <Modal
        title="Create new user"
        onExit={toggleCreateUserModal}
        className={`${baseClass}__create-user-modal`}
      >
        <CreateUserForm
          serverErrors={inviteErrors}
          createdBy={currentUser}
          onCancel={onInviteCancel}
          onSubmit={onInviteUserSubmit}
          canUseSSO={ssoEnabledForApp}
          availableTeams={currentUser.teams}
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
    const { renderModal, renderSmtpWarning, toggleCreateUserModal, onTableQueryChange, onActionSelect } = this;
    const { config, loadingTableData, users, invites, currentUser } = this.props;

    let tableData = [];
    if (!loadingTableData) {
      tableData = this.combineUsersAndInvites(users, invites, currentUser.id, onActionSelect);
    }

    console.log(tableData);

    return (
      <div className={`${baseClass} body-wrap`}>
        <p className={`${baseClass}__page-description`}>Create new users, customize user permissions, and remove users from Fleet.</p>
        {renderSmtpWarning()}
        {/* TODO: find a way to move these controls into the table component */}
        <TableContainer
          columns={usersTableHeaders}
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
        {renderModal()}
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
  const { entities: invites } = stateEntityGetter.get('invites');
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
