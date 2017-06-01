import React, { Component, PropTypes } from 'react';
import { connect } from 'react-redux';
import { push } from 'react-router-redux';
import NumberPill from 'components/NumberPill';
import decoratorActions from 'redux/nodes/entities/decorators/actions';
import DecoratorRows from 'components/decorators/DecoratorRows';
import Modal from 'components/modals/Modal';
import Button from 'components/buttons/Button';
import DecoratorInfoSidePanel from 'components/side_panels/DecoratorInfoSidePanel';
import decoratorInterface from 'interfaces/decorators';
import entityGetter from 'redux/utilities/entityGetter';
import { renderFlash } from 'redux/nodes/notifications/actions';
import paths from 'router/paths';
import { pull, get, isEmpty } from 'lodash';


const baseClass = 'manage-decorators-page';

export class ManageDecoratorsPage extends Component {
  static propTypes = {
    dispatch: PropTypes.func,
    decorators: PropTypes.arrayOf(decoratorInterface),
    selectedDecorator: decoratorInterface,
  }

  constructor(props) {
    super(props);
    this.state = {
      allChecked: false,
      checkedIDs: [],
      showDeleteModal: false,
    };
  }

  componentWillMount() {
    const { dispatch } = this.props;
    dispatch(decoratorActions.loadAll())
      .catch(() => false);
  }

  onCheckDecorator = (checked, id) => {
    const { checkedIDs } = this.state;
    const newCheckedIDs = checked ? checkedIDs.concat(id) : pull(checkedIDs, id);
    this.setState({ allChecked: false, checkedIDs: newCheckedIDs });
  }

  onCheckAll = (checked) => {
    const { decorators } = this.props;
    if (checked) {
      const newCheckedIDs = decorators.filter((decorator) => {
        return !decorator.built_in;
      }).map((decorator) => {
        return decorator.id;
      });
      this.setState({ allChecked: true, checkedIDs: newCheckedIDs });
      return;
    }

    this.setState({ allChecked: false, checkedIDs: [] });
  }

  onSelectDecorator = (decorator) => {
    const { dispatch, selectedDecorator } = this.props;
    // if selected decorator is clicked again, this will undo the selected status
    if (selectedDecorator && (selectedDecorator.id === decorator.id)) {
      dispatch(push('/decorators/manage'));
      return false;
    }
    const path = {
      pathname: '/decorators/manage',
      query: { selectedDecorator: decorator.id },
    };
    dispatch(push(path));
    return false;
  }

  onDoubleClick = (decorator) => {
    const { dispatch } = this.props;
    const path = `/decorators/${decorator.id}`;
    dispatch(push(path));
    return false;
  }

  onDeleteDecorators = (evt) => {
    evt.preventDefault();
    const { checkedIDs } = this.state;
    const { dispatch } = this.props;
    const { destroy } = decoratorActions;

    const promises = checkedIDs.map((id: number) => {
      return dispatch(destroy({ id }));
    });
    return Promise.all(promises)
      .then(() => {
        dispatch(renderFlash('success', 'Successfully deleted selected decorators.'));
        this.setState({ checkedIDs: [], showDeleteModal: false, allChecked: false });
      })
      .catch(() => {
        dispatch(renderFlash('error', 'Something went wrong.'));
        this.setState({ showDeleteModal: false });
        return false;
      });
  }

  toggleDeleteModal = () => {
    const { showDeleteModal } = this.state;
    this.setState({ showDeleteModal: !showDeleteModal });
    return false;
  }

  showNewQueryPage = () => {
    const { dispatch } = this.props;
    const { NEW_DECORATOR } = paths;
    dispatch(push(NEW_DECORATOR));
    return false;
  }

  showEditDecorator = () => {
    const { selectedDecorator, dispatch } = this.props;
    const path = `/decorators/${selectedDecorator.id}`;
    dispatch(push(path));
    return false;
  }

  renderDeleteConfirmationModel = () => {
    const { showDeleteModal } = this.state;
    if (!showDeleteModal) {
      return false;
    }

    const { toggleDeleteModal, onDeleteDecorators } = this;
    return (
      <Modal
        title="Delete Decorator"
        onExit={toggleDeleteModal}
      >
        <p>Are you sure that you want to delete the selected decorators?</p>
        <div className={`${baseClass}__modal-btn-wrap`}>
          <Button onClick={onDeleteDecorators} variant="alert">Delete</Button>
          <Button onClick={toggleDeleteModal} variant="inverse">Cancel</Button>
        </div>
      </Modal>
    );
  }

  renderNewButton = () => {
    return (
      <Button
        variant="brand"
        onClick={this.showNewQueryPage}
      >
        CREATE DECORATOR
      </Button>
    );
  }

  renderDeleteButton = () => {
    return (
      <div>
        <Button
          onClick={this.toggleDeleteModal}
          variant="alert"
        >
          Delete
        </Button>
      </div>
    );
  }

  renderEditButton = () => {
    return (
      <div>
        <Button
          onClick={this.showEditDecorator}
        >
          EDIT DECORATOR
        </Button>
      </div>
    );
  }

  renderSidePanel = () => {
    return (
      <DecoratorInfoSidePanel />
    );
  }

  renderButtons = () => {
    const checkedCount = this.state.checkedIDs.length;
    const { selectedDecorator } = this.props;
    if (checkedCount) {
      return this.renderDeleteButton();
    }
    if (selectedDecorator) {
      return this.renderEditButton(selectedDecorator.id);
    }
    return this.renderNewButton();
  }

  render() {
    const { decorators, selectedDecorator } = this.props;
    const { checkedIDs, allChecked } = this.state;
    const {
      onCheckDecorator,
      onCheckAll,
      renderDeleteConfirmationModel,
      renderSidePanel,
      onSelectDecorator,
      onDoubleClick,
    } = this;

    return (
      <div className={`${baseClass} has-sidebar`}>
        <div className={`${baseClass}__wrapper body-wrap`}>
          <h1 className={`${baseClass}__title`}>
            <NumberPill number={decorators.length} /> Osquery Decorators
            <div className={`${baseClass}__top-buttons`}>
              {this.renderButtons()}
            </div>
          </h1>

          <DecoratorRows
            decorators={decorators}
            onCheckDecorator={onCheckDecorator}
            onCheckAll={onCheckAll}
            allChecked={allChecked}
            checkedDecoratorIDs={checkedIDs}
            onSelectDecorator={onSelectDecorator}
            onDoubleClick={onDoubleClick}
            selectedDecorator={selectedDecorator}
          />
        </div>
        {renderSidePanel()}
        {renderDeleteConfirmationModel()}
      </div>
    );
  }
}

const mapStateToProps = (state, { location }) => {
  const decoratorEntities = entityGetter(state).get('decorators');
  let { entities: decorators } = decoratorEntities;
  decorators = decorators.filter((decorator) => { return !isEmpty(decorator); });
  const selectedDecoratorID = get(location, 'query.selectedDecorator');
  const selectedDecorator = selectedDecoratorID && decoratorEntities.findBy({ id: selectedDecoratorID });
  return { decorators, selectedDecorator };
};

export default connect(mapStateToProps)(ManageDecoratorsPage);
