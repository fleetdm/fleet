import React, { PureComponent } from "react";
import PropTypes from "prop-types";

import Button from "components/buttons/Button";
import FleetIcon from "components/icons/FleetIcon";

const baseClass = "pagination";

/**
 * WARNING: DEPRICATED:
 * This pagination component is DEPRICATED. It is being kept around until we replace its
 * use. For now use the Pagination component in the pages/ManageControlsPage/components.
 */
class Pagination extends PureComponent {
  static propTypes = {
    currentPage: PropTypes.number,
    resultsPerPage: PropTypes.number,
    onPaginationChange: PropTypes.func,
    resultsOnCurrentPage: PropTypes.number,
    disableNextPage: PropTypes.bool,
  };

  disablePrev = () => {
    return this.props.currentPage === 0;
  };

  disableNext = () => {
    // NOTE: Disable next page is passed through from api metadata
    if (this.props.disableNextPage !== undefined) {
      return this.props.disableNextPage;
    }
    // NOTE: not sure why resultsOnCurrentPage is getting assigned undefined.
    // but this seems to work when there is no data in the table.
    return (
      this.props.resultsOnCurrentPage === undefined ||
      this.props.resultsOnCurrentPage < this.props.resultsPerPage ||
      this.props.disableNextPage
    );
  };

  render() {
    const { currentPage, onPaginationChange } = this.props;

    return (
      <div className={`${baseClass}__pager-wrap`}>
        <Button
          variant="unstyled"
          disabled={this.disablePrev()}
          onClick={() => onPaginationChange(parseInt(currentPage, 10) - 1)}
        >
          <FleetIcon name="chevronleft" /> Previous
        </Button>
        <Button
          variant="unstyled"
          disabled={this.disableNext()}
          onClick={() => onPaginationChange(parseInt(currentPage, 10) + 1)}
        >
          Next <FleetIcon name="chevronright" />
        </Button>
      </div>
    );
  }
}

export default Pagination;
