import React, { Component, PropTypes } from 'react';
import { connect } from 'react-redux';
import { isEqual } from 'lodash';
import { push } from 'react-router-redux';
import classnames from 'classnames';

import { activeTabFromPathname, activeSubTabFromPathname } from './helpers';
import configInterface from '../../../interfaces/config';
import kolideLogo from '../../../../assets/images/kolide-logo.svg';
import navItems from './navItems';
import userInterface from '../../../interfaces/user';

class SiteNavSidePanel extends Component {
  static propTypes = {
    config: configInterface,
    dispatch: PropTypes.func,
    pathname: PropTypes.string,
    user: userInterface,
  };

  constructor (props) {
    super(props);

    const { pathname, user: { admin } } = this.props;

    this.userNavItems = navItems(admin);

    const activeTab = activeTabFromPathname(this.userNavItems, pathname);
    const activeSubItem = activeSubTabFromPathname(activeTab, pathname);

    this.state = {
      activeTab,
      activeSubItem,
      showSubItems: false,
    };
  }

  componentWillReceiveProps (nextProps) {
    if (isEqual(nextProps, this.props)) return false;

    const { pathname } = nextProps;

    const activeTab = activeTabFromPathname(this.userNavItems, pathname);
    const activeSubItem = activeSubTabFromPathname(activeTab, pathname);

    this.setState({
      activeTab,
      activeSubItem,
    });

    return false;
  }

  setActiveSubItem = (activeSubItem) => {
    return (evt) => {
      evt.preventDefault();

      const { dispatch } = this.props;
      const { path: { location: tabLocation } } = activeSubItem;

      if (!tabLocation) return false;

      dispatch(push(tabLocation));
      return false;
    };
  }

  setActiveTab = (activeTab) => {
    return (evt) => {
      evt.preventDefault();

      const { pathname, dispatch } = this.props;
      const activeSubItem = activeSubTabFromPathname(activeTab, pathname);
      const { path: { location: tabLocation } } = activeSubItem;

      this.setState({ activeTab, activeSubItem });
      if (tabLocation) return dispatch(push(tabLocation));

      return false;
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

  renderHeader = () => {
    const {
      config: {
        org_name: orgName,
      },
      user: {
        enabled,
        username,
      },
    } = this.props;

    const headerBaseClass = 'site-nav-header';

    const userStatusClass = classnames(
      `${headerBaseClass}__user-status`,
      { [`${headerBaseClass}__user-status--enabled`]: enabled }
    );

    return (
      <header className={headerBaseClass}>
        <img
          alt="Company logo"
          src={kolideLogo}
          className={`${headerBaseClass}__logo`}
        />
        <h1 className={`${headerBaseClass}__org-name`}>{orgName}</h1>
        <div className={userStatusClass} />
        <h2 className={`${headerBaseClass}__username`}>{username}</h2>
        <i className={`${headerBaseClass}__org-chevron kolidecon-chevrondownbold`} />
      </header>
    );
  }

  renderNavItem = (navItem) => {
    const { activeTab = {} } = this.state;
    const { icon, name, subItems } = navItem;
    const active = activeTab.name === name;
    const { renderSubItems, setActiveTab } = this;

    const navItemBaseClass = 'site-nav-item';

    const navItemClasses = classnames(
      `${navItemBaseClass}__item`,
      { [`${navItemBaseClass}__item--active`]: active }
    );

    return (
      <div className={navItemBaseClass} key={`nav-item-${name}`}>
        <button
          className="button button--unstyled"
          onClick={setActiveTab(navItem)}
          style={{ width: '100%' }}
        >
          {active && <div className={`${navItemBaseClass}__active-nav`} />}
          <li
            key={name}
            className={navItemClasses}
          >
            <div style={{ position: 'relative', textAlign: 'left' }}>
              <i className={`${navItemBaseClass}__icon ${icon}`} />
              <span className={`${navItemBaseClass}__name`}>
                {name}
              </span>
            </div>
          </li>
        </button>
        {active && renderSubItems(subItems)}
      </div>
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
    const { activeSubItem } = this.state;
    const { name, path } = subItem;
    const active = activeSubItem === subItem;
    const { setActiveSubItem } = this;

    const baseSubItemClass = 'site-sub-item';

    const baseSubItemItemClass = classnames(
      `${baseSubItemClass}__item`,
      { [`${baseSubItemClass}__item--active`]: active }
    );

    const baseSubItemLinkClass = classnames(
      `${baseSubItemClass}__link`,
      { [`${baseSubItemClass}__link--active`]: active }
    );

    return (
      <button
        key={`sub-item-${name}`}
        onClick={setActiveSubItem(subItem)}
        className={`${baseSubItemClass} button button--unstyled`}
      >
        {active && <div className={`${baseSubItemClass}__before`} />}
        <li
          key={name}
          className={baseSubItemItemClass}
        >
          <span to={path.location} className={baseSubItemLinkClass}>{name}</span>
        </li>
      </button>
    );
  }

  renderSubItems = (subItems) => {
    const { renderCollapseSubItems, renderSubItem, setSubNavClass } = this;
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
        {renderCollapseSubItems()}
      </div>
    );
  }

  renderCollapseSubItems = () => {
    const { toggleShowSubItems } = this;
    const { showSubItems } = this.state;
    const iconName = showSubItems ? 'kolidecon-chevronleftbold' : 'kolidecon-chevronrightbold';

    return (
      <button
        className="button button--unstyled collapse-sub-item"
        onClick={toggleShowSubItems(!showSubItems)}
      >
        <i className={iconName} />
      </button>
    );
  }

  render () {
    const { renderHeader, renderNavItems } = this;

    return (
      <nav className="site-nav">
        {renderHeader()}
        {renderNavItems()}
      </nav>
    );
  }
}

const ConnectedComponent = connect()(SiteNavSidePanel);
export default ConnectedComponent;
