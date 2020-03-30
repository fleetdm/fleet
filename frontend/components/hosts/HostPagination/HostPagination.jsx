import React, { PureComponent } from 'react';
import PropTypes from 'prop-types';
import Pagination from 'rc-pagination';
import 'rc-pagination/assets/index.css';

import enUs from 'rc-pagination/lib/locale/en_US';

const baseClass = 'host-pagination';

class HostPagination extends PureComponent {
  static propTypes = {
    allHostCount: PropTypes.number,
    currentPage: PropTypes.number,
    hostsPerPage: PropTypes.number,
    onPaginationChange: PropTypes.func,
  };

  render () {
    const {
      allHostCount,
      currentPage,
      hostsPerPage,
      onPaginationChange,
    } = this.props;

    if (allHostCount === 0) {
      return false;
    }

    return (
      <div className={`${baseClass}__pager-wrap`}>
        <Pagination
          onChange={onPaginationChange}
          current={currentPage}
          total={allHostCount}
          pageSize={hostsPerPage}
          className={`${baseClass}__pagination`}
          locale={enUs}
          showLessItems
        />
      </div>
    );
  }
}

export default HostPagination;
