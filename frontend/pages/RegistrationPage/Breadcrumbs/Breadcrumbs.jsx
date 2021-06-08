import React, { Component } from "react";
import PropTypes from "prop-types";
import classnames from "classnames";

class Breadcrumbs extends Component {
  static propTypes = {
    onClick: PropTypes.func,
    pageProgress: PropTypes.number,
  };

  static defaultProps = {
    pageProgress: 1,
  };

  onClick = (page) => {
    return (evt) => {
      evt.preventDefault();

      const { onClick: handleClick } = this.props;

      return handleClick(page);
    };
  };

  render() {
    const { onClick } = this;
    const { pageProgress } = this.props;
    const baseClass = "registration-breadcrumbs";
    const pageBaseClass = `${baseClass}__page`;
    const page1ClassName = classnames(
      pageBaseClass,
      `${pageBaseClass}--1`,
      "button--unstyled",
      {
        [`${pageBaseClass}--active`]: pageProgress === 1,
        [`${pageBaseClass}--complete`]: pageProgress > 1,
      }
    );
    const page2ClassName = classnames(
      pageBaseClass,
      `${pageBaseClass}--2`,
      "button--unstyled",
      {
        [`${pageBaseClass}--active`]: pageProgress === 2,
        [`${pageBaseClass}--complete`]: pageProgress > 2,
      }
    );
    const page3ClassName = classnames(
      pageBaseClass,
      `${pageBaseClass}--3`,
      "button--unstyled",
      {
        [`${pageBaseClass}--active`]: pageProgress === 3,
        [`${pageBaseClass}--complete`]: pageProgress > 3,
      }
    );

    return (
      <div className={baseClass}>
        <button className={page1ClassName} onClick={onClick(1)}>
          Setup user
        </button>
        <button className={page2ClassName} onClick={onClick(2)}>
          Organization details
        </button>
        <button className={page3ClassName} onClick={onClick(3)}>
          Set Fleet URL
        </button>
      </div>
    );
  }
}

export default Breadcrumbs;
