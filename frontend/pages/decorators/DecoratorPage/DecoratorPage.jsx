import React, { Component, PropTypes } from 'react';
import { connect } from 'react-redux';
import { noop } from 'lodash';
import { push } from 'react-router-redux';
import debounce from 'utilities/debounce';
import decoratorActions from 'redux/nodes/entities/decorators/actions';
import DecoratorForm from 'components/forms/decorators/DecoratorForm';
import QuerySidePanel from 'components/side_panels/QuerySidePanel';
import osqueryTableInterface from 'interfaces/osquery_table';
import { renderFlash } from 'redux/nodes/notifications/actions';
import { selectOsqueryTable } from 'redux/nodes/components/Decorators/actions';
import { decoratorInterface } from 'interfaces/decorators';
import entityGetter from 'redux/utilities/entityGetter';


const baseClass = 'decorator-page';

export class DecoratorPage extends Component {
  static propTypes = {
    selectedOsqueryTable: osqueryTableInterface,
    dispatch: PropTypes.func,
    decorator: decoratorInterface,
    newDecorator: PropTypes.bool,
  };

  onSubmitNew = debounce((formData) => {
    const { dispatch } = this.props;
    formData.interval = Number(formData.interval);
    return dispatch(decoratorActions.create(formData))
      .then(() => {
        dispatch(push('/decorators/manage'));
      })
      .catch(() => false);
  })

  onSubmitUpdate = debounce((formData) => {
    const { dispatch } = this.props;
    formData.interval = Number(formData.interval);
    return dispatch(decoratorActions.update(formData))
      .then(() => {
        dispatch(push('/decorators/manage'));
      })
      .catch(() => false);
  })

  onCancel = () => {
    const { dispatch } = this.props;
    dispatch(push('/decorators/manage'));
    dispatch(renderFlash('success', 'Decorator canceled!'));
  }

  onOsqueryTableSelect = (tableName) => {
    const { dispatch } = this.props;
    dispatch(selectOsqueryTable(tableName));
    return false;
  }

  render() {
    const {
      onSubmitNew,
      onSubmitUpdate,
      onCancel,
      onOsqueryTableSelect,
    } = this;

    const {
      selectedOsqueryTable,
      decorator,
      newDecorator,
    } = this.props;

    const onSubmit = newDecorator ? onSubmitNew : onSubmitUpdate;

    return (
      <div className={`${baseClass} has-sidebar`}>
        <div className={`${baseClass}__content`}>
          <div className={`${baseClass}__form body-wrap`}>
            <DecoratorForm
              formData={decorator}
              handleSubmit={onSubmit}
              handleCancel={onCancel}
              newDecorator={newDecorator}
            />
          </div>
        </div>
        <QuerySidePanel
          onOsqueryTableSelect={onOsqueryTableSelect}
          onTextEditorInputChange={noop}
          selectedOsqueryTable={selectedOsqueryTable}
        />
      </div>
    );
  }
}

const mapStateToProps = (state, ownProps) => {
  const { queryText, selectedOsqueryTable } = state.components.Decorators;
  const { id: decoratorID } = ownProps.params;
  let decorator = { built_in: false, type: 'load', query: '', interval: 0, name: '' };
  let newDecorator = true;
  if (decoratorID) {
    decorator = entityGetter(state).get('decorators').findBy({ id: decoratorID });
    newDecorator = false;
  }
  return {
    queryText,
    selectedOsqueryTable,
    decorator,
    newDecorator,
  };
};

export default connect(mapStateToProps)(DecoratorPage);
