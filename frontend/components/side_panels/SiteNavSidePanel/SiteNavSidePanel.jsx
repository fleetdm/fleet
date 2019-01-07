import React, { Component } from 'react';
import PropTypes from 'prop-types';
import classnames from 'classnames';

import userInterface from 'interfaces/user';
import Icon from 'components/icons/Icon';

import navItems from './navItems';

class SiteNavSidePanel extends Component {
  static propTypes = {
    onNavItemClick: PropTypes.func,
    pathname: PropTypes.string,
    user: userInterface,
  };

  constructor (props) {
    super(props);

    const { user: { admin } } = this.props;

    this.userNavItems = navItems(admin);

    this.state = {
      showSubItems: false,
      userMenuOpened: false,
    };
  }

  setSubNavClass = (showSubItems) => {
    return showSubItems ? 'sub-nav sub-nav--expanded' : 'sub-nav';
  }

  toggleShowSubItems = (showSubItems) => {
    return (evt) => {
      evt.preventDefault();

      this.setState({ showSubItems });

      return false;
    };
  }

  toggleUserMenu = () => {
    const { userMenuOpened } = this.state;

    this.setState({ userMenuOpened: !userMenuOpened });
  }

  renderNavItem = (navItem) => {
    const { icon, name, subItems } = navItem;
    const { onNavItemClick, pathname } = this.props;
    const { renderSubItems } = this;
    const active = navItem.location.regex.test(pathname);
    const navItemBaseClass = 'site-nav-item';

    const navItemClasses = classnames(
      `${navItemBaseClass}`,
      {
        [`${navItemBaseClass}--active`]: active,
        [`${navItemBaseClass}--single`]: subItems.length === 0,
      }
    );

    return (
      <li className={navItemClasses} key={`nav-item-${name}`}>
        <button
          className={`${navItemBaseClass}__button button button--unstyled`}
          onClick={onNavItemClick(navItem.location.pathname)}
          style={{ width: '100%' }}
        >
          <Icon name={icon} className={`${navItemBaseClass}__icon`} />
          <span className={`${navItemBaseClass}__name`}>
            {name}
          </span>
        </button>
        {active && renderSubItems(subItems)}
      </li>
    );
  }

  renderNavItems = () => {
    const { renderNavItem, userNavItems } = this;

    return (
      <ul className="site-nav-list">
        {userNavItems.map((navItem) => {
          return renderNavItem(navItem);
        })}
      </ul>
    );
  }

  renderSubItem = (subItem) => {
    const { icon, name } = subItem;
    const { onNavItemClick, pathname } = this.props;
    const active = subItem.location.regex.test(pathname);

    const baseSubItemClass = 'site-sub-item';

    const baseSubItemItemClass = classnames(
      `${baseSubItemClass}`,
      { [`${baseSubItemClass}--active`]: active }
    );

    return (
      <li
        key={name}
        className={baseSubItemItemClass}
      >
        <button
          className={`${baseSubItemClass}__button button button--unstyled`}
          key={`sub-item-${name}`}
          onClick={onNavItemClick(subItem.location.pathname)}
        >
          <span className={`${baseSubItemClass}__name`}>{name}</span>
          <span className={`${baseSubItemClass}__icon`}><Icon name={icon} /></span>
        </button>
      </li>
    );
  }

  renderSubItems = (subItems) => {
    const { renderSubItem, setSubNavClass } = this;
    const { showSubItems } = this.state;

    const baseSubItemsClass = 'site-sub-items';

    const subItemListClasses = classnames(
      `${baseSubItemsClass}__list`,
      { [`${baseSubItemsClass}__list--expanded`]: showSubItems }
    );

    if (!subItems.length) return false;

    return (
      <div className={`${setSubNavClass(showSubItems)} ${baseSubItemsClass}`}>
        <ul className={subItemListClasses}>
          {subItems.map((subItem) => {
            return renderSubItem(subItem);
          })}
        </ul>
      </div>
    );
  }

  render () {
    const { renderNavItems } = this;

    return (
      renderNavItems()
    );
  }
}

export default SiteNavSidePanel;
