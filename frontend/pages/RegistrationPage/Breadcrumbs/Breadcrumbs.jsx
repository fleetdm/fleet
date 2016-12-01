import React, { Component, PropTypes } from 'react';
import classnames from 'classnames';

class Breadcrumbs extends Component {
  static propTypes = {
    onClick: PropTypes.func,
    page: PropTypes.number,
  };

  static defaultProps = {
    page: 1,
  };

  onClick = (page) => {
    return (evt) => {
      evt.preventDefault();

      const { onClick: handleClick } = this.props;

      return handleClick(page);
    };
  }

  render () {
    const { onClick } = this;
    const { page } = this.props;
    const baseClass = 'registration-breadcrumbs';
    const pageBaseClass = `${baseClass}__page`;
    const page1ClassName = classnames(pageBaseClass, `${pageBaseClass}--1`, 'button--unstyled', {
      [`${pageBaseClass}--active`]: page === 1,
      [`${pageBaseClass}--complete`]: page > 1,
    });
    const page2ClassName = classnames(pageBaseClass, `${pageBaseClass}--2`, 'button--unstyled', {
      [`${pageBaseClass}--active`]: page === 2,
      [`${pageBaseClass}--complete`]: page > 2,
    });
    const page3ClassName = classnames(pageBaseClass, `${pageBaseClass}--3`, 'button--unstyled', {
      [`${pageBaseClass}--active`]: page === 3,
      [`${pageBaseClass}--complete`]: page > 3,
    });

    return (
      <div className={baseClass}>
        <button className={page1ClassName} onClick={onClick(1)}>Setup User</button>
        <button className={page2ClassName} onClick={onClick(2)}>Setup Organization</button>
        <button className={page3ClassName} onClick={onClick(3)}>Set Kolide URL</button>
      </div>
    );
  }
}

export default Breadcrumbs;
