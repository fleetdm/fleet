import React, { PureComponent } from "react";
import PropTypes from "prop-types";

import Button from "components/buttons/Button";
import KolideIcon from "components/icons/KolideIcon";

const baseClass = "pagination";

class Pagination extends PureComponent {
  static propTypes = {
    currentPage: PropTypes.number,
    resultsPerPage: PropTypes.number,
    onPaginationChange: PropTypes.func,
    resultsOnCurrentPage: PropTypes.number,
  };

  disablePrev = () => {
    return this.props.currentPage === 0;
  };

  disableNext = () => {
    // NOTE: not sure why resultsOnCurrentPage is getting assigned undefined.
    // but this seems to work when there is no data in the table.
    return (
      this.props.resultsOnCurrentPage === undefined ||
      this.props.resultsOnCurrentPage < this.props.resultsPerPage
    );
  };

  render() {
    const { currentPage, onPaginationChange } = this.props;

    return (
      <div className={`${baseClass}__pager-wrap`}>
        <Button
          variant="unstyled"
          disabled={this.disablePrev()}
          onClick={() => onPaginationChange(currentPage - 1)}
        >
          <KolideIcon name="chevronleft" /> Prev
        </Button>
        <Button
          variant="unstyled"
          disabled={this.disableNext()}
          onClick={() => onPaginationChange(currentPage + 1)}
        >
          Next <KolideIcon name="chevronright" />
        </Button>
      </div>
    );
  }
}

export default Pagination;
