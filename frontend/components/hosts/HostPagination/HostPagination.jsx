import React, { PureComponent } from 'react';
import PropTypes from 'prop-types';
import Pagination from 'rc-pagination';
import Select from 'react-select';
import 'rc-pagination/assets/index.css';

import enUs from 'rc-pagination/lib/locale/en_US';

const baseClass = 'host-pagination';

class HostPagination extends PureComponent {
  static propTypes = {
    allHostCount: PropTypes.number,
    currentPage: PropTypes.number,
    hostsPerPage: PropTypes.number,
    onPaginationChange: PropTypes.func,
    onPerPageChange: PropTypes.func,
  };

  render () {
    const {
      allHostCount,
      currentPage,
      hostsPerPage,
      onPaginationChange,
      onPerPageChange,
    } = this.props;

    const paginationSelectOpts = [
      { value: 20, label: '20' },
      { value: 100, label: '100' },
      { value: 500, label: '500' },
      { value: 1000, label: '1,000' },
    ];

    const humanPage = currentPage + 1;
    const startRange = (currentPage * hostsPerPage) + 1;
    const endRange = Math.min(humanPage * hostsPerPage, allHostCount);

    if (allHostCount === 0) {
      return false;
    }

    return (
      <div className={`${baseClass}__pager-wrap`}>
        <Pagination
          onChange={onPaginationChange}
          current={humanPage}
          total={allHostCount}
          pageSize={hostsPerPage}
          className={`${baseClass}__pagination`}
          locale={enUs}
          showLessItems
        />
        <p className={`${baseClass}__pager-range`}>{`${startRange} - ${endRange} of ${allHostCount} hosts`}</p>
        <div className={`${baseClass}__pager-count`}>
          <Select
            name="pager-host-count"
            value={hostsPerPage}
            options={paginationSelectOpts}
            onChange={onPerPageChange}
            className={`${baseClass}__count-select`}
            clearable={false}
          /> <span>Hosts per page</span>
        </div>
      </div>
    );
  }
}

export default HostPagination;
