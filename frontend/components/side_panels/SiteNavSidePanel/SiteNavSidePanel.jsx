import React, { Component, PropTypes } from 'react';
import { connect } from 'react-redux';
import { isEqual } from 'lodash';
import { push } from 'react-router-redux';
import classnames from 'classnames';

import { logoutUser } from 'redux/nodes/auth/actions';
import userInterface from 'interfaces/user';
import Icon from 'components/Icon';

import { activeTabFromPathname, activeSubTabFromPathname } from './helpers';
import navItems from './navItems';

class SiteNavSidePanel extends Component {
  static propTypes = {
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
      userMenuOpened: false,
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

  onLogout = (evt) => {
    evt.preventDefault();

    const { dispatch } = this.props;

    dispatch(logoutUser());

    return false;
  }

  setActiveSubItem = (activeSubItem) => {
    return (evt) => {
      evt.preventDefault();

      if (activeSubItem) {
        const { dispatch } = this.props;
        const { path: { location: tabLocation } } = activeSubItem;

        if (!tabLocation) return false;

        dispatch(push(tabLocation));
      }

      return false;
    };
  }

  setActiveTab = (activeTab) => {
    return (evt) => {
      evt.preventDefault();

      const { pathname, dispatch } = this.props;
      const activeSubItem = activeSubTabFromPathname(activeTab, pathname);

      this.setState({ activeTab, activeSubItem });

      const tabLocation = activeSubItem ? activeSubItem.path.location : activeTab.path.location;

      dispatch(push(tabLocation));

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

  toggleUserMenu = () => {
    const { userMenuOpened } = this.state;

    this.setState({ userMenuOpened: !userMenuOpened });
  }

  renderNavItem = (navItem) => {
    const { activeTab = {} } = this.state;
    const { icon, name, subItems } = navItem;
    const active = activeTab.name === name;
    const { renderSubItems, setActiveTab } = this;

    const navItemBaseClass = 'site-nav-item';

    const navItemClasses = classnames(
      `${navItemBaseClass}`,
      { [`${navItemBaseClass}--active`]: active }
    );

    return (
      <li className={navItemClasses} key={`nav-item-${name}`}>
        <button
          className={`${navItemBaseClass}__button button button--unstyled`}
          onClick={setActiveTab(navItem)}
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
    const { activeSubItem } = this.state;
    const { icon, name, path } = subItem;
    const active = activeSubItem === subItem;
    const { setActiveSubItem } = this;

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
          key={`sub-item-${name}`}
          onClick={setActiveSubItem(subItem)}
          className={`${baseSubItemClass}__button button button--unstyled`}
          to={path.location}
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

const ConnectedComponent = connect()(SiteNavSidePanel);
export default ConnectedComponent;
