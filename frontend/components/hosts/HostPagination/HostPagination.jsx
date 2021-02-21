import React, { PureComponent } from 'react';
import PropTypes from 'prop-types';

import Button from 'components/buttons/Button';
import KolideIcon from 'components/icons/KolideIcon';

const baseClass = 'host-pagination';

class HostPagination extends PureComponent {
  static propTypes = {
    currentPage: PropTypes.number,
    hostsPerPage: PropTypes.number,
    onPaginationChange: PropTypes.func,
    hostsOnCurrentPage: PropTypes.number,
  };

  disablePrev = () => {
    return this.props.currentPage === 0;
  }

  disableNext = () => {
    console.log('HOSTS ON CURRENTPAG:', this.props.hostsOnCurrentPage);
    console.log('HOSTS PER PAGE:', this.props.hostsPerPage);
    // NOTE: not sure why hostsOnCurrentPage is getting assigned undefined.
    // but this seems to work when there is no data in the table.
    return this.props.hostsOnCurrentPage === undefined ||
      this.props.hostsOnCurrentPage < this.props.hostsPerPage;
  }

  render () {
    const {
      currentPage,
      onPaginationChange,
    } = this.props;

    return (
      <div className={`${baseClass}__pager-wrap`}>
        <Button variant={''} disabled={this.disablePrev()} onClick={() => onPaginationChange(currentPage - 1)}>
          <KolideIcon name="chevronleft" /> Prev
        </Button>
        <Button variant={''} disabled={this.disableNext()} onClick={() => onPaginationChange(currentPage + 1)}>
          Next <KolideIcon name="chevronright" />
        </Button>
      </div>
    );
  }
}

export default HostPagination;
