import React, { Component } from "react";
import PropTypes from "prop-types";
import classnames from "classnames";
import { includes, orderBy } from "lodash";

import Checkbox from "components/forms/fields/Checkbox";
import packInterface from "interfaces/pack";
import Row from "components/packs/PacksList/Row";

const baseClass = "packs-list";

class PacksList extends Component {
  static propTypes = {
    allPacksChecked: PropTypes.bool,
    checkedPackIDs: PropTypes.arrayOf(PropTypes.number),
    className: PropTypes.string,
    onCheckAllPacks: PropTypes.func.isRequired,
    onCheckPack: PropTypes.func.isRequired,
    onDoubleClickPack: PropTypes.func,
    onSelectPack: PropTypes.func.isRequired,
    packs: PropTypes.arrayOf(packInterface),
    selectedPack: packInterface,
  };

  static defaultProps = {
    checkedPackIDs: [],
    packs: [],
    selectedPack: {},
  };

  renderHelpText = () => {
    const { packs } = this.props;

    if (packs.length) {
      return false;
    }

    return (
      <tr className={`${baseClass}__empty-table`}>
        <td colSpan={6}>
          <p>No packs available. Try creating one.</p>
        </td>
      </tr>
    );
  };

  renderPack = (pack) => {
    const {
      checkedPackIDs,
      onCheckPack,
      onDoubleClickPack,
      onSelectPack,
      selectedPack,
    } = this.props;
    const checked = includes(checkedPackIDs, pack.id);
    const selected = pack.id === selectedPack.id;

    return (
      <Row
        checked={checked}
        key={`pack-row-${pack.id}`}
        onCheck={onCheckPack}
        onSelect={onSelectPack}
        onDoubleClick={onDoubleClickPack}
        pack={pack}
        selected={selected}
      />
    );
  };

  render() {
    const { allPacksChecked, className, onCheckAllPacks, packs } = this.props;
    const { renderPack, renderHelpText } = this;
    const tableClassName = classnames(baseClass, className);

    return (
      <div className={`${baseClass}__wrapper`}>
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
              <th className={`${baseClass}__th ${baseClass}__th-pack-name`}>
                Pack name
              </th>
              <th className={`${baseClass}__th`}>Queries</th>
              <th className={`${baseClass}__th`}>Status</th>
              <th className={`${baseClass}__th`}>Hosts</th>
              <th className={`${baseClass}__th`}>Last modified</th>
            </tr>
          </thead>
          <tbody>
            {renderHelpText()}
            {!!packs.length &&
              orderBy(packs, ["name"]).map((pack) => renderPack(pack))}
          </tbody>
        </table>
      </div>
    );
  }
}

export default PacksList;
