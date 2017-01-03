import React, { Component, PropTypes } from 'react';
import classnames from 'classnames';
import { includes } from 'lodash';

import Checkbox from 'components/forms/fields/Checkbox';
import packInterface from 'interfaces/pack';
import Row from 'components/packs/PacksList/Row';

const baseClass = 'packs-list';

class PacksList extends Component {
  static propTypes = {
    allPacksChecked: PropTypes.bool,
    checkedPackIDs: PropTypes.arrayOf(PropTypes.number),
    className: PropTypes.string,
    onCheckAllPacks: PropTypes.func.isRequired,
    onCheckPack: PropTypes.func.isRequired,
    onSelectPack: PropTypes.func.isRequired,
    packs: PropTypes.arrayOf(packInterface),
    selectedPack: packInterface,
  };

  static defaultProps = {
    checkedPackIDs: [],
    packs: [],
    selectedPack: {},
  };

  renderPack = (pack) => {
    const { checkedPackIDs, onCheckPack, onSelectPack, selectedPack } = this.props;
    const checked = includes(checkedPackIDs, pack.id);
    const selected = pack.id === selectedPack.id;

    return (
      <Row
        checked={checked}
        key={`pack-row-${pack.id}`}
        onCheck={onCheckPack}
        onSelect={onSelectPack}
        pack={pack}
        selected={selected}
      />
    );
  }

  render () {
    const { allPacksChecked, className, onCheckAllPacks, packs } = this.props;
    const { renderPack } = this;
    const tableClassName = classnames(baseClass, className);

    return (
      <table className={tableClassName}>
        <thead>
          <tr>
            <th className={`${baseClass}__th`}>
              <Checkbox
                name="select-all-packs"
                onChange={onCheckAllPacks}
                value={allPacksChecked}
                wrapperClassName={`${baseClass}__select-all`}
              />
            </th>
            <th className={`${baseClass}__th ${baseClass}__th-pack-name`}>Pack Name</th>
            <th className={`${baseClass}__th`}>Queries</th>
            <th className={`${baseClass}__th`}>Status</th>
            <th className={`${baseClass}__th`}>Hosts</th>
            <th className={`${baseClass}__th`}>Last Modified</th>
          </tr>
        </thead>
        <tbody>
          {packs.map(pack => renderPack(pack))}
        </tbody>
      </table>
    );
  }
}

export default PacksList;
