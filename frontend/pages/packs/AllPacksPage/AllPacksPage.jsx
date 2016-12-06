import React, { Component, PropTypes } from 'react';
import { connect } from 'react-redux';
import { push } from 'react-router-redux';

import Button from 'components/buttons/Button';
import entityGetter from 'redux/utilities/entityGetter';
import packActions from 'redux/nodes/entities/packs/actions';
import packInterface from 'interfaces/pack';
import paths from 'router/paths';

const baseClass = 'all-packs-page';

class AllPacksPage extends Component {

  static propTypes = {
    dispatch: PropTypes.func,
    packs: PropTypes.arrayOf(packInterface),
  }

  componentWillMount() {
    const { dispatch, packs } = this.props;
    if (!packs.length) {
      dispatch(packActions.loadAll());
    }

    return false;
  }

  goToNewPackPage = () => {
    const { dispatch } = this.props;
    const { NEW_PACK } = paths;

    dispatch(push(NEW_PACK));

    return false;
  }

  renderPack = (pack) => {
    return (
      <tr key={`pack-${pack.id}-table`}>
        <td>{pack.name}</td>
        <td>0?</td>
        <td>Enabled?</td>
        <td>Jason Meller?</td>
        <td>0?</td>
        <td>Yesterday?</td>
      </tr>
    );
  }

  render () {
    const { goToNewPackPage, renderPack } = this;
    const { packs } = this.props;

    return (
      <div className={`${baseClass} body-wrap`}>
        <div className={`${baseClass}__wrapper`}>
          <p className={`${baseClass}__title`}>
            Query Packs
          </p>
          <div className={`${baseClass}__new_pack`}>
            <Button
              text="CREATE NEW PACK"
              variant="brand"
              onClick={goToNewPackPage}
            />
          </div>
          <table className={`${baseClass}__table`}>
            <thead>
              <tr>
                <th>Name</th>
                <th>Queries</th>
                <th>Status</th>
                <th>Author</th>
                <th>Number of Hosts</th>
                <th>Last Updated</th>
              </tr>
            </thead>
            <tbody>
              {packs.map((pack) => {
                return renderPack(pack);
              })}
            </tbody>
          </table>
        </div>
      </div>
    );
  }
}

const mapStateToProps = (state) => {
  const { entities: packs } = entityGetter(state).get('packs');

  return { packs };
};

export default connect(mapStateToProps)(AllPacksPage);
